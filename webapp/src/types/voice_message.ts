import type {Post} from '@mattermost/types/posts';

/** The shape of Props['voice_message'] set by the server on every voice message post. */
export type VoiceMessageProps = {
    version: number;
    file_id: string;
    mime_type: string;
    duration_ms: number;
    size: number;
};

/** Type guard around a post's props, used since props are loosely typed (Record<string, any>). */
export function getVoiceMessageProps(post: Post | undefined | null): VoiceMessageProps | null {
    if (!post) {
        return null;
    }
    const props = post.props?.voice_message as Partial<VoiceMessageProps> | undefined;
    if (!props || typeof props !== 'object' || typeof props.file_id !== 'string') {
        return null;
    }
    return props as VoiceMessageProps;
}

export function getVoiceTranscript(post: Post | undefined | null): string | null {
    const transcript = post?.props?.voice_transcript;
    return typeof transcript === 'string' && transcript.length > 0 ? transcript : null;
}

export type PluginConfig = {
    maxDurationSecs: number;
    audioBitsPerSecond: number;
};
