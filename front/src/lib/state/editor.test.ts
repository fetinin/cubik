import { describe, it, expect } from 'vitest';
import { createFrameFromPixels, isEditorEmpty, sameFrames } from './editor';

describe('isEditorEmpty', () => {
	it('is empty when canvas and frames are all black', () => {
		expect(isEditorEmpty([0, 0, 0], [createFrameFromPixels('F1', [0, 0, 0])])).toBe(true);
		expect(isEditorEmpty([0, 0], [])).toBe(true);
	});

	it('is not empty when the canvas has a lit pixel', () => {
		expect(isEditorEmpty([0, 0xff0000, 0], [])).toBe(false);
	});

	it('is not empty when any frame has a lit pixel', () => {
		const frames = [createFrameFromPixels('F1', [0, 0]), createFrameFromPixels('F2', [0, 1])];
		expect(isEditorEmpty([0, 0], frames)).toBe(false);
	});
});

describe('sameFrames', () => {
	it('matches identical frames', () => {
		expect(sameFrames([[1, 2], [3]], [[1, 2], [3]])).toBe(true);
	});

	it('detects a changed pixel, frame count or order', () => {
		expect(sameFrames([[1, 2]], [[1, 5]])).toBe(false);
		expect(sameFrames([[1]], [[1], [2]])).toBe(false);
		expect(sameFrames([[1], [2]], [[2], [1]])).toBe(false);
	});
});
