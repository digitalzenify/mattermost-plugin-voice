package main

import "testing"

func TestNormalizeDeclaredMimeType(t *testing.T) {
	cases := []struct {
		input    string
		wantMime string
		wantExt  string
		wantOK   bool
	}{
		{"audio/webm;codecs=opus", "audio/webm", ".webm", true},
		{"AUDIO/OGG", "audio/ogg", ".ogg", true},
		{"audio/x-wav", "audio/wav", ".wav", true},
		{"audio/mp4", "audio/mp4", ".m4a", true},
		{"video/mp4", "", "", false},
		{"", "", "", false},
	}

	for _, tc := range cases {
		mime, ext, ok := normalizeDeclaredMimeType(tc.input)
		if ok != tc.wantOK || mime != tc.wantMime || ext != tc.wantExt {
			t.Errorf("normalizeDeclaredMimeType(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tc.input, mime, ext, ok, tc.wantMime, tc.wantExt, tc.wantOK)
		}
	}
}

func TestSniffAudioMimeType(t *testing.T) {
	cases := []struct {
		name     string
		data     []byte
		wantMime string
		wantOK   bool
	}{
		{"webm", []byte{0x1a, 0x45, 0xdf, 0xa3, 0x00, 0x00}, "audio/webm", true},
		{"ogg", []byte("OggS0123456789"), "audio/ogg", true},
		{"mp4", append([]byte{0, 0, 0, 0x20}, []byte("ftypM4A ")...), "audio/mp4", true},
		{"mp3 with id3", []byte("ID3\x03\x00\x00\x00"), "audio/mpeg", true},
		{"mp3 frame sync", []byte{0xff, 0xfb, 0x90, 0x00}, "audio/mpeg", true},
		{"wav", []byte("RIFF\x24\x00\x00\x00WAVEfmt "), "audio/wav", true},
		{"unknown", []byte("not audio at all"), "", false},
		{"too short", []byte{0x01}, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mime, ok := sniffAudioMimeType(tc.data)
			if ok != tc.wantOK || mime != tc.wantMime {
				t.Errorf("sniffAudioMimeType(%q) = (%q, %v), want (%q, %v)", tc.data, mime, ok, tc.wantMime, tc.wantOK)
			}
		})
	}
}

func TestExtensionForMimeType(t *testing.T) {
	if got := extensionForMimeType("audio/webm"); got != ".webm" {
		t.Errorf("extensionForMimeType(audio/webm) = %q, want .webm", got)
	}
	if got := extensionForMimeType("audio/unknown"); got != "" {
		t.Errorf("extensionForMimeType(audio/unknown) = %q, want empty string", got)
	}
}

func TestIsSupportedAudioMimeType(t *testing.T) {
	if !isSupportedAudioMimeType("audio/mpeg") {
		t.Error("expected audio/mpeg to be supported")
	}
	if isSupportedAudioMimeType("image/png") {
		t.Error("expected image/png to be unsupported")
	}
}
