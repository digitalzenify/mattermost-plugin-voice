package main

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// MessageWillBePosted auto-tags a post with voice message metadata when a user attaches a single
// supported audio file directly (e.g. via drag-and-drop or the regular attachment button) instead
// of using this plugin's recorder. This means any audio clip - not just ones recorded through the
// plugin's own UI - gets the nicer inline player and becomes eligible for automatic transcription.
func (p *Plugin) MessageWillBePosted(_ *plugin.Context, post *model.Post) (*model.Post, string) {
	if post == nil || post.Type != "" || len(post.FileIds) != 1 {
		return nil, ""
	}
	if _, alreadyTagged := post.Props[propsKeyVoiceMessage]; alreadyTagged {
		return nil, ""
	}

	fileInfo, appErr := p.API.GetFileInfo(post.FileIds[0])
	if appErr != nil || fileInfo == nil {
		return nil, ""
	}
	if !isSupportedAudioMimeType(fileInfo.MimeType) {
		return nil, ""
	}
	mimeType, _, _ := normalizeDeclaredMimeType(fileInfo.MimeType)

	updated := post.Clone()
	if updated.Props == nil {
		updated.Props = model.StringInterface{}
	}
	updated.Props[propsKeyVoiceMessage] = voiceMessageProps(fileInfo, mimeType, 0)

	return updated, ""
}

// MessageHasBeenPosted fires the optional outgoing transcription webhook, asynchronously, once a
// voice message post has been successfully created.
func (p *Plugin) MessageHasBeenPosted(_ *plugin.Context, post *model.Post) {
	if post == nil {
		return
	}
	if _, ok := post.Props[propsKeyVoiceMessage]; !ok {
		return
	}
	go p.sendTranscriptionWebhook(post)
}
