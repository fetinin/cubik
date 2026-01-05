export type Device = {
	id: string;
	name: string;
	location: string;
};

export type MatrixSize = {
	width: number;
	height: number;
};

export type AnimationPayload = {
	fps: 1;
	width: number;
	height: number;
	frames: number[][]; // each frame is packed RGB ints, row-major length = width*height
};

import { DefaultApi, Configuration } from '$lib/api/generated';
import type { RGBPixel } from '$lib/api/generated';

const api = new DefaultApi(
	new Configuration({
		basePath: 'http://localhost:8080'
	})
);

function sleep(ms: number) {
	return new Promise<void>((resolve) => setTimeout(resolve, ms));
}

export async function getDevices(): Promise<Device[]> {
	const response = await api.getDevices();
	const devices = response.devices;
	return devices.map((d) => ({ id: d.id, name: d.name, location: d.location }));
}

export async function getMatrixSize(_deviceId: string): Promise<MatrixSize> {
	// Mocked constant for now
	void _deviceId;
	await sleep(100);
	return { width: 20, height: 5 };
}

export async function applyAnimation(deviceId: string, payload: AnimationPayload): Promise<void> {
	const toPixel = (packed: number): RGBPixel => ({
		r: (packed >> 16) & 0xff,
		g: (packed >> 8) & 0xff,
		b: packed & 0xff
	});

	// NOTE: The backend expects device_location (yeelight://IP:PORT). We pass deviceId as the location here.
	await api.startAnimation({
		startAnimationRequest: {
			deviceLocation: deviceId,
			frames: payload.frames.map((frame) => frame.map(toPixel))
		}
	});
}

export async function stopAnimation(deviceId: string): Promise<void> {
	// NOTE: The backend expects device_location (yeelight://IP:PORT). We pass deviceId as the location here.
	await api.stopAnimation({
		stopAnimationRequest: {
			deviceLocation: deviceId
		}
	});
}

// Import SavedAnimation type for animation storage functions
import type { SavedAnimation } from '$lib/api/generated';

// Helper to convert API format to packed RGB
const fromPixel = (pixel: RGBPixel): number => (pixel.r << 16) | (pixel.g << 8) | pixel.b;

export async function saveAnimation(
	deviceId: string,
	name: string,
	frames: number[][]
): Promise<SavedAnimation> {
	const toPixel = (packed: number): RGBPixel => ({
		r: (packed >> 16) & 0xff,
		g: (packed >> 8) & 0xff,
		b: packed & 0xff
	});

	const response = await api.saveAnimation({
		saveAnimationRequest: {
			deviceId,
			name,
			frames: frames.map((frame) => frame.map(toPixel))
		}
	});
	return response.animation;
}

export async function listAnimations(deviceId: string): Promise<SavedAnimation[]> {
	const response = await api.listAnimations({ deviceId });
	return response.animations;
}

export async function loadAnimation(id: string): Promise<SavedAnimation> {
	const response = await api.getAnimation({ id });
	return response.animation;
}

export async function updateAnimation(
	id: string,
	name: string,
	frames: number[][]
): Promise<SavedAnimation> {
	const toPixel = (packed: number): RGBPixel => ({
		r: (packed >> 16) & 0xff,
		g: (packed >> 8) & 0xff,
		b: packed & 0xff
	});

	const response = await api.updateAnimation({
		id,
		updateAnimationRequest: {
			name,
			frames: frames.map((frame) => frame.map(toPixel))
		}
	});
	return response.animation;
}

export async function deleteAnimation(id: string): Promise<void> {
	await api.deleteAnimation({ id });
}
