package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/mattermost/mattermost/server/public/model"
)

type voiceMessageResponse struct {
	Post     *model.Post     `json:"post"`
	FileInfo *model.FileInfo `json:"file_info"`
}

type configResponse struct {
	MaxDurationSecs    int `json:"max_duration_secs"`
	AudioBitsPerSecond int `json:"audio_bits_per_second"`
}

// handleGetConfig exposes the subset of plugin configuration the webapp recorder needs in order
// to enforce the same limits the server will enforce.
func (p *Plugin) handleGetConfig(w http.ResponseWriter, _ *http.Request) {
	config := p.getConfiguration()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(configResponse{
		MaxDurationSecs:    config.maxDurationSecs(),
		AudioBitsPerSecond: config.audioBitsPerSecond(),
	}); err != nil {
		p.API.LogWarn("failed to encode config response", "error", err.Error())
	}
}

// handleCreateVoiceMessage accepts a multipart upload containing a recorded audio clip, uploads
// it as a normal Mattermost file attachment, and creates a standard (non-custom-typed) post
// referencing it. Keeping the post type empty and the file attached via the regular FileIds field
// ensures every Mattermost client - including mobile, which does not support plugin post-type or
// file-upload-method components - can still see and download the attachment.
func (p *Plugin) handleCreateVoiceMessage(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")

	r.Body = http.MaxBytesReader(w, r.Body, maxVoiceMessageBytes+multipartOverheadBytes)
	if err := r.ParseMultipartForm(multipartOverheadBytes); err != nil {
		http.Error(w, "voice message too large or malformed", http.StatusRequestEntityTooLarge)
		return
	}

	channelID := r.FormValue("channel_id")
	if !model.IsValidId(channelID) {
		http.Error(w, "invalid channel_id", http.StatusBadRequest)
		return
	}

	rootID := r.FormValue("root_id")
	if rootID != "" {
		if !model.IsValidId(rootID) {
			http.Error(w, "invalid root_id", http.StatusBadRequest)
			return
		}
		rootPost, appErr := p.API.GetPost(rootID)
		if appErr != nil || rootPost == nil || rootPost.ChannelId != channelID {
			http.Error(w, "invalid root_id", http.StatusBadRequest)
			return
		}
	}

	durationMS, err := parseDurationMS(r.FormValue("duration_ms"), p.getConfiguration().maxDurationSecs())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !p.API.HasPermissionToChannel(userID, channelID, model.PermissionCreatePost) {
		http.Error(w, "you do not have permission to post in this channel", http.StatusForbidden)
		return
	}
	if !p.API.HasPermissionToChannel(userID, channelID, model.PermissionUploadFile) {
		http.Error(w, "you do not have permission to upload files in this channel", http.StatusForbidden)
		return
	}

	data, mimeType, err := readAndValidateAudioPart(r, maxVoiceMessageBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
		return
	}

	filename := model.NewId() + extensionForMimeType(mimeType)
	fileInfo, appErr := p.API.UploadFile(data, channelID, filename)
	if appErr != nil {
		p.API.LogWarn("failed to upload voice message audio", "error", appErr.Error())
		http.Error(w, "failed to upload voice message", appErrStatus(appErr))
		return
	}

	post := &model.Post{
		UserId:    userID,
		ChannelId: channelID,
		RootId:    rootID,
		Message:   fmt.Sprintf(":microphone: Voice message (%s)", formatDuration(durationMS)),
		FileIds:   model.StringArray{fileInfo.Id},
		Props: model.StringInterface{
			propsKeyVoiceMessage: voiceMessageProps(fileInfo, mimeType, durationMS),
		},
	}

	createdPost, appErr := p.API.CreatePost(post)
	if appErr != nil {
		p.API.LogWarn("voice message file uploaded but post creation failed", "file_id", fileInfo.Id, "error", appErr.Error())
		http.Error(w, "failed to create voice message post", appErrStatus(appErr))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(voiceMessageResponse{Post: createdPost, FileInfo: fileInfo}); err != nil {
		p.API.LogWarn("failed to encode voice message response", "error", err.Error())
	}
}

func voiceMessageProps(fileInfo *model.FileInfo, mimeType string, durationMS int64) map[string]any {
	return map[string]any{
		"version":     1,
		"file_id":     fileInfo.Id,
		"mime_type":   mimeType,
		"duration_ms": durationMS,
		"size":        fileInfo.Size,
	}
}

func parseDurationMS(value string, maxDurationSecs int) (int64, error) {
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid duration_ms")
	}
	maxMS := int64(maxDurationSecs)*1000 + durationToleranceMS
	if parsed > maxMS {
		return 0, fmt.Errorf("recording exceeds the maximum allowed duration")
	}
	return parsed, nil
}

func readAndValidateAudioPart(r *http.Request, maxBytes int64) (data []byte, mimeType string, err error) {
	file, header, err := r.FormFile("audio")
	if err != nil {
		return nil, "", fmt.Errorf("missing audio file")
	}
	defer file.Close()

	data, err = io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read audio file")
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("empty audio file")
	}
	if int64(len(data)) > maxBytes {
		return nil, "", fmt.Errorf("voice message exceeds the maximum allowed size")
	}

	declaredMimeType := ""
	if header != nil {
		if base, _, ok := normalizeDeclaredMimeType(header.Header.Get("Content-Type")); ok {
			declaredMimeType = base
		}
	}

	sniffedMimeType, ok := sniffAudioMimeType(data)
	if !ok {
		return nil, "", fmt.Errorf("unsupported audio format")
	}
	if declaredMimeType != "" && declaredMimeType != sniffedMimeType {
		return nil, "", fmt.Errorf("audio content does not match its declared type")
	}

	return data, sniffedMimeType, nil
}

// formatDuration renders a millisecond duration as m:ss, matching the format used client-side.
func formatDuration(durationMS int64) string {
	totalSeconds := durationMS / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func appErrStatus(appErr *model.AppError) int {
	if appErr == nil || appErr.StatusCode == 0 {
		return http.StatusInternalServerError
	}
	return appErr.StatusCode
}
