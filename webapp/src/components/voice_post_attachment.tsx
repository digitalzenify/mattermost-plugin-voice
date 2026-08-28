import React, {useLayoutEffect, useRef} from 'react';

import type {Post} from '@mattermost/types/posts';

import {VoiceAudioPlayer} from './voice_audio_player';

import {getFileDownloadUrl} from '../client';
import {getVoiceMessageProps, getVoiceTranscript} from '../types/voice_message';

export type VoicePostAttachmentProps = {
    post?: Post;
    postId?: string;
    onHeightChange?: (height: number) => void;
};

/**
 * Builds the component registered via registerPostMessageAttachmentComponent. Mattermost renders
 * this under the body of every post, so it must decide for itself - based on post props - whether
 * there's anything to show. Falling back to a store lookup by postId keeps it working regardless
 * of which prop shape a given Mattermost version passes.
 */
export function createVoicePostAttachmentComponent(getPostById: (postId: string) => Post | undefined) {
    return function VoicePostAttachment(props: VoicePostAttachmentProps) {
        const post = props.post ?? (props.postId ? getPostById(props.postId) : undefined);
        const voiceProps = getVoiceMessageProps(post);
        const transcript = getVoiceTranscript(post);
        const rootRef = useRef<HTMLDivElement | null>(null);

        const shouldRender = Boolean(post && post.delete_at === 0 && voiceProps);

        useLayoutEffect(() => {
            if (shouldRender) {
                props.onHeightChange?.(rootRef.current?.offsetHeight ?? 0);
            }
            // eslint-disable-next-line react-hooks/exhaustive-deps
        }, [shouldRender, post?.update_at]);

        if (!post || !voiceProps) {
            return null;
        }

        return (
            <div
                className='VoiceMessageAttachment'
                ref={rootRef}
            >
                <VoiceAudioPlayer
                    src={getFileDownloadUrl(voiceProps.file_id)}
                    downloadUrl={getFileDownloadUrl(voiceProps.file_id)}
                    durationHintMs={voiceProps.duration_ms}
                    variant='message'
                />
                {transcript && (
                    <div className='VoiceMessageAttachment__transcript'>
                        <span className='VoiceMessageAttachment__transcriptLabel'>{'Transcript'}</span>
                        <p>{transcript}</p>
                    </div>
                )}
            </div>
        );
    };
}
