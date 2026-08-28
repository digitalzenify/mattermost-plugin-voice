package main

import "testing"

func TestConfigurationMaxDurationSecs(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{"valid", 120, 120},
		{"zero falls back to default", 0, defaultVoiceMaxDurationSecs},
		{"negative falls back to default", -5, defaultVoiceMaxDurationSecs},
		{"too large falls back to default", 100000, defaultVoiceMaxDurationSecs},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &configuration{VoiceMaxDuration: tc.in}
			if got := c.maxDurationSecs(); got != tc.want {
				t.Errorf("maxDurationSecs() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestConfigurationAudioBitsPerSecond(t *testing.T) {
	cases := []struct {
		quality string
		want    int
	}{
		{"128", 128000},
		{"64", 64000},
		{"32", 32000},
		{"bogus", validVoiceAudioQualities[defaultVoiceAudioQuality]},
		{"", validVoiceAudioQualities[defaultVoiceAudioQuality]},
	}
	for _, tc := range cases {
		c := &configuration{VoiceAudioQuality: tc.quality}
		if got := c.audioBitsPerSecond(); got != tc.want {
			t.Errorf("audioBitsPerSecond() with quality %q = %d, want %d", tc.quality, got, tc.want)
		}
	}
}

func TestConfigurationTranscriptionWebhookEnabled(t *testing.T) {
	cases := []struct {
		name string
		c    configuration
		want bool
	}{
		{"fully configured", configuration{EnableTranscriptionWebhook: true, TranscriptionWebhookURL: "https://example.com", TranscriptionWebhookSecret: "secret"}, true},
		{"disabled", configuration{EnableTranscriptionWebhook: false, TranscriptionWebhookURL: "https://example.com", TranscriptionWebhookSecret: "secret"}, false},
		{"missing url", configuration{EnableTranscriptionWebhook: true, TranscriptionWebhookSecret: "secret"}, false},
		{"missing secret", configuration{EnableTranscriptionWebhook: true, TranscriptionWebhookURL: "https://example.com"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.c.transcriptionWebhookEnabled(); got != tc.want {
				t.Errorf("transcriptionWebhookEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestConfigurationIsValid(t *testing.T) {
	valid := configuration{EnableTranscriptionWebhook: true, TranscriptionWebhookURL: "https://example.com/hook", TranscriptionWebhookSecret: "secret"}
	if err := valid.IsValid(); err != nil {
		t.Errorf("expected valid configuration to pass, got error: %v", err)
	}

	missingSecret := configuration{EnableTranscriptionWebhook: true, TranscriptionWebhookURL: "https://example.com/hook"}
	if err := missingSecret.IsValid(); err == nil {
		t.Error("expected error for missing secret")
	}

	badURL := configuration{EnableTranscriptionWebhook: true, TranscriptionWebhookURL: "not-a-url", TranscriptionWebhookSecret: "secret"}
	if err := badURL.IsValid(); err == nil {
		t.Error("expected error for invalid URL scheme")
	}

	disabled := configuration{EnableTranscriptionWebhook: false}
	if err := disabled.IsValid(); err != nil {
		t.Errorf("expected disabled configuration to always be valid, got: %v", err)
	}
}
