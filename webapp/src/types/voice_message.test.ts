import type {Post} from '@mattermost/types/posts';

import {getVoiceMessageProps, getVoiceTranscript} from './voice_message';

function makePost(props: Record<string, unknown>): Post {
    return {props} as unknown as Post;
}

describe('getVoiceMessageProps', () => {
    it('returns null for a post with no voice_message prop', () => {
        expect(getVoiceMessageProps(makePost({}))).toBeNull();
    });

    it('returns null for null/undefined posts', () => {
        expect(getVoiceMessageProps(null)).toBeNull();
        expect(getVoiceMessageProps(undefined)).toBeNull();
    });

    it('returns null when voice_message is malformed', () => {
        expect(getVoiceMessageProps(makePost({voice_message: 'not an object'}))).toBeNull();
        expect(getVoiceMessageProps(makePost({voice_message: {}}))).toBeNull();
    });

    it('returns the props when well-formed', () => {
        const voiceMessage = {version: 1, file_id: 'abc123', mime_type: 'audio/webm', duration_ms: 4200, size: 1024};
        expect(getVoiceMessageProps(makePost({voice_message: voiceMessage}))).toEqual(voiceMessage);
    });
});

describe('getVoiceTranscript', () => {
    it('returns null when there is no transcript', () => {
        expect(getVoiceTranscript(makePost({}))).toBeNull();
        expect(getVoiceTranscript(makePost({voice_transcript: ''}))).toBeNull();
        expect(getVoiceTranscript(makePost({voice_transcript: 42}))).toBeNull();
    });

    it('returns the transcript text when present', () => {
        expect(getVoiceTranscript(makePost({voice_transcript: 'hello world'}))).toBe('hello world');
    });
});
