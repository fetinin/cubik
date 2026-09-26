/**
 * Browser file channel for opaque-text download/upload.
 *
 * `downloadJsonFile` triggers a save via Blob + temporary anchor click.
 * `readJsonFile` opens a file picker (`<input type=file>` + FileReader) and
 * resolves with the chosen file's name and decoded UTF-8 text.
 *
 * Both helpers operate on opaque text — they do not parse, validate, or
 * inspect the JSON shape. Callers handle that downstream.
 */

const FILE_SUFFIX = '.cubik.json';
const CONTENT_TYPE = 'application/json';
const DEFAULT_NAME = 'animation';
const ACCEPT_ATTR = 'application/json,.json,.cubik.json';

export class UploadCancelled extends Error {
	constructor() {
		super('upload-cancelled');
		this.name = 'UploadCancelled';
	}
}

/**
 * Sanitizes a user-provided animation name into a safe filename stem.
 * - Trims whitespace.
 * - Strips path separators, control chars, and other forbidden filesystem chars.
 * - Collapses leading dots (no `.hidden` files, no `..` traversal).
 * - Falls back to `DEFAULT_NAME` if the result is empty.
 */
function sanitizeFilename(name: string): string {
	const trimmed = (name ?? '').trim();
	// Replace forbidden chars (path separators, control chars 0x00-0x1F and 0x7F,
	// and Windows-reserved chars) with underscore.
	// eslint-disable-next-line no-control-regex
	const cleaned = trimmed.replace(/[\x00-\x1f\x7f/\\:*?"<>|]/g, '_');
	// Collapse runs of two or more consecutive dots into a single dot. This
	// removes the visual `..` traversal pattern that would otherwise survive
	// after path separators were replaced (e.g. `../../foo` -> `_.._foo`).
	const dotsCollapsed = cleaned.replace(/\.{2,}/g, '.');
	// Strip leading dots so we never produce '.hidden' files.
	const withoutLeadingDots = dotsCollapsed.replace(/^\.+/, '');
	const stem = withoutLeadingDots.trim();
	return stem === '' ? DEFAULT_NAME : stem;
}

/**
 * Builds a `.cubik.json` filename from a user-provided animation name.
 * Idempotent: if the input already ends with the suffix it is not duplicated.
 */
function buildFilename(name: string): string {
	const stem = sanitizeFilename(name);
	if (stem.toLowerCase().endsWith(FILE_SUFFIX)) {
		return stem;
	}
	return `${stem}${FILE_SUFFIX}`;
}

export function downloadJsonFile(name: string, json: string): void {
	if (typeof document === 'undefined') {
		throw new Error('downloadJsonFile requires a browser DOM');
	}

	const filename = buildFilename(name);
	const blob = new Blob([json], { type: CONTENT_TYPE });
	const url = URL.createObjectURL(blob);

	const anchor = document.createElement('a');
	try {
		anchor.href = url;
		anchor.download = filename;
		anchor.style.display = 'none';
		document.body.appendChild(anchor);
		anchor.click();
	} finally {
		anchor.remove();
		URL.revokeObjectURL(url);
	}
}

export async function readJsonFile(): Promise<{ name: string; text: string }> {
	if (typeof document === 'undefined') {
		throw new Error('readJsonFile requires a browser DOM');
	}

	const input = document.createElement('input');
	input.type = 'file';
	input.accept = ACCEPT_ATTR;
	input.style.display = 'none';
	document.body.appendChild(input);

	try {
		return await new Promise<{ name: string; text: string }>((resolve, reject) => {
			const onChange = () => {
				const file = input.files?.[0];
				if (!file) {
					reject(new UploadCancelled());
					return;
				}
				const reader = new FileReader();
				reader.onload = () => {
					const result = reader.result;
					if (typeof result !== 'string') {
						reject(new Error('FileReader produced non-string result'));
						return;
					}
					resolve({ name: file.name, text: result });
				};
				reader.onerror = () => {
					reject(reader.error ?? new Error('File read failed'));
				};
				reader.readAsText(file);
			};

			const onCancel = () => {
				reject(new UploadCancelled());
			};

			input.addEventListener('change', onChange, { once: true });
			input.addEventListener('cancel', onCancel, { once: true });
			input.click();
		});
	} finally {
		input.remove();
	}
}
