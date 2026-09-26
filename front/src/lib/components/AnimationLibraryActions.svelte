<script lang="ts">
	/**
	 * Export / Copy / Import affordances for the animation library.
	 *
	 * Per-row buttons:
	 *   - Export: GET /api/animation/{id}/export → downloads JSON via fileChannel.
	 *   - Copy:   same fetch, copies JSON to clipboard via clipboardChannel.
	 *
	 * Top-level Import button opens a small picker with two paths:
	 *   1. From file (fileChannel.readJsonFile)
	 *   2. From clipboard / paste (clipboardChannel.readFromClipboardOrAsk)
	 *
	 * On a successful import, calls `onrefresh` so the parent can re-fetch the
	 * library. Surfaces validation errors (ImportError field+reason), name
	 * collisions (NameConflict via a small modal), and other API errors inline.
	 */
	import {
		DefaultApi,
		Configuration,
		ImportAnimationOperationModeEnum,
		ResponseError,
		instanceOfSparseAnimation,
		SparseAnimationToJSON,
		type SavedAnimation,
		type SparseAnimation,
		type ImportAnimationResponse,
		type ImportError,
		type NameConflict
	} from '$lib/api/generated';
	import { copyToClipboard, readFromClipboardOrAsk } from '$lib/io/clipboardChannel';
	import { downloadJsonFile, readJsonFile, UploadCancelled } from '$lib/io/fileChannel';

	type ApiSurface = Pick<DefaultApi, 'exportAnimation' | 'importAnimation'>;

	type Props = {
		animations: SavedAnimation[];
		deviceId: string | null;
		onrefresh?: () => void | Promise<void>;
		// Test seam: lets callers swap the API client (e.g. with a stub).
		api?: ApiSurface;
	};

	let { animations, deviceId, onrefresh, api: apiOverride }: Props = $props();

	// Lazily construct the default client so that `$env/dynamic/public` is only
	// touched at runtime in the SvelteKit app context (and not during tests).
	let defaultApi: ApiSurface | null = null;
	async function getApi(): Promise<ApiSurface> {
		if (apiOverride) return apiOverride;
		if (!defaultApi) {
			const { env } = await import('$env/dynamic/public');
			defaultApi = new DefaultApi(new Configuration({ basePath: env.PUBLIC_API_BASE_PATH || '' }));
		}
		return defaultApi;
	}

	type ImportInfo = { kind: 'success'; message: string };
	type ImportFieldError = { kind: 'field-error'; field: string; reason: string };
	type GenericError = { kind: 'error'; message: string };
	type Status = ImportInfo | ImportFieldError | GenericError | null;

	let pickerOpen = $state(false);
	let busy = $state(false);
	let status = $state<Status>(null);

	// Conflict-resolution modal state
	let conflictOpen = $state(false);
	let conflictAnim = $state<SparseAnimation | null>(null);
	let conflictExistingName = $state<string>('');

	// Per-row UI feedback
	let copiedId = $state<string | null>(null);
	let rowError = $state<{ id: string; message: string } | null>(null);
	let copyTimer: ReturnType<typeof setTimeout> | null = null;

	function describeError(err: unknown): string {
		if (err instanceof Error && err.message) return err.message;
		return 'Unknown error';
	}

	function flagCopied(id: string) {
		copiedId = id;
		if (copyTimer) clearTimeout(copyTimer);
		copyTimer = setTimeout(() => {
			copiedId = null;
			copyTimer = null;
		}, 1500);
	}

	async function fetchExportJson(
		id: string
	): Promise<{ animation: SparseAnimation; json: string } | null> {
		try {
			const api = await getApi();
			const animation = await api.exportAnimation({ id });
			// Use generator's serializer so we emit canonical snake_case JSON.
			const json = JSON.stringify(SparseAnimationToJSON(animation), null, 2);
			return { animation, json };
		} catch (err) {
			let message = `Export failed: ${describeError(err)}`;
			if (err instanceof ResponseError) {
				if (err.response.status === 404) {
					message = 'Animation not found';
				} else {
					message = `Export failed: server returned ${err.response.status}`;
				}
			}
			rowError = { id, message };
			return null;
		}
	}

	async function handleExport(anim: SavedAnimation) {
		rowError = null;
		const result = await fetchExportJson(anim.id);
		if (!result) return;
		try {
			downloadJsonFile(result.animation.name || anim.name, result.json);
		} catch (err) {
			rowError = { id: anim.id, message: `Download failed: ${describeError(err)}` };
		}
	}

	async function handleCopy(anim: SavedAnimation) {
		rowError = null;
		const result = await fetchExportJson(anim.id);
		if (!result) return;
		try {
			await copyToClipboard(result.json);
			flagCopied(anim.id);
		} catch (err) {
			rowError = { id: anim.id, message: `Copy failed: ${describeError(err)}` };
		}
	}

	async function readImportError(err: ResponseError): Promise<ImportError | null> {
		try {
			const data = (await err.response.clone().json()) as Partial<ImportError>;
			if (typeof data?.field === 'string' && typeof data?.reason === 'string') {
				return { field: data.field, reason: data.reason };
			}
		} catch {
			// fall through
		}
		return null;
	}

	async function readNameConflict(err: ResponseError): Promise<NameConflict | null> {
		try {
			const data = (await err.response.clone().json()) as {
				existing_id?: string;
				existing_name?: string;
			};
			if (typeof data?.existing_id === 'string' && typeof data?.existing_name === 'string') {
				return { existingId: data.existing_id, existingName: data.existing_name };
			}
		} catch {
			// fall through
		}
		return null;
	}

	function parseSparseAnimation(text: string): SparseAnimation | string {
		let parsed: unknown;
		try {
			parsed = JSON.parse(text);
		} catch {
			return 'File is not valid JSON';
		}
		if (
			typeof parsed !== 'object' ||
			parsed === null ||
			!instanceOfSparseAnimation(parsed as object)
		) {
			return 'Not a Cubik animation file';
		}
		return parsed as SparseAnimation;
	}

	async function postImport(
		animation: SparseAnimation,
		mode: ImportAnimationOperationModeEnum
	): Promise<void> {
		if (!deviceId) {
			status = { kind: 'error', message: 'Select a device before importing' };
			return;
		}
		busy = true;
		try {
			const api = await getApi();
			const response: ImportAnimationResponse = await api.importAnimation({
				importAnimationRequest: { deviceId, animation },
				mode
			});
			const savedName = response.animation.name;
			if (response.renamedFrom) {
				status = {
					kind: 'success',
					message: `Imported as "${savedName}" — name "${response.renamedFrom}" was already taken.`
				};
			} else {
				status = { kind: 'success', message: `Imported "${savedName}".` };
			}
			conflictOpen = false;
			conflictAnim = null;
			if (onrefresh) await onrefresh();
		} catch (err) {
			if (err instanceof ResponseError) {
				const r = err.response;
				if (r.status === 400) {
					const info = await readImportError(err);
					if (info) {
						status = { kind: 'field-error', field: info.field, reason: info.reason };
						return;
					}
					status = { kind: 'error', message: 'Import failed: invalid animation' };
					return;
				}
				if (r.status === 409) {
					const info = await readNameConflict(err);
					conflictAnim = animation;
					conflictExistingName = info?.existingName ?? animation.name;
					conflictOpen = true;
					return;
				}
				if (r.status === 413) {
					status = { kind: 'error', message: 'Import failed: file too large' };
					return;
				}
				status = { kind: 'error', message: `Import failed: server returned ${r.status}` };
				return;
			}
			status = { kind: 'error', message: `Import failed: ${describeError(err)}` };
		} finally {
			busy = false;
		}
	}

	async function handleFileImport() {
		pickerOpen = false;
		status = null;
		let payload: { name: string; text: string };
		try {
			payload = await readJsonFile();
		} catch (err) {
			if (err instanceof UploadCancelled) return;
			status = { kind: 'error', message: `Could not read file: ${describeError(err)}` };
			return;
		}
		const parsed = parseSparseAnimation(payload.text);
		if (typeof parsed === 'string') {
			status = { kind: 'error', message: parsed };
			return;
		}
		await postImport(parsed, ImportAnimationOperationModeEnum.Rename);
	}

	async function handleClipboardImport() {
		pickerOpen = false;
		status = null;
		let text: string;
		try {
			text = await readFromClipboardOrAsk();
		} catch {
			// Paste cancelled or unavailable — silent.
			return;
		}
		const parsed = parseSparseAnimation(text);
		if (typeof parsed === 'string') {
			status = { kind: 'error', message: parsed };
			return;
		}
		await postImport(parsed, ImportAnimationOperationModeEnum.Rename);
	}

	async function retryWithMode(mode: ImportAnimationOperationModeEnum) {
		if (!conflictAnim) return;
		await postImport(conflictAnim, mode);
	}

	function cancelConflict() {
		conflictOpen = false;
		conflictAnim = null;
		status = { kind: 'error', message: 'Import cancelled.' };
	}
</script>

<div class="flex flex-col gap-3" data-testid="library-actions">
	<div class="flex items-center justify-between gap-2">
		<div class="text-sm font-medium">Library</div>
		<button
			type="button"
			class="rounded border border-gray-300 px-3 py-1.5 text-xs font-medium disabled:opacity-50"
			onclick={() => {
				pickerOpen = true;
				status = null;
			}}
			disabled={!deviceId || busy}
			data-testid="import-button"
		>
			Import Animation
		</button>
	</div>

	{#if animations.length > 0}
		<ul class="flex flex-col gap-1" data-testid="library-rows">
			{#each animations as anim (anim.id)}
				<li
					class="flex items-center justify-between gap-2 rounded border border-gray-200 px-2 py-1"
					data-testid="library-row"
				>
					<div class="flex-1 truncate text-xs text-gray-700">{anim.name}</div>
					<div class="flex items-center gap-1">
						<button
							type="button"
							onclick={() => handleExport(anim)}
							class="rounded border border-gray-300 px-2 py-1 text-xs"
							data-testid="export-button"
							disabled={busy}
						>
							Export
						</button>
						<button
							type="button"
							onclick={() => handleCopy(anim)}
							class="rounded border border-gray-300 px-2 py-1 text-xs"
							data-testid="copy-button"
							disabled={busy}
						>
							{copiedId === anim.id ? 'Copied!' : 'Copy'}
						</button>
					</div>
				</li>
				{#if rowError && rowError.id === anim.id}
					<li
						class="rounded border border-red-200 bg-red-50 px-2 py-1 text-xs text-red-700"
						data-testid="row-error"
					>
						{rowError.message}
					</li>
				{/if}
			{/each}
		</ul>
	{/if}

	{#if status}
		<div
			class="rounded border px-3 py-2 text-xs {status.kind === 'success'
				? 'border-green-200 bg-green-50 text-green-800'
				: 'border-red-200 bg-red-50 text-red-800'}"
			data-testid="import-status"
		>
			{#if status.kind === 'success'}
				<div data-testid="import-status-success">{status.message}</div>
			{:else if status.kind === 'field-error'}
				<div class="font-medium">Validation failed</div>
				<div class="mt-1" data-testid="import-status-field">
					<span class="font-mono">{status.field || '(root)'}</span>
					<span class="mx-1">—</span>
					<span data-testid="import-status-reason">{status.reason}</span>
				</div>
			{:else}
				<div data-testid="import-status-error">{status.message}</div>
			{/if}
		</div>
	{/if}
</div>

{#if pickerOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		onclick={() => (pickerOpen = false)}
		onkeydown={(e) => e.key === 'Escape' && (pickerOpen = false)}
		role="presentation"
		data-testid="import-picker-backdrop"
	>
		<div
			class="mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
			role="dialog"
			aria-modal="true"
			aria-labelledby="import-picker-title"
			tabindex="-1"
			data-testid="import-picker-content"
		>
			<h2 id="import-picker-title" class="mb-2 text-xl font-semibold">Import Animation</h2>
			<p class="mb-4 text-sm text-gray-600">
				Choose how to load the animation. The file or pasted text must be a Cubik animation JSON.
			</p>

			<div class="flex flex-col gap-2">
				<button
					type="button"
					onclick={handleFileImport}
					class="rounded bg-blue-600 px-3 py-2 text-xs font-medium text-white"
					data-testid="import-from-file"
				>
					From file…
				</button>
				<button
					type="button"
					onclick={handleClipboardImport}
					class="rounded border border-gray-300 px-3 py-2 text-xs font-medium"
					data-testid="import-from-clipboard"
				>
					From clipboard / paste…
				</button>
				<button
					type="button"
					onclick={() => (pickerOpen = false)}
					class="rounded border border-gray-300 px-3 py-1.5 text-xs font-medium"
					data-testid="import-picker-cancel"
				>
					Cancel
				</button>
			</div>
		</div>
	</div>
{/if}

{#if conflictOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		role="presentation"
		data-testid="import-conflict-backdrop"
	>
		<div
			class="mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl"
			role="dialog"
			aria-modal="true"
			aria-labelledby="import-conflict-title"
			tabindex="-1"
			data-testid="import-conflict-content"
		>
			<h2 id="import-conflict-title" class="mb-2 text-xl font-semibold">Name already in use</h2>
			<p class="mb-4 text-sm text-gray-600">
				An animation named <span class="font-mono">{conflictExistingName}</span> already exists for this
				device. How should the import be saved?
			</p>

			<div class="flex flex-wrap justify-end gap-2">
				<button
					type="button"
					onclick={() => retryWithMode(ImportAnimationOperationModeEnum.Rename)}
					disabled={busy}
					class="rounded bg-blue-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50"
					data-testid="import-conflict-rename"
				>
					Rename
				</button>
				<button
					type="button"
					onclick={() => retryWithMode(ImportAnimationOperationModeEnum.Overwrite)}
					disabled={busy}
					class="rounded bg-red-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50"
					data-testid="import-conflict-overwrite"
				>
					Overwrite
				</button>
				<button
					type="button"
					onclick={cancelConflict}
					disabled={busy}
					class="rounded border border-gray-300 px-3 py-1.5 text-xs font-medium disabled:opacity-50"
					data-testid="import-conflict-cancel"
				>
					Cancel
				</button>
			</div>
		</div>
	</div>
{/if}
