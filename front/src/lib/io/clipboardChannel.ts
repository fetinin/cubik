/**
 * Browser clipboard helper for opaque-text copy/paste.
 *
 * `copyToClipboard` writes via `navigator.clipboard.writeText`.
 * `readFromClipboardOrAsk` tries `navigator.clipboard.readText` and, if that
 * is unavailable or denied, falls back to a paste-from-modal flow where the
 * user pastes into a textarea.
 */

export type PromptPaste = () => Promise<string>;

export async function copyToClipboard(text: string): Promise<void> {
	if (!navigator.clipboard?.writeText) {
		throw new Error('Clipboard write API is not available');
	}
	await navigator.clipboard.writeText(text);
}

/**
 * Returns the clipboard text on success, or `null` if the read API is
 * unavailable, blocked by permissions, or otherwise rejects. We deliberately
 * collapse all failure modes to `null` so callers can decide on a fallback.
 */
export async function tryReadClipboard(): Promise<string | null> {
	if (!navigator.clipboard?.readText) {
		return null;
	}

	// Best-effort permission check. Older browsers/Firefox may not expose the
	// 'clipboard-read' descriptor — in that case we just attempt the read.
	try {
		const permissions = (
			navigator as Navigator & {
				permissions?: {
					query: (descriptor: { name: string }) => Promise<{ state: string }>;
				};
			}
		).permissions;
		if (permissions?.query) {
			const status = await permissions.query({ name: 'clipboard-read' });
			if (status.state === 'denied') {
				return null;
			}
		}
	} catch {
		// Permission descriptor not supported — fall through and attempt the read.
	}

	try {
		const text = await navigator.clipboard.readText();
		return typeof text === 'string' ? text : null;
	} catch {
		return null;
	}
}

export async function readFromClipboardOrAsk(
	promptPaste: PromptPaste = openPasteModal
): Promise<string> {
	const direct = await tryReadClipboard();
	if (direct !== null && direct !== '') {
		return direct;
	}
	return promptPaste();
}

/**
 * Default modal helper. Lazily imports the Svelte component and mounts it on
 * `document.body`. Resolves with the textarea contents on Submit, rejects on
 * Cancel. The dynamic import keeps the modal out of the eager bundle and
 * avoids running Svelte mount code at module load.
 */
async function openPasteModal(): Promise<string> {
	const { mount, unmount } = await import('svelte');
	const { default: PasteFallbackModal } = await import('./PasteFallbackModal.svelte');

	return new Promise<string>((resolve, reject) => {
		const target = document.createElement('div');
		document.body.appendChild(target);

		let component: ReturnType<typeof mount> | null = null;

		const cleanup = () => {
			if (component) {
				unmount(component);
				component = null;
			}
			target.remove();
		};

		component = mount(PasteFallbackModal, {
			target,
			props: {
				onsubmit: (text: string) => {
					cleanup();
					resolve(text);
				},
				oncancel: () => {
					cleanup();
					reject(new Error('Paste cancelled'));
				}
			}
		});
	});
}
