import type {FileInfo} from '@mattermost/types/files';
import type {Post} from '@mattermost/types/posts';

import manifest from './manifest';
import type {PluginConfig} from './types/voice_message';

function basePath(): string {
    return window.basename ?? '';
}

function pluginApiUrl(path: string): string {
    return `${basePath()}/plugins/${manifest.id}/api/v1${path}`;
}

/** The public file download URL for a file already attached to a post the caller can see. */
export function getFileDownloadUrl(fileId: string): string {
    return `${basePath()}/api/v4/files/${fileId}`;
}

const jsonHeaders: HeadersInit = {
    'X-Requested-With': 'XMLHttpRequest',
};

export async function fetchPluginConfig(): Promise<PluginConfig> {
    const response = await fetch(pluginApiUrl('/config'), {
        method: 'GET',
        credentials: 'same-origin',
        headers: jsonHeaders,
    });
    if (!response.ok) {
        throw new Error(`failed to load voice plugin config (${response.status})`);
    }
    const body = await response.json() as {max_duration_secs: number; audio_bits_per_second: number};
    return {
        maxDurationSecs: body.max_duration_secs,
        audioBitsPerSecond: body.audio_bits_per_second,
    };
}

export type UploadVoiceMessageInput = {
    channelId: string;
    rootId?: string;
    durationMs: number;
    blob: Blob;
    mimeType: string;
    filename: string;
    signal?: AbortSignal;
};

export type UploadVoiceMessageResponse = {
    post: Post;
    file_info: FileInfo;
};

export async function uploadVoiceMessage(input: UploadVoiceMessageInput): Promise<UploadVoiceMessageResponse> {
    const form = new FormData();
    form.append('audio', input.blob, input.filename);
    form.append('channel_id', input.channelId);
    if (input.rootId) {
        form.append('root_id', input.rootId);
    }
    form.append('duration_ms', String(Math.round(input.durationMs)));

    const response = await fetch(pluginApiUrl('/voice-messages'), {
        method: 'POST',
        credentials: 'same-origin',
        headers: jsonHeaders,
        body: form,
        signal: input.signal,
    });

    if (!response.ok) {
        const text = await response.text().catch(() => '');
        throw new Error(text || `failed to send voice message (${response.status})`);
    }

    return response.json() as Promise<UploadVoiceMessageResponse>;
}
