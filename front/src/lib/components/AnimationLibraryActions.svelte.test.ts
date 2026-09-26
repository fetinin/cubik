import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { ResponseError, type SavedAnimation, type SparseAnimation } from '$lib/api/generated';

// Hoisted mock factories so the module-level mocks below can reach the spies.
const mocks = vi.hoisted(() => {
	class UploadCancelledStub extends Error {
		constructor() {
			super('upload-cancelled');
			this.name = 'UploadCancelled';
		}
	}
	return {
		downloadJsonFile: vi.fn<(name: string, json: string) => void>(),
		copyToClipboard: vi.fn<(text: string) => Promise<void>>(),
		readJsonFile: vi.fn<() => Promise<{ name: string; text: string }>>(),
		readFromClipboardOrAsk: vi.fn<() => Promise<string>>(),
		UploadCancelled: UploadCancelledStub
	};
});

vi.mock('$lib/io/fileChannel', () => ({
	downloadJsonFile: mocks.downloadJsonFile,
	readJsonFile: mocks.readJsonFile,
	UploadCancelled: mocks.UploadCancelled
}));

vi.mock('$lib/io/clipboardChannel', () => ({
	copyToClipboard: mocks.copyToClipboard,
	readFromClipboardOrAsk: mocks.readFromClipboardOrAsk
}));

// Imported after the vi.mock declarations so the component sees the mocked modules.
import AnimationLibraryActions from './AnimationLibraryActions.svelte';

function makeSavedAnimation(overrides: Partial<SavedAnimation> = {}): SavedAnimation {
	return {
		id: 'anim-1',
		deviceId: 'device-1',
		name: 'Wave',
		frames: [],
		createdAt: new Date('2026-01-01T00:00:00Z'),
		updatedAt: new Date('2026-01-01T00:00:00Z'),
		...overrides
	};
}

function makeSparseAnimation(overrides: Partial<SparseAnimation> = {}): SparseAnimation {
	return {
		version: '1.0',
		name: 'Wave',
		width: 20,
		height: 5,
		frames: [[{ x: 0, y: 0, c: 0xff0000 }]],
		...overrides
	};
}

function makeResponseError(status: number, body: unknown): ResponseError {
	const response = new Response(JSON.stringify(body), {
		status,
		headers: { 'content-type': 'application/json' }
	});
	return new ResponseError(response, `Response returned an error code`);
}

describe('AnimationLibraryActions', () => {
	beforeEach(() => {
		mocks.downloadJsonFile.mockReset();
		mocks.copyToClipboard.mockReset().mockResolvedValue(undefined);
		mocks.readJsonFile.mockReset();
		mocks.readFromClipboardOrAsk.mockReset();
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('Export button calls exportAnimation and triggers downloadJsonFile', async () => {
		const sparse = makeSparseAnimation({ name: 'Wave' });
		const stubApi = {
			exportAnimation: vi.fn(async () => sparse),
			importAnimation: vi.fn()
		};
		const screen = render(AnimationLibraryActions, {
			animations: [makeSavedAnimation({ id: 'anim-1', name: 'Wave' })],
			deviceId: 'device-1',
			api: stubApi
		});

		await screen.getByTestId('export-button').click();

		expect(stubApi.exportAnimation).toHaveBeenCalledWith({ id: 'anim-1' });
		expect(mocks.downloadJsonFile).toHaveBeenCalledTimes(1);
		const [filename, json] = mocks.downloadJsonFile.mock.calls[0];
		expect(filename).toBe('Wave');
		expect(typeof json).toBe('string');
		expect(JSON.parse(json as string)).toMatchObject({
			version: '1.0',
			name: 'Wave',
			width: 20,
			height: 5
		});
	});

	it('Copy button calls exportAnimation and copyToClipboard, flips affordance', async () => {
		const sparse = makeSparseAnimation({ name: 'Wave' });
		const stubApi = {
			exportAnimation: vi.fn(async () => sparse),
			importAnimation: vi.fn()
		};
		const screen = render(AnimationLibraryActions, {
			animations: [makeSavedAnimation({ id: 'anim-1', name: 'Wave' })],
			deviceId: 'device-1',
			api: stubApi
		});

		await screen.getByTestId('copy-button').click();

		expect(stubApi.exportAnimation).toHaveBeenCalledWith({ id: 'anim-1' });
		expect(mocks.copyToClipboard).toHaveBeenCalledTimes(1);
		const copiedJson = mocks.copyToClipboard.mock.calls[0][0] as string;
		expect(JSON.parse(copiedJson)).toMatchObject({ name: 'Wave', version: '1.0' });

		// Affordance flips to "Copied!"
		await expect.element(screen.getByTestId('copy-button')).toHaveTextContent('Copied!');
	});

	it('Import via file calls importAnimation with mode=rename and fires onrefresh on 200', async () => {
		const parsed = makeSparseAnimation({ name: 'Imported' });
		mocks.readJsonFile.mockResolvedValue({ name: 'wave.cubik.json', text: JSON.stringify(parsed) });

		const onrefresh = vi.fn();
		const stubApi = {
			exportAnimation: vi.fn(),
			importAnimation: vi.fn(async () => ({
				animation: {
					id: 'new-1',
					deviceId: 'device-1',
					name: 'Imported',
					frames: [],
					createdAt: new Date(),
					updatedAt: new Date()
				}
			}))
		};
		const screen = render(AnimationLibraryActions, {
			animations: [],
			deviceId: 'device-1',
			onrefresh,
			api: stubApi
		});

		await screen.getByTestId('import-button').click();
		await screen.getByTestId('import-from-file').click();

		// Wait for the import to complete by polling the success affordance.
		await expect.element(screen.getByTestId('import-status-success')).toBeInTheDocument();

		expect(stubApi.importAnimation).toHaveBeenCalledTimes(1);
		expect(stubApi.importAnimation).toHaveBeenCalledWith({
			importAnimationRequest: {
				deviceId: 'device-1',
				animation: parsed
			},
			mode: 'rename'
		});
		expect(onrefresh).toHaveBeenCalledTimes(1);
	});

	it('Import via clipboard calls importAnimation with mode=rename', async () => {
		const parsed = makeSparseAnimation({ name: 'FromClip' });
		mocks.readFromClipboardOrAsk.mockResolvedValue(JSON.stringify(parsed));

		const onrefresh = vi.fn();
		const stubApi = {
			exportAnimation: vi.fn(),
			importAnimation: vi.fn(async () => ({
				animation: {
					id: 'new-2',
					deviceId: 'device-1',
					name: 'FromClip',
					frames: [],
					createdAt: new Date(),
					updatedAt: new Date()
				}
			}))
		};
		const screen = render(AnimationLibraryActions, {
			animations: [],
			deviceId: 'device-1',
			onrefresh,
			api: stubApi
		});

		await screen.getByTestId('import-button').click();
		await screen.getByTestId('import-from-clipboard').click();

		await expect.element(screen.getByTestId('import-status-success')).toBeInTheDocument();

		expect(stubApi.importAnimation).toHaveBeenCalledWith({
			importAnimationRequest: {
				deviceId: 'device-1',
				animation: parsed
			},
			mode: 'rename'
		});
		expect(onrefresh).toHaveBeenCalledTimes(1);
	});

	it('shows ImportError field+reason inline on 400', async () => {
		const parsed = makeSparseAnimation();
		mocks.readFromClipboardOrAsk.mockResolvedValue(JSON.stringify(parsed));

		const stubApi = {
			exportAnimation: vi.fn(),
			importAnimation: vi.fn(async () => {
				throw makeResponseError(400, {
					field: 'frames[0].pixels[2].x',
					reason: 'x out of range'
				});
			})
		};
		const screen = render(AnimationLibraryActions, {
			animations: [],
			deviceId: 'device-1',
			api: stubApi
		});

		await screen.getByTestId('import-button').click();
		await screen.getByTestId('import-from-clipboard').click();

		await expect.element(screen.getByTestId('import-status-field')).toBeInTheDocument();
		await expect
			.element(screen.getByTestId('import-status-field'))
			.toHaveTextContent('frames[0].pixels[2].x');
		await expect
			.element(screen.getByTestId('import-status-reason'))
			.toHaveTextContent('x out of range');
	});

	it('opens conflict modal on 409 and retries with mode=overwrite when chosen', async () => {
		const parsed = makeSparseAnimation({ name: 'Wave' });
		mocks.readFromClipboardOrAsk.mockResolvedValue(JSON.stringify(parsed));

		let callCount = 0;
		const stubApi = {
			exportAnimation: vi.fn(),
			importAnimation: vi.fn(async () => {
				callCount += 1;
				if (callCount === 1) {
					throw makeResponseError(409, {
						existing_id: '550e8400-e29b-41d4-a716-446655440000',
						existing_name: 'Wave'
					});
				}
				return {
					animation: {
						id: 'new-3',
						deviceId: 'device-1',
						name: 'Wave',
						frames: [],
						createdAt: new Date(),
						updatedAt: new Date()
					}
				};
			})
		};
		const onrefresh = vi.fn();
		const screen = render(AnimationLibraryActions, {
			animations: [],
			deviceId: 'device-1',
			onrefresh,
			api: stubApi
		});

		await screen.getByTestId('import-button').click();
		await screen.getByTestId('import-from-clipboard').click();

		// Conflict modal renders with three buttons
		await expect.element(screen.getByTestId('import-conflict-content')).toBeInTheDocument();
		await expect.element(screen.getByTestId('import-conflict-rename')).toBeInTheDocument();
		await expect.element(screen.getByTestId('import-conflict-overwrite')).toBeInTheDocument();
		await expect.element(screen.getByTestId('import-conflict-cancel')).toBeInTheDocument();

		await screen.getByTestId('import-conflict-overwrite').click();

		await expect.element(screen.getByTestId('import-status-success')).toBeInTheDocument();

		expect(stubApi.importAnimation).toHaveBeenCalledTimes(2);
		const secondCall = stubApi.importAnimation.mock.calls[1] as unknown as [{ mode: string }];
		expect(secondCall[0]).toMatchObject({ mode: 'overwrite' });
		expect(onrefresh).toHaveBeenCalledTimes(1);
	});

	it('shows renamed-from banner when server renamed the import', async () => {
		const parsed = makeSparseAnimation({ name: 'Wave' });
		mocks.readFromClipboardOrAsk.mockResolvedValue(JSON.stringify(parsed));

		const stubApi = {
			exportAnimation: vi.fn(),
			importAnimation: vi.fn(async () => ({
				animation: {
					id: 'new-4',
					deviceId: 'device-1',
					name: 'Wave (2)',
					frames: [],
					createdAt: new Date(),
					updatedAt: new Date()
				},
				renamedFrom: 'Wave'
			}))
		};
		const screen = render(AnimationLibraryActions, {
			animations: [],
			deviceId: 'device-1',
			api: stubApi
		});

		await screen.getByTestId('import-button').click();
		await screen.getByTestId('import-from-clipboard').click();

		await expect.element(screen.getByTestId('import-status-success')).toHaveTextContent('Wave (2)');
		await expect.element(screen.getByTestId('import-status-success')).toHaveTextContent('Wave');
	});

	it('shows parse error and does not call importAnimation on bad JSON paste', async () => {
		mocks.readFromClipboardOrAsk.mockResolvedValue('not-json-{');

		const stubApi = {
			exportAnimation: vi.fn(),
			importAnimation: vi.fn()
		};
		const screen = render(AnimationLibraryActions, {
			animations: [],
			deviceId: 'device-1',
			api: stubApi
		});

		await screen.getByTestId('import-button').click();
		await screen.getByTestId('import-from-clipboard').click();

		await expect
			.element(screen.getByTestId('import-status-error'))
			.toHaveTextContent('not valid JSON');
		expect(stubApi.importAnimation).not.toHaveBeenCalled();
	});
});
