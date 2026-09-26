import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { downloadJsonFile, readJsonFile, UploadCancelled } from './fileChannel.ts';

type CapturedAnchor = {
	href: string;
	download: string;
	clicked: boolean;
};

/**
 * Hooks into `document.createElement('a')` to capture the anchor that
 * `downloadJsonFile` builds. Returns a getter for the most recent anchor
 * along with a restore function. We intercept `click()` to a no-op so the
 * test browser does not actually trigger a download dialog.
 */
function captureAnchor(): {
	getLast: () => CapturedAnchor | null;
	restore: () => void;
} {
	const realCreate = document.createElement.bind(document);
	let last: CapturedAnchor | null = null;

	const spy = vi
		.spyOn(document, 'createElement')
		.mockImplementation((tagName: string, options?: ElementCreationOptions) => {
			const el = realCreate(tagName, options);
			if (tagName.toLowerCase() === 'a') {
				const anchor = el as HTMLAnchorElement;
				const captured: CapturedAnchor = { href: '', download: '', clicked: false };
				last = captured;
				// Mirror property assignments into the captured record so that the
				// test can read them after the helper resolves and removes the node.
				Object.defineProperty(anchor, 'href', {
					get: () => captured.href,
					set: (v: string) => {
						captured.href = v;
					},
					configurable: true
				});
				Object.defineProperty(anchor, 'download', {
					get: () => captured.download,
					set: (v: string) => {
						captured.download = v;
					},
					configurable: true
				});
				anchor.click = () => {
					captured.clicked = true;
				};
			}
			return el;
		});

	return {
		getLast: () => last,
		restore: () => spy.mockRestore()
	};
}

describe('fileChannel.downloadJsonFile', () => {
	let createSpy: ReturnType<typeof vi.spyOn>;
	let revokeSpy: ReturnType<typeof vi.spyOn>;
	let blobs: Blob[];
	let urls: string[];
	let anchorCapture: ReturnType<typeof captureAnchor>;

	beforeEach(() => {
		blobs = [];
		urls = [];
		createSpy = vi.spyOn(URL, 'createObjectURL').mockImplementation((arg: Blob | MediaSource) => {
			blobs.push(arg as Blob);
			const url = `blob:fake/${urls.length}`;
			urls.push(url);
			return url;
		});
		revokeSpy = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined);
		anchorCapture = captureAnchor();
	});

	afterEach(() => {
		anchorCapture.restore();
		createSpy.mockRestore();
		revokeSpy.mockRestore();
	});

	it('builds an application/json blob, sets the filename, and revokes the URL', () => {
		downloadJsonFile('My Animation', '{"hello":"world"}');

		expect(createSpy).toHaveBeenCalledOnce();
		expect(blobs[0]).toBeInstanceOf(Blob);
		expect(blobs[0].type).toBe('application/json');

		const anchor = anchorCapture.getLast();
		expect(anchor).not.toBeNull();
		expect(anchor!.download).toBe('My Animation.cubik.json');
		expect(anchor!.href).toBe(urls[0]);
		expect(anchor!.clicked).toBe(true);

		expect(revokeSpy).toHaveBeenCalledWith(urls[0]);
	});

	it('sanitizes hostile names: strips path separators, control chars, and dot-traversal', () => {
		downloadJsonFile('../../etc/passwd', '{}');
		const a1 = anchorCapture.getLast();
		expect(a1).not.toBeNull();
		expect(a1!.download).not.toContain('..');
		expect(a1!.download).not.toContain('/');
		expect(a1!.download).not.toContain('\\');
		// eslint-disable-next-line no-control-regex
		expect(/[\x00-\x1f\x7f]/.test(a1!.download)).toBe(false);
		expect(a1!.download.endsWith('.cubik.json')).toBe(true);

		downloadJsonFile('\x00bad\x07name', '{}');
		const a2 = anchorCapture.getLast();
		expect(a2).not.toBeNull();
		// eslint-disable-next-line no-control-regex
		expect(/[\x00-\x1f\x7f]/.test(a2!.download)).toBe(false);
		expect(a2!.download.endsWith('.cubik.json')).toBe(true);
	});

	it('falls back to a default filename when the name is empty or whitespace', () => {
		downloadJsonFile('', '{}');
		const a1 = anchorCapture.getLast();
		expect(a1).not.toBeNull();
		expect(a1!.download).toBe('animation.cubik.json');

		downloadJsonFile('   ', '{}');
		const a2 = anchorCapture.getLast();
		expect(a2).not.toBeNull();
		expect(a2!.download).toBe('animation.cubik.json');
	});

	it('does not double-append the suffix when input already ends with .cubik.json', () => {
		downloadJsonFile('foo.cubik.json', '{}');
		const a = anchorCapture.getLast();
		expect(a).not.toBeNull();
		expect(a!.download).toBe('foo.cubik.json');
	});

	it('removes the temporary anchor from the DOM after invocation', () => {
		const before = document.body.querySelectorAll('a').length;
		downloadJsonFile('cleanup-test', '{}');
		const after = document.body.querySelectorAll('a').length;
		expect(after).toBe(before);
	});
});

/**
 * Stubbed FileReader that lets the test drive `onload` / `onerror` manually.
 * `readAsText` records the file and returns; the test fires the callback to
 * simulate completion.
 */
type StubReader = {
	readAsText: (file: Blob) => void;
	onload: ((this: FileReader, ev: ProgressEvent<FileReader>) => unknown) | null;
	onerror: ((this: FileReader, ev: ProgressEvent<FileReader>) => unknown) | null;
	result: string | ArrayBuffer | null;
	error: DOMException | null;
	__file?: Blob;
};

function installStubFileReader(): {
	getInstances: () => StubReader[];
	restore: () => void;
} {
	const original = window.FileReader;
	const instances: StubReader[] = [];

	class FakeFileReader implements StubReader {
		onload: ((this: FileReader, ev: ProgressEvent<FileReader>) => unknown) | null = null;
		onerror: ((this: FileReader, ev: ProgressEvent<FileReader>) => unknown) | null = null;
		result: string | ArrayBuffer | null = null;
		error: DOMException | null = null;
		__file?: Blob;

		readAsText(file: Blob) {
			this.__file = file;
		}
	}

	// Wrap so we record each constructed instance for the test to drive.
	const Wrapped = function (this: unknown) {
		const instance = new FakeFileReader();
		instances.push(instance);
		return instance;
	} as unknown as typeof window.FileReader;

	(window as unknown as { FileReader: typeof window.FileReader }).FileReader = Wrapped;

	return {
		getInstances: () => instances,
		restore: () => {
			(window as unknown as { FileReader: typeof window.FileReader }).FileReader = original;
		}
	};
}

/**
 * Hooks `<input>.click()` so we can simulate the file picker firing either a
 * `change` event (after seeding `input.files`) or a `cancel` event.
 */
function captureFileInput(): {
	getInput: () => HTMLInputElement | null;
	restore: () => void;
} {
	const realCreate = document.createElement.bind(document);
	let captured: HTMLInputElement | null = null;

	const spy = vi
		.spyOn(document, 'createElement')
		.mockImplementation((tagName: string, options?: ElementCreationOptions) => {
			const el = realCreate(tagName, options);
			if (tagName.toLowerCase() === 'input') {
				const input = el as HTMLInputElement;
				// Don't actually open the OS picker. The test will dispatch the event manually.
				input.click = () => {};
				captured = input;
			}
			return el;
		});

	return {
		getInput: () => captured,
		restore: () => spy.mockRestore()
	};
}

describe('fileChannel.readJsonFile', () => {
	let inputCapture: ReturnType<typeof captureFileInput>;
	let readerStub: ReturnType<typeof installStubFileReader>;

	beforeEach(() => {
		inputCapture = captureFileInput();
		readerStub = installStubFileReader();
	});

	afterEach(() => {
		inputCapture.restore();
		readerStub.restore();
	});

	it('resolves with { name, text } when a file is chosen', async () => {
		const promise = readJsonFile();

		const input = inputCapture.getInput();
		expect(input).not.toBeNull();

		const file = new File(['{"version":"1.0"}'], 'sample.cubik.json', {
			type: 'application/json'
		});
		Object.defineProperty(input!, 'files', {
			value: [file] as unknown as FileList,
			configurable: true
		});
		input!.dispatchEvent(new Event('change'));

		// Drive the stub FileReader's onload now that readAsText has been called.
		const readers = readerStub.getInstances();
		expect(readers.length).toBe(1);
		readers[0].result = '{"version":"1.0"}';
		readers[0].onload?.call(
			readers[0] as unknown as FileReader,
			new ProgressEvent('load') as ProgressEvent<FileReader>
		);

		await expect(promise).resolves.toEqual({
			name: 'sample.cubik.json',
			text: '{"version":"1.0"}'
		});

		// Input is removed from DOM after resolution.
		expect(input!.isConnected).toBe(false);
	});

	it('rejects with UploadCancelled on cancel event', async () => {
		const promise = readJsonFile();
		const input = inputCapture.getInput();
		expect(input).not.toBeNull();

		input!.dispatchEvent(new Event('cancel'));

		await expect(promise).rejects.toBeInstanceOf(UploadCancelled);
		expect(input!.isConnected).toBe(false);
	});

	it('rejects with UploadCancelled when change fires with no file selected', async () => {
		const promise = readJsonFile();
		const input = inputCapture.getInput();
		expect(input).not.toBeNull();

		Object.defineProperty(input!, 'files', {
			value: [] as unknown as FileList,
			configurable: true
		});
		input!.dispatchEvent(new Event('change'));

		await expect(promise).rejects.toBeInstanceOf(UploadCancelled);
		expect(input!.isConnected).toBe(false);
	});

	it('rejects when FileReader emits an error', async () => {
		const promise = readJsonFile();
		const input = inputCapture.getInput();
		expect(input).not.toBeNull();

		const file = new File(['x'], 'x.json', { type: 'application/json' });
		Object.defineProperty(input!, 'files', {
			value: [file] as unknown as FileList,
			configurable: true
		});
		input!.dispatchEvent(new Event('change'));

		const readers = readerStub.getInstances();
		expect(readers.length).toBe(1);
		readers[0].error = new DOMException('boom', 'NotReadableError');
		readers[0].onerror?.call(
			readers[0] as unknown as FileReader,
			new ProgressEvent('error') as ProgressEvent<FileReader>
		);

		await expect(promise).rejects.toThrow('boom');
		expect(input!.isConnected).toBe(false);
	});
});
