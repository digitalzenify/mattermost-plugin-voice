import React, {useEffect, useRef, useState} from 'react';

import {DownloadIcon, PauseIcon, PlayIcon} from './icons';

import {formatDuration} from '../audio/format';

export type VoiceAudioPlayerProps = {
    src: string;
    downloadUrl?: string;
    durationHintMs?: number;
    variant?: 'preview' | 'message';
};

export function VoiceAudioPlayer({src, downloadUrl, durationHintMs, variant = 'message'}: VoiceAudioPlayerProps) {
    const audioRef = useRef<HTMLAudioElement | null>(null);
    const [playing, setPlaying] = useState(false);
    const [currentMs, setCurrentMs] = useState(0);
    const [durationMs, setDurationMs] = useState(durationHintMs ?? 0);
    const [loadError, setLoadError] = useState(false);

    useEffect(() => {
        const audio = audioRef.current;
        if (!audio) {
            return undefined;
        }

        const onTimeUpdate = () => setCurrentMs(audio.currentTime * 1000);
        const onLoadedMetadata = () => {
            if (Number.isFinite(audio.duration) && audio.duration > 0) {
                setDurationMs(audio.duration * 1000);
            }
        };
        const onPlay = () => setPlaying(true);
        const onPauseOrEnd = () => setPlaying(false);
        const onError = () => setLoadError(true);

        audio.addEventListener('timeupdate', onTimeUpdate);
        audio.addEventListener('loadedmetadata', onLoadedMetadata);
        audio.addEventListener('play', onPlay);
        audio.addEventListener('pause', onPauseOrEnd);
        audio.addEventListener('ended', onPauseOrEnd);
        audio.addEventListener('error', onError);

        return () => {
            audio.removeEventListener('timeupdate', onTimeUpdate);
            audio.removeEventListener('loadedmetadata', onLoadedMetadata);
            audio.removeEventListener('play', onPlay);
            audio.removeEventListener('pause', onPauseOrEnd);
            audio.removeEventListener('ended', onPauseOrEnd);
            audio.removeEventListener('error', onError);
        };
    }, [src]);

    if (loadError) {
        return (
            <div className='VoiceAudioPlayer VoiceAudioPlayer--error'>
                {'Voice message unavailable.'}
            </div>
        );
    }

    const togglePlay = () => {
        const audio = audioRef.current;
        if (!audio) {
            return;
        }
        if (audio.paused) {
            audio.play().catch(() => setLoadError(true));
        } else {
            audio.pause();
        }
    };

    const onSeek = (event: React.ChangeEvent<HTMLInputElement>) => {
        const audio = audioRef.current;
        if (!audio || !durationMs) {
            return;
        }
        const fraction = Number(event.target.value) / 1000;
        audio.currentTime = fraction * (durationMs / 1000);
    };

    const progress = durationMs > 0 ? Math.min(1000, Math.round((currentMs / durationMs) * 1000)) : 0;
    const timeLabel = playing || currentMs > 0 ? formatDuration(currentMs) : formatDuration(durationMs);

    return (
        <div className={`VoiceAudioPlayer VoiceAudioPlayer--${variant}`}>
            <button
                type='button'
                className='VoiceAudioPlayer__playButton'
                onClick={togglePlay}
                aria-label={playing ? 'Pause voice message' : 'Play voice message'}
            >
                {playing ? <PauseIcon/> : <PlayIcon/>}
            </button>
            <input
                type='range'
                className='VoiceAudioPlayer__seek'
                min={0}
                max={1000}
                value={progress}
                onChange={onSeek}
                aria-label='Seek'
            />
            <span className='VoiceAudioPlayer__time'>{timeLabel}</span>
            {downloadUrl && (
                <a
                    className='VoiceAudioPlayer__download'
                    href={downloadUrl}
                    download={true}
                    aria-label='Download voice message'
                    title='Download'
                >
                    <DownloadIcon/>
                </a>
            )}
            <audio
                ref={audioRef}
                src={src}
                preload='metadata'
            />
        </div>
    );
}
