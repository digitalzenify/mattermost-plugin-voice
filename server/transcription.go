package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
)

// transcriptionWebhookPayload is the JSON body POSTed to the configured transcription webhook
// URL whenever a voice message is posted. It carries everything an external workflow (e.g. an
// n8n flow calling a speech-to-text API) needs in order to fetch the audio and post a transcript
// back, without needing a Mattermost bot account or personal access token: download_url is a
// short-lived, HMAC-signed link scoped to this one file.
type transcriptionWebhookPayload struct {
	PostID      string `json:"post_id"`
	RootID      string `json:"root_id,omitempty"`
	ChannelID   string `json:"channel_id"`
	TeamID      string `json:"team_id"`
	UserID      string `json:"user_id"`
	FileID      string `json:"file_id"`
	FileName    string `json:"file_name"`
	MimeType    string `json:"mime_type"`
	DurationMS  int64  `json:"duration_ms"`
	CreatedAt   int64  `json:"created_at"`
	DownloadURL string `json:"download_url"`
	DownloadExp int64  `json:"download_url_expires_at"`
	CallbackURL string `json:"callback_url"`
}

type transcriptionCallbackRequest struct {
	PostID string `json:"post_id"`
	Text   string `json:"text"`
}

// sendTranscriptionWebhook notifies the configured external service about a newly posted voice
// message. It is fire-and-forget: failures are logged but never surfaced to the user, since
// transcription is an optional, best-effort feature layered on top of a voice message that has
// already been posted successfully.
func (p *Plugin) sendTranscriptionWebhook(post *model.Post) {
	config := p.getConfiguration()
	if !config.transcriptionWebhookEnabled() {
		return
	}

	voiceProps, ok := post.Props[propsKeyVoiceMessage].(map[string]any)
	if !ok {
		return
	}
	fileID, _ := voiceProps["file_id"].(string)
	if fileID == "" {
		return
	}

	fileInfo, appErr := p.API.GetFileInfo(fileID)
	if appErr != nil || fileInfo == nil {
		p.API.LogWarn("skipping transcription webhook: could not load file info", "file_id", fileID, "error", appErr.Error())
		return
	}

	channel, appErr := p.API.GetChannel(post.ChannelId)
	if appErr != nil || channel == nil {
		p.API.LogWarn("skipping transcription webhook: could not load channel", "channel_id", post.ChannelId, "error", appErr.Error())
		return
	}

	mimeType, _ := voiceProps["mime_type"].(string)
	durationMS := int64FromProps(voiceProps["duration_ms"])

	exp := time.Now().Add(time.Duration(config.transcriptionLinkTTLSecs()) * time.Second).Unix()
	sig := signDownloadToken(config.TranscriptionWebhookSecret, fileID, exp)

	siteURL := p.siteURL()
	payload := transcriptionWebhookPayload{
		PostID:      post.Id,
		RootID:      post.RootId,
		ChannelID:   post.ChannelId,
		TeamID:      channel.TeamId,
		UserID:      post.UserId,
		FileID:      fileID,
		FileName:    fileInfo.Name,
		MimeType:    mimeType,
		DurationMS:  durationMS,
		CreatedAt:   post.CreateAt,
		DownloadURL: fmt.Sprintf("%s/plugins/%s/api/v1/files/%s?exp=%d&sig=%s", siteURL, manifest.Id, fileID, exp, sig),
		DownloadExp: exp,
		CallbackURL: fmt.Sprintf("%s/plugins/%s/api/v1/transcriptions", siteURL, manifest.Id),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		p.API.LogWarn("failed to marshal transcription webhook payload", "error", err.Error())
		return
	}

	req, err := http.NewRequest(http.MethodPost, config.TranscriptionWebhookURL, bytes.NewReader(body))
	if err != nil {
		p.API.LogWarn("failed to build transcription webhook request", "error", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Voice-Signature", "sha256="+hmacHex(config.TranscriptionWebhookSecret, body))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		p.API.LogWarn("transcription webhook request failed", "post_id", post.Id, "error", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		p.API.LogWarn("transcription webhook returned a non-success status", "post_id", post.Id, "status", resp.StatusCode)
	}
}

// handleTranscriptionCallback lets an external service report back the transcript for a voice
// message it downloaded via the signed link. Authentication is a shared bearer secret rather than
// a Mattermost session, since the caller is a server-to-server integration, not a logged-in user.
func (p *Plugin) handleTranscriptionCallback(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()
	if !config.transcriptionWebhookEnabled() {
		http.Error(w, "transcription webhook is not enabled", http.StatusNotFound)
		return
	}

	if !hasValidBearerSecret(r, config.TranscriptionWebhookSecret) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req transcriptionCallbackRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024)).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !model.IsValidId(req.PostID) {
		http.Error(w, "invalid post_id", http.StatusBadRequest)
		return
	}

	post, appErr := p.API.GetPost(req.PostID)
	if appErr != nil || post == nil {
		http.Error(w, "post not found", http.StatusNotFound)
		return
	}
	if _, ok := post.Props[propsKeyVoiceMessage]; !ok {
		http.Error(w, "post is not a voice message", http.StatusBadRequest)
		return
	}

	updated := post.Clone()
	if updated.Props == nil {
		updated.Props = model.StringInterface{}
	}
	updated.Props[propsKeyTranscript] = req.Text
	updated.Props[propsKeyTranscriptAt] = model.GetMillis()

	if _, appErr := p.API.UpdatePost(updated); appErr != nil {
		p.API.LogWarn("failed to save transcript", "post_id", req.PostID, "error", appErr.Error())
		http.Error(w, "failed to save transcript", appErrStatus(appErr))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
}

// handleSignedFileDownload serves a single file's bytes to a caller presenting a valid,
// unexpired signature, without requiring a Mattermost session. This lets an external
// transcription workflow fetch the audio it was told about via the webhook without needing a bot
// account or personal access token.
func (p *Plugin) handleSignedFileDownload(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()
	if !config.transcriptionWebhookEnabled() {
		http.NotFound(w, r)
		return
	}

	fileID := mux.Vars(r)["fileId"]
	if !model.IsValidId(fileID) {
		http.NotFound(w, r)
		return
	}

	expStr := r.URL.Query().Get("exp")
	sig := r.URL.Query().Get("sig")
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid or missing signature", http.StatusForbidden)
		return
	}
	if time.Now().Unix() > exp {
		http.Error(w, "download link has expired", http.StatusForbidden)
		return
	}
	if !hmac.Equal([]byte(sig), []byte(signDownloadToken(config.TranscriptionWebhookSecret, fileID, exp))) {
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}

	info, appErr := p.API.GetFileInfo(fileID)
	if appErr != nil || info == nil {
		http.NotFound(w, r)
		return
	}

	data, appErr := p.API.GetFile(fileID)
	if appErr != nil {
		http.NotFound(w, r)
		return
	}

	if info.MimeType != "" {
		w.Header().Set("Content-Type", info.MimeType)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", info.Name))
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
	w.Write(data) //nolint:errcheck
}

func signDownloadToken(secret, fileID string, exp int64) string {
	return hmacHex(secret, []byte(fmt.Sprintf("%s:%d", fileID, exp)))
}

func hmacHex(secret string, data []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data) //nolint:errcheck
	return hex.EncodeToString(mac.Sum(nil))
}

func hasValidBearerSecret(r *http.Request, secret string) bool {
	if secret == "" {
		return false
	}
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix {
		return false
	}
	provided := auth[len(prefix):]
	return subtle.ConstantTimeCompare([]byte(provided), []byte(secret)) == 1
}

func int64FromProps(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func (p *Plugin) siteURL() string {
	config := p.API.GetConfig()
	if config != nil && config.ServiceSettings.SiteURL != nil && *config.ServiceSettings.SiteURL != "" {
		return *config.ServiceSettings.SiteURL
	}
	return ""
}
