# Mattermost Voice Plugin

Record and send voice messages in Mattermost, straight from the message box, and have them
play back inline for anyone who receives them. Recordings are just regular Mattermost file
attachments under the hood, so they show up (and can be downloaded) in every client, including
mobile - even though recording itself is only available on web and desktop, since Mattermost
doesn't support third-party plugin UI in the native mobile apps.

The plugin also has an optional **transcription webhook**: when enabled, it notifies an external
service (an n8n workflow, for example) every time a voice message is posted, hands it a short-lived
signed link to the audio, and accepts a transcript back - which then appears inline under the
voice message for everyone in the channel. This is the intended way to have an AI agent transcribe
messages sent through this plugin. See [Automatic transcription](#automatic-transcription-eg-with-n8n) below.

## Usage

Click the microphone icon in the message box to start recording. You'll see a live level meter and
a running timer; stop recording to review the clip (with full playback) before sending, or discard
it and start over. Recording also works from thread replies. Typing `/voice` points you at the
microphone icon, for anyone used to the old slash-command workflow.

Voice messages posted by anyone - whether recorded with this plugin or simply attached as a regular
audio file - render with an inline player showing play/pause, a seek bar, elapsed/total time, and a
download link.

## Automatic transcription (e.g. with n8n)

This is off by default. To turn it on, go to **System Console > Plugins > Voice** and:

1. Enable **Outgoing Transcription Webhook**.
2. Set **Transcription Webhook URL** to the endpoint your workflow exposes (e.g. an n8n Webhook
   node's URL).
3. Save the settings page once to have Mattermost generate a **Transcription Webhook Secret** (or
   set your own).

From then on, every voice message triggers an HTTP POST to your webhook URL with a JSON body like:

```json
{
  "post_id": "n5tR...",
  "channel_id": "c8pQ...",
  "team_id": "t1zK...",
  "user_id": "u9wA...",
  "file_id": "f3mB...",
  "file_name": "a1b2c3.webm",
  "mime_type": "audio/webm",
  "duration_ms": 4210,
  "created_at": 1735689600000,
  "download_url": "https://your-server/plugins/com.mattermost.voice/api/v1/files/f3mB...?exp=...&sig=...",
  "download_url_expires_at": 1735690500,
  "callback_url": "https://your-server/plugins/com.mattermost.voice/api/v1/transcriptions"
}
```

The request body is signed with `X-Voice-Signature: sha256=<hmac>` (HMAC-SHA256 over the raw body,
using the webhook secret) if you want to verify it came from your Mattermost server.

`download_url` is a short-lived, single-file link scoped to just that recording - your workflow can
fetch it directly with a plain HTTP GET, with **no Mattermost bot account or personal access token
required**. Its lifetime is controlled by the **Signed Download Link Lifetime** setting.

Once you have a transcript, POST it back to `callback_url`:

```bash
curl -X POST "$CALLBACK_URL" \
  -H "Authorization: Bearer $TRANSCRIPTION_WEBHOOK_SECRET" \
  -H "Content-Type: application/json" \
  -d '{"post_id": "n5tR...", "text": "the transcribed text"}'
```

The plugin verifies the bearer secret, then attaches the transcript to the original post - it
appears inline under the voice player for everyone in the channel, live, via Mattermost's normal
post-update mechanism.

A minimal n8n flow is: **Webhook** (receives the payload above) → **HTTP Request** (GET
`download_url`) → your speech-to-text node of choice (OpenAI Whisper, etc.) → **HTTP Request** (POST
the transcript to `callback_url` with the bearer header above).

## Limitations

Recording requires a browser with `MediaRecorder` and microphone access, so it's only available on
the web app and desktop app - not the native mobile apps, which don't support plugin UI at all.
Voice messages themselves are ordinary file attachments, though, so mobile users can still see,
download, and (subject to Mattermost's own mobile audio-playback support) play them back.

## Installation

1. Download the latest release from the [releases page](https://github.com/digitalzenify/mattermost-plugin-voice/releases).
2. Upload it via **System Console > Plugins > Plugin Management**, or place it manually in the
   server's plugin directory. See the
   [Mattermost documentation](https://docs.mattermost.com/administration/plugins.html#set-up-guide)
   for details.

## Development

Requires Go 1.23+ and Node.js 22+ (see `webapp/.nvmrc`).

```bash
make dist          # build and bundle the plugin into dist/
make test          # run the server and webapp test suites
make check-style   # run linting and type checking
make watch         # rebuild and redeploy on every webapp change
```

To deploy to a local development server, set either `MM_SERVICESETTINGS_SITEURL` +
`MM_ADMIN_TOKEN`, or `MM_SERVICESETTINGS_SITEURL` + `MM_ADMIN_USERNAME` + `MM_ADMIN_PASSWORD`, then
run `make deploy`. If the server is running locally with local mode enabled, `make deploy` will use
that instead and no environment variables are needed.

For more on plugin development in general, see the
[Mattermost developer documentation](https://developers.mattermost.com/extend/plugins/).

## License

[MIT](LICENSE)
