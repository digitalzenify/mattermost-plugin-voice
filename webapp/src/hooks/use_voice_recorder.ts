import {useCallback, useEffect, useRef, useState} from 'react';

import {extensionForMimeType} from '../audio/format';
import {describeMicrophoneError, VoiceRecorder} from '../audio/recorder';
import {fetchPluginConfig, uploadVoiceMessage} from '../client';
import type {PluginConfig} from '../types/voice_message';

export type RecorderStatus = 'idle' | 'requesting-permission' | 'recording' | 'review' | 'uploading' | 'error';

export type ReviewClip = {
    blob: Blob;
    url: string;
    mimeType: string;
    durationMs: number;
};

export type VoiceRecorderDraft = {
    channelId?: string;
    rootId?: string;
};

const defaultConfig: PluginConfig = {maxDurationSecs: 300, audioBitsPerSecond: 64000};

export function useVoiceRecorder(draft: VoiceRecorderDraft) {
    const [status, setStatus] = useState<RecorderStatus>('idle');
    const [elapsedMs, setElapsedMs] = useState(0);
    const [level, setLevel] = useState(0);
    const [review, setReview] = useState<ReviewClip | null>(null);
    const [error, setError] = useState('');
    const [config, setConfig] = useState<PluginConfig>(defaultConfig);

    const recorderRef = useRef<VoiceRecorder | null>(null);
    const tickIntervalRef = useRef<number | null>(null);
    const reviewUrlRef = useRef<string | null>(null);

    useEffect(() => {
        let cancelled = false;
        fetchPluginConfig().then((loaded) => {
            if (!cancelled) {
                setConfig(loaded);
            }
        }).catch(() => { /* keep defaults */ });
        return () => {
            cancelled = true;
        };
    }, []);

    const stopTicking = useCallback(() => {
        if (tickIntervalRef.current !== null) {
            window.clearInterval(tickIntervalRef.current);
            tickIntervalRef.current = null;
        }
    }, []);

    const releaseReviewUrl = useCallback(() => {
        if (reviewUrlRef.current) {
            URL.revokeObjectURL(reviewUrlRef.current);
            reviewUrlRef.current = null;
        }
    }, []);

    const stopRecording = useCallback(async () => {
        const recorder = recorderRef.current;
        if (!recorder) {
            return;
        }
        stopTicking();
        try {
            const result = await recorder.stop();
            const url = URL.createObjectURL(result.blob);
            releaseReviewUrl();
            reviewUrlRef.current = url;
            setReview({blob: result.blob, url, mimeType: result.mimeType, durationMs: result.durationMs});
            setStatus('review');
        } catch {
            setStatus('error');
            setError('Could not finish recording. Please try again.');
        } finally {
            recorderRef.current = null;
        }
    }, [releaseReviewUrl, stopTicking]);

    const startRecording = useCallback(async () => {
        if (!draft.channelId) {
            return;
        }
        setError('');
        releaseReviewUrl();
        setReview(null);
        setElapsedMs(0);

        if (!VoiceRecorder.isSupported()) {
            setStatus('error');
            setError('Voice messages are not supported in this browser.');
            return;
        }

        setStatus('requesting-permission');
        const recorder = new VoiceRecorder(config.audioBitsPerSecond);
        try {
            await recorder.start((lvl) => setLevel(lvl));
        } catch (err) {
            setStatus('error');
            setError(describeMicrophoneError(err));
            return;
        }

        recorderRef.current = recorder;
        setStatus('recording');

        const maxDurationMs = config.maxDurationSecs * 1000;
        tickIntervalRef.current = window.setInterval(() => {
            const elapsed = recorder.elapsedMs;
            setElapsedMs(elapsed);
            if (elapsed >= maxDurationMs) {
                stopRecording();
            }
        }, 200);
    }, [config.audioBitsPerSecond, config.maxDurationSecs, draft.channelId, releaseReviewUrl, stopRecording]);

    const cancelRecording = useCallback(() => {
        stopTicking();
        recorderRef.current?.cancel();
        recorderRef.current = null;
        releaseReviewUrl();
        setReview(null);
        setElapsedMs(0);
        setLevel(0);
        setError('');
        setStatus('idle');
    }, [releaseReviewUrl, stopTicking]);

    const sendRecording = useCallback(async () => {
        if (!review || !draft.channelId) {
            return;
        }
        setStatus('uploading');
        try {
            await uploadVoiceMessage({
                channelId: draft.channelId,
                rootId: draft.rootId,
                durationMs: review.durationMs,
                blob: review.blob,
                mimeType: review.mimeType,
                filename: `voice-message.${extensionForMimeType(review.mimeType)}`,
            });
            releaseReviewUrl();
            setReview(null);
            setElapsedMs(0);
            setStatus('idle');
        } catch (err) {
            setStatus('error');
            setError(err instanceof Error ? err.message : 'Failed to send voice message.');
        }
    }, [draft.channelId, draft.rootId, releaseReviewUrl, review]);

    useEffect(() => {
        return () => {
            stopTicking();
            recorderRef.current?.cancel();
            releaseReviewUrl();
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    return {
        status,
        elapsedMs,
        level,
        review,
        error,
        maxDurationSecs: config.maxDurationSecs,
        recordingSupported: VoiceRecorder.isSupported(),
        startRecording,
        stopRecording,
        cancelRecording,
        sendRecording,
    };
}
