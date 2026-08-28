import {pickSupportedMimeType} from './format';

export type RecordingResult = {
    blob: Blob;
    mimeType: string;
    durationMs: number;
};

/**
 * Wraps the browser's native MediaRecorder API to capture a microphone recording, plus a live
 * input-level meter (via an AnalyserNode) so the recording UI can show real feedback instead of
 * just a spinning duration counter.
 */
export class VoiceRecorder {
    private stream: MediaStream | null = null;
    private mediaRecorder: MediaRecorder | null = null;
    private audioContext: AudioContext | null = null;
    private analyser: AnalyserNode | null = null;
    private levelFrameId: number | null = null;
    private chunks: Blob[] = [];
    private startedAt = 0;
    private pausedAccumMs = 0;
    private pausedAt = 0;
    readonly mimeType: string;

    constructor(private readonly audioBitsPerSecond?: number) {
        this.mimeType = pickSupportedMimeType();
    }

    static isSupported(): boolean {
        return typeof MediaRecorder !== 'undefined' && Boolean(navigator.mediaDevices?.getUserMedia) && pickSupportedMimeType() !== '';
    }

    async start(onLevel?: (level: number) => void): Promise<void> {
        this.stream = await navigator.mediaDevices.getUserMedia({audio: true});

        if (onLevel) {
            this.startLevelMeter(this.stream, onLevel);
        }

        this.chunks = [];
        this.mediaRecorder = new MediaRecorder(this.stream, {
            mimeType: this.mimeType,
            audioBitsPerSecond: this.audioBitsPerSecond,
        });
        this.mediaRecorder.addEventListener('dataavailable', (event) => {
            if (event.data.size > 0) {
                this.chunks.push(event.data);
            }
        });
        this.mediaRecorder.start(250);

        this.startedAt = Date.now();
        this.pausedAccumMs = 0;
        this.pausedAt = 0;
    }

    pause(): void {
        if (this.mediaRecorder?.state === 'recording') {
            this.mediaRecorder.pause();
            this.pausedAt = Date.now();
        }
    }

    resume(): void {
        if (this.mediaRecorder?.state === 'paused') {
            this.mediaRecorder.resume();
            this.pausedAccumMs += Date.now() - this.pausedAt;
        }
    }

    get elapsedMs(): number {
        if (!this.startedAt) {
            return 0;
        }
        const activeEnd = this.mediaRecorder?.state === 'paused' ? this.pausedAt : Date.now();
        return activeEnd - this.startedAt - this.pausedAccumMs;
    }

    get state(): 'inactive' | 'recording' | 'paused' {
        return this.mediaRecorder?.state ?? 'inactive';
    }

    stop(): Promise<RecordingResult> {
        return new Promise((resolve, reject) => {
            const recorder = this.mediaRecorder;
            if (!recorder || recorder.state === 'inactive') {
                reject(new Error('recorder is not active'));
                return;
            }
            const durationMs = this.elapsedMs;
            recorder.addEventListener('stop', () => {
                const blob = new Blob(this.chunks, {type: this.mimeType});
                this.release();
                resolve({blob, mimeType: this.mimeType, durationMs});
            }, {once: true});
            recorder.stop();
        });
    }

    /** Stops recording and releases all resources without producing a result. */
    cancel(): void {
        if (this.mediaRecorder && this.mediaRecorder.state !== 'inactive') {
            this.mediaRecorder.stop();
        }
        this.release();
    }

    private startLevelMeter(stream: MediaStream, onLevel: (level: number) => void): void {
        const AudioContextCtor = window.AudioContext ?? (window as unknown as {webkitAudioContext?: typeof AudioContext}).webkitAudioContext;
        if (!AudioContextCtor) {
            return;
        }
        this.audioContext = new AudioContextCtor();
        const source = this.audioContext.createMediaStreamSource(stream);
        this.analyser = this.audioContext.createAnalyser();
        this.analyser.fftSize = 256;
        source.connect(this.analyser);

        const data = new Uint8Array(this.analyser.frequencyBinCount);
        const sample = () => {
            if (!this.analyser) {
                return;
            }
            this.analyser.getByteTimeDomainData(data);
            let sumSquares = 0;
            for (let i = 0; i < data.length; i++) {
                const normalized = (data[i] - 128) / 128;
                sumSquares += normalized * normalized;
            }
            const rms = Math.sqrt(sumSquares / data.length);
            onLevel(Math.min(1, rms * 4));
            this.levelFrameId = requestAnimationFrame(sample);
        };
        this.levelFrameId = requestAnimationFrame(sample);
    }

    private release(): void {
        if (this.levelFrameId !== null) {
            cancelAnimationFrame(this.levelFrameId);
            this.levelFrameId = null;
        }
        this.stream?.getTracks().forEach((track) => track.stop());
        this.stream = null;
        this.mediaRecorder = null;
        if (this.audioContext && this.audioContext.state !== 'closed') {
            this.audioContext.close().catch(() => { /* already closed */ });
        }
        this.audioContext = null;
        this.analyser = null;
    }
}

/** Maps a getUserMedia rejection to a short, user-facing explanation. */
export function describeMicrophoneError(err: unknown): string {
    const name = err instanceof DOMException ? err.name : '';
    switch (name) {
    case 'NotAllowedError':
    case 'SecurityError':
        return 'Microphone access was denied. Allow microphone access for this site and try again.';
    case 'NotFoundError':
    case 'OverconstrainedError':
        return 'No microphone was found. Connect a microphone and try again.';
    case 'NotReadableError':
        return 'The microphone is already in use by another application.';
    default:
        return 'Could not start recording. Please try again.';
    }
}
