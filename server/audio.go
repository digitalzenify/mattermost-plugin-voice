package main

import (
	"bytes"
	"strings"
)

// voiceAudioFormat describes one audio container/codec combination the plugin will accept for
// voice message uploads.
type voiceAudioFormat struct {
	mimeType  string
	extension string
}

var supportedVoiceAudioFormats = []voiceAudioFormat{
	{mimeType: "audio/webm", extension: ".webm"},
	{mimeType: "audio/ogg", extension: ".ogg"},
	{mimeType: "audio/mp4", extension: ".m4a"},
	{mimeType: "audio/mpeg", extension: ".mp3"},
	{mimeType: "audio/wav", extension: ".wav"},
}

// normalizeDeclaredMimeType strips any parameters (e.g. codec info) from a declared Content-Type
// and reports whether the resulting base type is one the plugin accepts.
func normalizeDeclaredMimeType(contentType string) (mimeType string, extension string, ok bool) {
	base := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if base == "audio/x-wav" {
		base = "audio/wav"
	}
	for _, format := range supportedVoiceAudioFormats {
		if format.mimeType == base {
			return format.mimeType, format.extension, true
		}
	}
	return "", "", false
}

// extensionForMimeType returns the canonical file extension for a supported mime type.
func extensionForMimeType(mimeType string) string {
	for _, format := range supportedVoiceAudioFormats {
		if format.mimeType == mimeType {
			return format.extension
		}
	}
	return ""
}

// sniffAudioMimeType inspects the leading bytes of audio data to determine its real container
// format, independent of any client-declared Content-Type. This guards against a mismatched or
// forged Content-Type header.
func sniffAudioMimeType(data []byte) (mimeType string, ok bool) {
	switch {
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}):
		// EBML header, used by both WebM and Matroska.
		return "audio/webm", true
	case len(data) >= 4 && bytes.Equal(data[:4], []byte("OggS")):
		return "audio/ogg", true
	case len(data) >= 12 && bytes.Equal(data[4:8], []byte("ftyp")):
		return "audio/mp4", true
	case len(data) >= 3 && bytes.Equal(data[:3], []byte("ID3")):
		return "audio/mpeg", true
	case len(data) >= 2 && data[0] == 0xff && data[1]&0xe0 == 0xe0:
		// MPEG audio frame sync without a leading ID3 tag.
		return "audio/mpeg", true
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WAVE")):
		return "audio/wav", true
	default:
		return "", false
	}
}

// isSupportedAudioMimeType reports whether the given mime type is one the plugin renders a voice
// player for.
func isSupportedAudioMimeType(mimeType string) bool {
	_, _, ok := normalizeDeclaredMimeType(mimeType)
	return ok
}
