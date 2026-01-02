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

function sleep(ms: number) {
	return new Promise<void>((resolve) => setTimeout(resolve, ms));
}

export async function getDevices(): Promise<Device[]> {
	// Mocked backend call
	await sleep(200);

	return [
		{ id: 'mock-cube-1', name: 'CubeLite (Mock #1)' },
		{ id: 'mock-cube-2', name: 'CubeLite (Mock #2)' }
	];
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
