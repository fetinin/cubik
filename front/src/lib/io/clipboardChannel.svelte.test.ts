import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { copyToClipboard, readFromClipboardOrAsk, tryReadClipboard } from './clipboardChannel.ts';
import PasteFallbackModal from './PasteFallbackModal.svelte';

type ClipboardLike = {
	writeText: (text: string) => Promise<void>;
	readText: () => Promise<string>;
};

function installFakeClipboard(initial = ''): ClipboardLike {
	let buffer = initial;
	const fake: ClipboardLike = {
		writeText: vi.fn(async (text: string) => {
			buffer = text;
		}),
		readText: vi.fn(async () => buffer)
	};
	Object.defineProperty(navigator, 'clipboard', {
		value: fake,
		configurable: true,
		writable: true
	});
	return fake;
}

function installFakePermissions(state: 'granted' | 'denied' | 'prompt' = 'granted') {
	Object.defineProperty(navigator, 'permissions', {
		value: {
			query: vi.fn(async () => ({ state }))
		},
		configurable: true,
		writable: true
	});
}

describe('clipboardChannel', () => {
	let originalClipboardDescriptor: PropertyDescriptor | undefined;
	let originalPermissionsDescriptor: PropertyDescriptor | undefined;

	beforeEach(() => {
		originalClipboardDescriptor = Object.getOwnPropertyDescriptor(navigator, 'clipboard');
		originalPermissionsDescriptor = Object.getOwnPropertyDescriptor(navigator, 'permissions');
	});

	afterEach(() => {
		if (originalClipboardDescriptor) {
			Object.defineProperty(navigator, 'clipboard', originalClipboardDescriptor);
		}
		if (originalPermissionsDescriptor) {
			Object.defineProperty(navigator, 'permissions', originalPermissionsDescriptor);
		}
		vi.restoreAllMocks();
	});

	it('copyToClipboard writes the given text via navigator.clipboard.writeText', async () => {
		const fake = installFakeClipboard();
		installFakePermissions('granted');

		await copyToClipboard('hello world');

		expect(fake.writeText).toHaveBeenCalledWith('hello world');
		expect(await fake.readText()).toBe('hello world');
	});

	it('readFromClipboardOrAsk returns text written to the clipboard', async () => {
		installFakeClipboard('payload-from-clipboard');
		installFakePermissions('granted');

		const result = await readFromClipboardOrAsk(async () => {
			throw new Error('promptPaste should not be invoked when readText succeeds');
		});

		expect(result).toBe('payload-from-clipboard');
	});

	it('falls back to promptPaste when readText throws (modal-fallback path)', async () => {
		const fake = installFakeClipboard();
		fake.readText = vi.fn(async () => {
			throw new Error('NotAllowedError: read denied');
		});
		installFakePermissions('granted');

		const promptPaste = vi.fn(async () => 'from textarea');

		const result = await readFromClipboardOrAsk(promptPaste);

		expect(result).toBe('from textarea');
		expect(promptPaste).toHaveBeenCalledOnce();
		expect(fake.readText).toHaveBeenCalledOnce();
	});

	it('tryReadClipboard returns null when permission is denied', async () => {
		installFakeClipboard('something');
		installFakePermissions('denied');

		const result = await tryReadClipboard();
		expect(result).toBeNull();
	});

	it('PasteFallbackModal collects text from textarea event and calls onsubmit', async () => {
		const onsubmit = vi.fn();
		const oncancel = vi.fn();

		const screen = render(PasteFallbackModal, { onsubmit, oncancel });

		const textarea = screen.getByTestId('paste-fallback-textarea');
		await textarea.fill('typed-text');

		const submitBtn = screen.getByTestId('paste-fallback-submit');
		await submitBtn.click();

		expect(onsubmit).toHaveBeenCalledWith('typed-text');
		expect(oncancel).not.toHaveBeenCalled();
	});

	it('PasteFallbackModal cancel button calls oncancel without submitting', async () => {
		const onsubmit = vi.fn();
		const oncancel = vi.fn();

		const screen = render(PasteFallbackModal, { onsubmit, oncancel });

		const cancelBtn = screen.getByTestId('paste-fallback-cancel');
		await cancelBtn.click();

		expect(oncancel).toHaveBeenCalledOnce();
		expect(onsubmit).not.toHaveBeenCalled();
	});
});
