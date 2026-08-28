import React, {useRef} from 'react';
import {createPortal} from 'react-dom';

import {MicIcon, SendIcon, StopIcon, TrashIcon} from './icons';
import {VoiceAudioPlayer} from './voice_audio_player';

import {formatDuration} from '../audio/format';
import {useAnchoredPopoverPosition} from '../hooks/use_anchored_popover_position';
import {useVoiceRecorder, type VoiceRecorderDraft} from '../hooks/use_voice_recorder';

export type VoiceRecorderActionProps = {
    draft: VoiceRecorderDraft;
};

/**
 * The mic button always occupies the same small footprint in the message box's action row,
 * regardless of recording state. Everything else - the timer, the review player, upload/error
 * feedback - renders in a floating popover positioned above the button via a portal into
 * document.body, sized and clamped to the viewport in JS. This is deliberate: earlier versions
 * rendered that content inline in the action row, which on a narrow mobile viewport pushed into
 * (or covered) the Send button. A popover can never collide with the row it's anchored to.
 */
export function VoiceRecorderAction({draft}: VoiceRecorderActionProps) {
    const controller = useVoiceRecorder(draft);
    const anchorRef = useRef<HTMLButtonElement | null>(null);

    const popoverOpen = controller.status === 'recording' ||
        controller.status === 'review' ||
        controller.status === 'uploading' ||
        controller.status === 'error';
    const position = useAnchoredPopoverPosition(anchorRef, popoverOpen);

    let anchorLabel = 'Record voice message';
    let anchorTitle = 'Record voice message';
    let anchorDisabled = !controller.recordingSupported || !draft.channelId;
    let anchorOnClick = () => controller.startRecording();
    let anchorContent: React.ReactNode = <MicIcon/>;
    let anchorModifier = '';

    switch (controller.status) {
    case 'requesting-permission':
        anchorLabel = 'Requesting microphone permission';
        anchorTitle = 'Requesting microphone permission…';
        anchorDisabled = true;
        anchorModifier = 'VoiceRecorderAction__button--pending';
        break;
    case 'recording':
        anchorLabel = 'Recording in progress';
        anchorTitle = 'Recording…';
        anchorDisabled = true;
        anchorModifier = 'VoiceRecorderAction__button--recording';
        anchorContent = (
            <span
                className='VoiceRecorderAction__dot'
                aria-hidden='true'
            />
        );
        break;
    case 'review':
        anchorLabel = 'Reviewing recorded voice message';
        anchorTitle = 'Review before sending';
        anchorDisabled = true;
        anchorModifier = 'VoiceRecorderAction__button--active';
        break;
    case 'uploading':
        anchorLabel = 'Sending voice message';
        anchorTitle = 'Sending…';
        anchorDisabled = true;
        anchorContent = (
            <span
                className='VoiceRecorderAction__spinner'
                aria-hidden='true'
            />
        );
        break;
    case 'error':
        anchorLabel = 'Voice message failed, click to try again';
        anchorTitle = 'Try again';
        anchorDisabled = false;
        anchorModifier = 'VoiceRecorderAction__button--error';
        anchorOnClick = () => controller.startRecording();
        break;
    default:
        break;
    }

    let popoverContent: React.ReactNode = null;
    if (controller.status === 'recording') {
        const nearingLimit = controller.elapsedMs > (controller.maxDurationSecs * 1000) - 10000;
        popoverContent = (
            <div className='VoiceRecorderAction__panel VoiceRecorderAction__panel--recording'>
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
    } else if (controller.status === 'review' && controller.review) {
        popoverContent = (
            <div className='VoiceRecorderAction__panel VoiceRecorderAction__panel--review'>
                <VoiceAudioPlayer
                    src={controller.review.url}
                    durationHintMs={controller.review.durationMs}
                    variant='preview'
                />
                <div className='VoiceRecorderAction__panelActions'>
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
            </div>
        );
    } else if (controller.status === 'uploading') {
        popoverContent = (
            <div className='VoiceRecorderAction__panel VoiceRecorderAction__panel--uploading'>
                <span
                    className='VoiceRecorderAction__spinner'
                    aria-hidden='true'
                />
                <span>{'Sending…'}</span>
            </div>
        );
    } else if (controller.status === 'error') {
        popoverContent = (
            <div className='VoiceRecorderAction__panel VoiceRecorderAction__panel--error'>
                <span
                    className='VoiceRecorderAction__errorText'
                    role='alert'
                >{controller.error}</span>
            </div>
        );
    }

    return (
        <>
            <button
                ref={anchorRef}
                type='button'
                className={`VoiceRecorderAction__button ${anchorModifier}`}
                disabled={anchorDisabled}
                onClick={anchorOnClick}
                aria-label={anchorLabel}
                title={anchorTitle}
            >
                {anchorContent}
            </button>
            {position && popoverContent && createPortal(
                <div
                    className='VoiceRecorderAction__popover'
                    style={{
                        top: position.top,
                        left: position.left,
                        width: position.width,
                    }}
                >
                    {popoverContent}
                </div>,
                document.body,
            )}
        </>
    );
}
