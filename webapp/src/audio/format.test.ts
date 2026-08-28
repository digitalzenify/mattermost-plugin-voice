import {baseMimeType, extensionForMimeType, formatDuration} from './format';

describe('formatDuration', () => {
    it('formats zero and sub-minute durations', () => {
        expect(formatDuration(0)).toBe('0:00');
        expect(formatDuration(5000)).toBe('0:05');
        expect(formatDuration(59000)).toBe('0:59');
    });

    it('formats minute and hour-scale durations', () => {
        expect(formatDuration(65000)).toBe('1:05');
        expect(formatDuration(600000)).toBe('10:00');
    });

    it('clamps negative durations to zero', () => {
        expect(formatDuration(-500)).toBe('0:00');
    });
});

describe('baseMimeType', () => {
    it('strips codec parameters', () => {
        expect(baseMimeType('audio/webm;codecs=opus')).toBe('audio/webm');
        expect(baseMimeType('audio/wav')).toBe('audio/wav');
    });
});

describe('extensionForMimeType', () => {
    it('maps known types to their extension', () => {
        expect(extensionForMimeType('audio/webm;codecs=opus')).toBe('webm');
        expect(extensionForMimeType('audio/ogg')).toBe('ogg');
        expect(extensionForMimeType('audio/mp4')).toBe('m4a');
        expect(extensionForMimeType('audio/mpeg')).toBe('mp3');
        expect(extensionForMimeType('audio/wav')).toBe('wav');
    });

    it('falls back to webm for unknown types', () => {
        expect(extensionForMimeType('audio/mystery')).toBe('webm');
    });
});
