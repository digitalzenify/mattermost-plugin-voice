package main

import (
	"net/url"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

const (
	defaultVoiceMaxDurationSecs = 300
	defaultVoiceAudioQuality    = "64"
	defaultTranscriptionLinkTTL = 900
	minTranscriptionLinkTTLSecs = 60
	maxTranscriptionLinkTTLSecs = 86400
	minVoiceMaxDurationSecs     = 5
	maxVoiceMaxDurationSecs     = 3600
)

var validVoiceAudioQualities = map[string]int{
	"128": 128000,
	"64":  64000,
	"32":  32000,
}

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes.
type configuration struct {
	EnableSlashCommand bool

	VoiceMaxDuration  int
	VoiceAudioQuality string

	EnableTranscriptionWebhook bool
	TranscriptionWebhookURL    string
	TranscriptionWebhookSecret string
	TranscriptionLinkTTL       int
}

// Clone shallow copies the configuration.
func (c *configuration) Clone() *configuration {
	clone := *c
	return &clone
}

// maxDurationSecs returns the effective, sanitized maximum recording duration in seconds.
func (c *configuration) maxDurationSecs() int {
	if c.VoiceMaxDuration < minVoiceMaxDurationSecs || c.VoiceMaxDuration > maxVoiceMaxDurationSecs {
		return defaultVoiceMaxDurationSecs
	}
	return c.VoiceMaxDuration
}

// audioBitsPerSecond returns the effective audio encoding bitrate, derived from the configured
// audio quality dropdown value.
func (c *configuration) audioBitsPerSecond() int {
	if bitrate, ok := validVoiceAudioQualities[c.VoiceAudioQuality]; ok {
		return bitrate
	}
	return validVoiceAudioQualities[defaultVoiceAudioQuality]
}

// transcriptionLinkTTLSecs returns the effective, sanitized lifetime of signed download links
// handed out to the outgoing transcription webhook.
func (c *configuration) transcriptionLinkTTLSecs() int {
	if c.TranscriptionLinkTTL < minTranscriptionLinkTTLSecs || c.TranscriptionLinkTTL > maxTranscriptionLinkTTLSecs {
		return defaultTranscriptionLinkTTL
	}
	return c.TranscriptionLinkTTL
}

// transcriptionWebhookEnabled reports whether the outgoing transcription webhook is fully and
// validly configured.
func (c *configuration) transcriptionWebhookEnabled() bool {
	return c.EnableTranscriptionWebhook &&
		strings.TrimSpace(c.TranscriptionWebhookURL) != "" &&
		strings.TrimSpace(c.TranscriptionWebhookSecret) != ""
}

// IsValid performs basic sanity checks on the configuration, returning an error for anything an
// administrator should fix.
func (c *configuration) IsValid() error {
	if c.EnableTranscriptionWebhook {
		if strings.TrimSpace(c.TranscriptionWebhookURL) == "" {
			return errors.New("transcription webhook is enabled but no webhook URL is configured")
		}
		parsed, err := url.Parse(c.TranscriptionWebhookURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return errors.New("transcription webhook URL must be a valid http(s) URL")
		}
		if strings.TrimSpace(c.TranscriptionWebhookSecret) == "" {
			return errors.New("transcription webhook is enabled but no shared secret is configured")
		}
	}
	return nil
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	configuration := new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	if err := configuration.IsValid(); err != nil {
		p.API.LogWarn("voice plugin configuration is invalid", "error", err.Error())
	}

	p.setConfiguration(configuration)

	if p.router != nil {
		p.syncCommandRegistration()
	}

	return nil
}
