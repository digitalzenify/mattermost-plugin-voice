package main

const (
	// propsKeyVoiceMessage marks a post as carrying a voice recording and holds the associated
	// metadata (file id, mime type, duration, size).
	propsKeyVoiceMessage = "voice_message"

	// propsKeyTranscript and propsKeyTranscriptAt hold the transcript text and timestamp set by
	// the incoming transcription callback, once an external service has processed the recording.
	propsKeyTranscript   = "voice_transcript"
	propsKeyTranscriptAt = "voice_transcript_at"

	// maxVoiceMessageBytes caps the size of an uploaded recording, independent of the configured
	// max duration, to bound memory use while handling the upload.
	maxVoiceMessageBytes = 25 * 1024 * 1024

	// multipartOverheadBytes accounts for form field overhead on top of the raw audio payload
	// when limiting the size of the incoming request body.
	multipartOverheadBytes = 1 * 1024 * 1024

	// durationToleranceMS allows the client-reported duration to exceed the configured max by a
	// small margin (e.g. clock drift between the recorder's timer and its own auto-stop) without
	// being rejected outright.
	durationToleranceMS = 5000
)
