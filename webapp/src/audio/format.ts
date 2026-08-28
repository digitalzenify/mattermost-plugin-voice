// Preference order for the audio container/codec used when recording. Opus in WebM/Ogg gives the
// best size-to-quality ratio and is what Chrome, Firefox and Edge support; Safari (14.1+) only
// implements MediaRecorder for MP4/AAC, so it's kept as a fallback.
const candidateMimeTypes = [
    'audio/webm;codecs=opus',
    'audio/ogg;codecs=opus',
    'audio/webm',
    'audio/mp4',
];

/** Returns the best supported MediaRecorder mime type for this browser, or '' if none is. */
export function pickSupportedMimeType(): string {
    if (typeof MediaRecorder === 'undefined' || typeof MediaRecorder.isTypeSupported !== 'function') {
        return '';
    }
    return candidateMimeTypes.find((type) => MediaRecorder.isTypeSupported(type)) ?? '';
}

export function baseMimeType(mimeType: string): string {
    return mimeType.split(';')[0];
}

export function extensionForMimeType(mimeType: string): string {
    switch (baseMimeType(mimeType)) {
    case 'audio/ogg':
        return 'ogg';
    case 'audio/mp4':
        return 'm4a';
    case 'audio/mpeg':
        return 'mp3';
    case 'audio/wav':
        return 'wav';
    case 'audio/webm':
    default:
        return 'webm';
    }
}

/** Formats a millisecond duration as m:ss, matching the format the server uses in fallback text. */
export function formatDuration(ms: number): string {
    const totalSeconds = Math.max(0, Math.floor(ms / 1000));
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}
