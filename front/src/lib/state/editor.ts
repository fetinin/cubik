import { derived, get, writable, type Readable, type Writable } from 'svelte/store';
import type { AnimationPayload, Device, MatrixSize } from '$lib/api/mock';

export type PackedRGB = number; // 0xRRGGBB

export type Frame = {
	id: string;
	name: string;
	pixels: PackedRGB[]; // row-major, length = width*height
};

export type ApplyStatus =
	| { state: 'idle' }
	| { state: 'applying' }
	| { state: 'success' }
	| { state: 'error'; message: string };

export function packRGB(r: number, g: number, b: number): PackedRGB {
	return ((r & 0xff) << 16) | ((g & 0xff) << 8) | (b & 0xff);
}

export function unpackRGB(rgb: PackedRGB): { r: number; g: number; b: number } {
	return { r: (rgb >> 16) & 0xff, g: (rgb >> 8) & 0xff, b: rgb & 0xff };
}

export function packedToCss(rgb: PackedRGB): string {
	const { r, g, b } = unpackRGB(rgb);
	return `rgb(${r}, ${g}, ${b})`;
}

function makeId(prefix: string) {
	return `${prefix}-${Math.random().toString(16).slice(2)}-${Date.now().toString(16)}`;
}

export type EditorState = {
	// devices
	devices: Writable<Device[]>;
	selectedDeviceId: Writable<string | null>;

	// matrix + current drawing buffer
	matrix: Writable<MatrixSize | null>;
	pixels: Writable<PackedRGB[]>;

	// frames / animation
	frames: Writable<Frame[]>;
	selectedFrameId: Writable<string | null>;

	// derived helpers
	selectedDevice: Readable<Device | null>;
	selectedFrame: Readable<Frame | null>;

	// apply
	applyStatus: Writable<ApplyStatus>;
};

export function createEditorState(): EditorState {
	const devices = writable<Device[]>([]);
	const selectedDeviceId = writable<string | null>(null);
	const matrix = writable<MatrixSize | null>(null);
	const pixels = writable<PackedRGB[]>([]);
	const frames = writable<Frame[]>([]);
	const selectedFrameId = writable<string | null>(null);
	const applyStatus = writable<ApplyStatus>({ state: 'idle' });

	const selectedDevice = derived([devices, selectedDeviceId], ([$devices, $id]) => {
		if (!$id) return null;
		return $devices.find((d) => d.id === $id) ?? null;
	});

	const selectedFrame = derived([frames, selectedFrameId], ([$frames, $id]) => {
		if (!$id) return null;
		return $frames.find((f) => f.id === $id) ?? null;
	});

	return {
		devices,
		selectedDeviceId,
		matrix,
		pixels,
		frames,
		selectedFrameId,
		selectedDevice,
		selectedFrame,
		applyStatus
	};
}

export function initPixelsForSize(size: MatrixSize, fill: PackedRGB = 0x000000): PackedRGB[] {
	return Array.from({ length: size.width * size.height }, () => fill);
}

export function paintPixel(pixels: PackedRGB[], index: number, color: PackedRGB): PackedRGB[] {
	if (index < 0 || index >= pixels.length) return pixels;
	if (pixels[index] === color) return pixels;
	const next = pixels.slice();
	next[index] = color;
	return next;
}

export function createFrameFromPixels(name: string, pixels: PackedRGB[]): Frame {
	return { id: makeId('frame'), name, pixels: pixels.slice() };
}

export function loadFrameIntoPixels(state: EditorState, frameId: string) {
	const f = get(state.frames).find((x) => x.id === frameId);
	if (!f) return;
	state.selectedFrameId.set(frameId);
	state.pixels.set(f.pixels.slice());
}

export function replaceSelectedFramePixels(state: EditorState) {
	const frameId = get(state.selectedFrameId);
	if (!frameId) return;
	const currentPixels = get(state.pixels);
	state.frames.update((list) =>
		list.map((f) => (f.id === frameId ? { ...f, pixels: currentPixels.slice() } : f))
	);
}

export function buildAnimationPayload(size: MatrixSize, frames: Frame[]): AnimationPayload {
	return {
		fps: 1,
		width: size.width,
		height: size.height,
		frames: frames.map((f) => f.pixels.slice())
	};
}
