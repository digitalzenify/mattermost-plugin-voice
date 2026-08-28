package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const commandTriggerVoice = "voice"

// Plugin implements the interface expected by the Mattermost server to communicate between the
// server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// router is the HTTP router used to serve the plugin's REST API.
	router *mux.Router

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// commandRegistered tracks whether the /voice slash command is currently registered, so
	// OnConfigurationChange can react to the setting being toggled at runtime.
	commandRegistered bool
}

// OnActivate is invoked when the plugin is activated. Mattermost itself already enforces
// min_server_version from plugin.json before this is ever called.
func (p *Plugin) OnActivate() error {
	p.router = p.initRouter()
	p.syncCommandRegistration()
	return nil
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	return nil
}

// syncCommandRegistration registers or unregisters the /voice slash command to match the current
// EnableSlashCommand setting.
func (p *Plugin) syncCommandRegistration() {
	shouldBeRegistered := p.getConfiguration().EnableSlashCommand

	if shouldBeRegistered && !p.commandRegistered {
		if err := p.API.RegisterCommand(&model.Command{
			Trigger:          commandTriggerVoice,
			AutoComplete:     true,
			AutoCompleteDesc: "Record and send a voice message",
			DisplayName:      "Record a voice message",
		}); err != nil {
			p.API.LogError("failed to register /voice command", "error", err.Error())
			return
		}
		p.commandRegistered = true
	} else if !shouldBeRegistered && p.commandRegistered {
		if err := p.API.UnregisterCommand("", commandTriggerVoice); err != nil {
			p.API.LogError("failed to unregister /voice command", "error", err.Error())
			return
		}
		p.commandRegistered = false
	}
}

// ExecuteCommand executes a command that has been previously registered via RegisterCommand. The
// recorder itself lives as a microphone button directly in the message box, so /voice exists as a
// discoverable pointer to it for anyone used to typing a slash command.
func (p *Plugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	trigger := strings.TrimPrefix(strings.Fields(args.Command)[0], "/")

	if trigger == commandTriggerVoice {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Click the microphone icon in the message box to record a voice message.",
		}, nil
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         fmt.Sprintf("Unknown command: %s", args.Command),
	}, nil
}
