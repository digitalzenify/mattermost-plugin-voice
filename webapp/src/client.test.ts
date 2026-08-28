import {fetchPluginConfig, getFileDownloadUrl, uploadVoiceMessage} from './client';

describe('getFileDownloadUrl', () => {
    afterEach(() => {
        delete (window as {basename?: string}).basename;
    });

    it('builds a plain file URL with no basename', () => {
        expect(getFileDownloadUrl('file123')).toBe('/api/v4/files/file123');
    });

    it('respects a configured basename (subpath installs)', () => {
        window.basename = '/mattermost';
        expect(getFileDownloadUrl('file123')).toBe('/mattermost/api/v4/files/file123');
    });
});

describe('fetchPluginConfig', () => {
    afterEach(() => {
        jest.restoreAllMocks();
    });

    it('maps the server response to camelCase', async () => {
        global.fetch = jest.fn().mockResolvedValue({
            ok: true,
            json: () => Promise.resolve({max_duration_secs: 120, audio_bits_per_second: 64000}),
        }) as unknown as typeof fetch;

        const config = await fetchPluginConfig();

        expect(config).toEqual({maxDurationSecs: 120, audioBitsPerSecond: 64000});
        expect(global.fetch).toHaveBeenCalledWith(
            expect.stringContaining('/api/v1/config'),
            expect.objectContaining({credentials: 'same-origin'}),
        );
    });

    it('throws when the server responds with an error status', async () => {
        global.fetch = jest.fn().mockResolvedValue({ok: false, status: 500}) as unknown as typeof fetch;

        await expect(fetchPluginConfig()).rejects.toThrow('500');
    });
});

describe('uploadVoiceMessage', () => {
    afterEach(() => {
        jest.restoreAllMocks();
    });

    it('posts multipart form data and returns the parsed response', async () => {
        const fakeResponse = {post: {id: 'post1'}, file_info: {id: 'file1'}};
        global.fetch = jest.fn().mockResolvedValue({
            ok: true,
            json: () => Promise.resolve(fakeResponse),
        }) as unknown as typeof fetch;

        const result = await uploadVoiceMessage({
            channelId: 'channel1',
            durationMs: 1500,
            blob: new Blob(['abc']),
            mimeType: 'audio/webm',
            filename: 'voice-message.webm',
        });

        expect(result).toEqual(fakeResponse);
        const [, init] = (global.fetch as jest.Mock).mock.calls[0];
        expect(init.method).toBe('POST');
        expect(init.body).toBeInstanceOf(FormData);
    });

    it('throws with the response body text on failure', async () => {
        global.fetch = jest.fn().mockResolvedValue({
            ok: false,
            status: 413,
            text: () => Promise.resolve('voice message too large'),
        }) as unknown as typeof fetch;

        await expect(uploadVoiceMessage({
            channelId: 'channel1',
            durationMs: 1000,
            blob: new Blob(['abc']),
            mimeType: 'audio/webm',
            filename: 'voice-message.webm',
        })).rejects.toThrow('voice message too large');
    });
});
