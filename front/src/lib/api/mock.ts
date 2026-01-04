export type Device = {
	id: string;
	name: string;
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

function sleep(ms: number) {
	return new Promise<void>((resolve) => setTimeout(resolve, ms));
}

export async function getDevices(): Promise<Device[]> {
	const api = new DefaultApi(
		new Configuration({
			basePath: 'http://localhost:8080'
		})
	);

	const response = await api.getDevices();
	const devices = response.devices;
	return devices.map((d) => ({ id: d.id, name: d.name }));
}

export async function getMatrixSize(_deviceId: string): Promise<MatrixSize> {
	// Mocked constant for now
	void _deviceId;
	await sleep(100);
	return { width: 20, height: 5 };
}

export async function applyAnimation(deviceId: string, payload: AnimationPayload): Promise<void> {
	// Mocked backend call - later this becomes a real HTTP request
	// Keep a small delay to simulate network + device scheduling.
	await sleep(250);

	console.log('[mock] applyAnimation', { deviceId, payload });
}
