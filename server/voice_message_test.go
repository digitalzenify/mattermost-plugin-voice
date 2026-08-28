package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/mock"
)

func TestParseDurationMS(t *testing.T) {
	if got, err := parseDurationMS("", 300); err != nil || got != 0 {
		t.Errorf("empty duration: got (%d, %v), want (0, nil)", got, err)
	}
	if got, err := parseDurationMS("1500", 300); err != nil || got != 1500 {
		t.Errorf("valid duration: got (%d, %v), want (1500, nil)", got, err)
	}
	if _, err := parseDurationMS("not-a-number", 300); err == nil {
		t.Error("expected error for non-numeric duration")
	}
	if _, err := parseDurationMS("-1", 300); err == nil {
		t.Error("expected error for negative duration")
	}
	if _, err := parseDurationMS("999999999", 300); err == nil {
		t.Error("expected error for duration exceeding the max plus tolerance")
	}
	// Within tolerance of the configured max should be accepted.
	if _, err := parseDurationMS("304000", 300); err != nil {
		t.Errorf("expected duration within tolerance to be accepted, got: %v", err)
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		ms   int64
		want string
	}{
		{0, "0:00"},
		{5000, "0:05"},
		{65000, "1:05"},
		{600000, "10:00"},
	}
	for _, tc := range cases {
		if got := formatDuration(tc.ms); got != tc.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tc.ms, got, tc.want)
		}
	}
}

func buildVoiceMessageRequest(t *testing.T, channelID, rootID, durationMS string, audio []byte, contentType string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if channelID != "" {
		_ = writer.WriteField("channel_id", channelID)
	}
	if rootID != "" {
		_ = writer.WriteField("root_id", rootID)
	}
	if durationMS != "" {
		_ = writer.WriteField("duration_ms", durationMS)
	}

	if audio != nil {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", `form-data; name="audio"; filename="clip.webm"`)
		if contentType != "" {
			header.Set("Content-Type", contentType)
		}
		part, err := writer.CreatePart(header)
		if err != nil {
			t.Fatalf("failed to create audio part: %v", err)
		}
		if _, err := part.Write(audio); err != nil {
			t.Fatalf("failed to write audio part: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice-messages", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Mattermost-User-Id", model.NewId())
	return req
}

func TestHandleCreateVoiceMessage_Success(t *testing.T) {
	webmData := append([]byte{0x1a, 0x45, 0xdf, 0xa3}, bytes.Repeat([]byte{0x00}, 32)...)
	channelID := model.NewId()

	req := buildVoiceMessageRequest(t, channelID, "", "1500", webmData, "audio/webm")
	rec := httptest.NewRecorder()

	api := &plugintest.API{}
	api.On("HasPermissionToChannel", mock.Anything, channelID, model.PermissionCreatePost).Return(true)
	api.On("HasPermissionToChannel", mock.Anything, channelID, model.PermissionUploadFile).Return(true)
	api.On("UploadFile", mock.Anything, channelID, mock.Anything).Return(&model.FileInfo{Id: model.NewId(), Size: int64(len(webmData))}, nil)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(func(post *model.Post) *model.Post {
		post.Id = model.NewId()
		return post
	}, nil)

	p := &Plugin{}
	p.SetAPI(api)
	p.setConfiguration(&configuration{VoiceMaxDuration: 300})

	p.handleCreateVoiceMessage(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	api.AssertExpectations(t)
}

func TestHandleCreateVoiceMessage_RejectsInvalidChannel(t *testing.T) {
	req := buildVoiceMessageRequest(t, "not-a-valid-id", "", "", nil, "")
	rec := httptest.NewRecorder()

	p := &Plugin{}
	p.setConfiguration(&configuration{VoiceMaxDuration: 300})
	p.handleCreateVoiceMessage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleCreateVoiceMessage_RejectsMismatchedContentType(t *testing.T) {
	webmData := append([]byte{0x1a, 0x45, 0xdf, 0xa3}, bytes.Repeat([]byte{0x00}, 32)...)
	channelID := model.NewId()

	req := buildVoiceMessageRequest(t, channelID, "", "", webmData, "audio/mpeg")
	rec := httptest.NewRecorder()

	api := &plugintest.API{}
	api.On("HasPermissionToChannel", mock.Anything, channelID, model.PermissionCreatePost).Return(true)
	api.On("HasPermissionToChannel", mock.Anything, channelID, model.PermissionUploadFile).Return(true)

	p := &Plugin{}
	p.SetAPI(api)
	p.setConfiguration(&configuration{VoiceMaxDuration: 300})

	p.handleCreateVoiceMessage(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415 for declared/sniffed mime type mismatch, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCreateVoiceMessage_RejectsMissingPermission(t *testing.T) {
	webmData := append([]byte{0x1a, 0x45, 0xdf, 0xa3}, bytes.Repeat([]byte{0x00}, 32)...)
	channelID := model.NewId()

	req := buildVoiceMessageRequest(t, channelID, "", "", webmData, "audio/webm")
	rec := httptest.NewRecorder()

	api := &plugintest.API{}
	api.On("HasPermissionToChannel", mock.Anything, channelID, model.PermissionCreatePost).Return(false)

	p := &Plugin{}
	p.SetAPI(api)
	p.setConfiguration(&configuration{VoiceMaxDuration: 300})

	p.handleCreateVoiceMessage(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
