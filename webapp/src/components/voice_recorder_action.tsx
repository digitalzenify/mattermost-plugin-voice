import React from 'react';

import {MicIcon, SendIcon, StopIcon, TrashIcon} from './icons';
import {VoiceAudioPlayer} from './voice_audio_player';

import {formatDuration} from '../audio/format';
import {useVoiceRecorder, type VoiceRecorderDraft} from '../hooks/use_voice_recorder';

export type VoiceRecorderActionProps = {
    draft: VoiceRecorderDraft;
};

export function VoiceRecorderAction({draft}: VoiceRecorderActionProps) {
    const controller = useVoiceRecorder(draft);

    switch (controller.status) {
    case 'requesting-permission':
        return (
            <button
                type='button'
                className='VoiceRecorderAction__button VoiceRecorderAction__button--pending'
                disabled={true}
                aria-label='Requesting microphone permission'
                title='Requesting microphone permission…'
            >
                <MicIcon/>
            </button>
        );

    case 'recording': {
        const nearingLimit = controller.elapsedMs > (controller.maxDurationSecs * 1000) - 10000;
        return (
            <div className='VoiceRecorderAction VoiceRecorderAction--recording'>
                <span
                    className='VoiceRecorderAction__level'
                    style={{transform: `scale(${1 + (controller.level * 0.6)})`}}
                    aria-hidden='true'
                />
                <span className={`VoiceRecorderAction__timer ${nearingLimit ? 'VoiceRecorderAction__timer--warning' : ''}`}>
                    {formatDuration(controller.elapsedMs)}
                </span>
                <button
                    type='button'
                    className='VoiceRecorderAction__iconButton'
                    onClick={() => controller.stopRecording()}
                    aria-label='Stop recording'
                    title='Stop recording'
                >
                    <StopIcon/>
                </button>
                <button
                    type='button'
                    className='VoiceRecorderAction__iconButton VoiceRecorderAction__iconButton--cancel'
                    onClick={controller.cancelRecording}
                    aria-label='Cancel recording'
                    title='Cancel recording'
                >
                    <TrashIcon/>
                </button>
            </div>
        );
    }

    case 'review':
        return controller.review ? (
            <div className='VoiceRecorderAction VoiceRecorderAction--review'>
                <VoiceAudioPlayer
                    src={controller.review.url}
                    durationHintMs={controller.review.durationMs}
                    variant='preview'
                />
                <button
                    type='button'
                    className='VoiceRecorderAction__iconButton VoiceRecorderAction__iconButton--send'
                    onClick={() => controller.sendRecording()}
                    aria-label='Send voice message'
                    title='Send'
                >
                    <SendIcon/>
                </button>
                <button
                    type='button'
                    className='VoiceRecorderAction__iconButton VoiceRecorderAction__iconButton--cancel'
                    onClick={controller.cancelRecording}
                    aria-label='Discard recording'
                    title='Discard'
                >
                    <TrashIcon/>
                </button>
            </div>
        ) : null;

    case 'uploading':
        return (
            <div className='VoiceRecorderAction VoiceRecorderAction--uploading'>
                <span
                    className='VoiceRecorderAction__spinner'
                    aria-hidden='true'
                />
                <span>{'Sending…'}</span>
            </div>
        );

    case 'error':
        return (
            <div className='VoiceRecorderAction VoiceRecorderAction--error'>
                <span
                    className='VoiceRecorderAction__errorText'
                    role='alert'
                >{controller.error}</span>
                <button
                    type='button'
                    className='VoiceRecorderAction__button'
                    onClick={() => controller.startRecording()}
                    aria-label='Record voice message'
                    title='Try again'
                >
                    <MicIcon/>
                </button>
            </div>
        );

    case 'idle':
    default:
        return (
            <button
                type='button'
                className='VoiceRecorderAction__button'
                disabled={!controller.recordingSupported || !draft.channelId}
                onClick={() => controller.startRecording()}
                aria-label='Record voice message'
                title='Record voice message'
            >
                <MicIcon/>
            </button>
        );
    }
}
