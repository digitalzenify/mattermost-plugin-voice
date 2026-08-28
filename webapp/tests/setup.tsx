import '@testing-library/jest-dom';

// jsdom does not implement the Web Audio / MediaRecorder APIs the recorder relies on; individual
// test files stub out just what they need on top of this.
export {};
