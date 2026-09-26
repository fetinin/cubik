<script lang="ts">
	type Props = {
		onsubmit: (text: string) => void;
		oncancel: () => void;
	};

	const { onsubmit, oncancel }: Props = $props();

	let text = $state('');

	function handleSubmit() {
		onsubmit(text);
	}

	function handleCancel() {
		oncancel();
	}
</script>

<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
	onclick={handleCancel}
	onkeydown={(e) => e.key === 'Escape' && handleCancel()}
	role="presentation"
	data-testid="paste-fallback-backdrop"
>
	<div
		class="mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl"
		onclick={(e) => e.stopPropagation()}
		onkeydown={(e) => e.stopPropagation()}
		role="dialog"
		aria-modal="true"
		aria-labelledby="paste-fallback-title"
		tabindex="-1"
		data-testid="paste-fallback-content"
	>
		<h2 id="paste-fallback-title" class="mb-2 text-xl font-semibold">Paste from Clipboard</h2>
		<p class="mb-3 text-sm text-gray-600">
			Browser clipboard read is unavailable. Paste the text below and press Submit.
		</p>

		<textarea
			bind:value={text}
			placeholder="Paste here..."
			class="mb-4 h-40 w-full rounded border border-gray-300 px-3 py-2 font-mono text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none"
			data-testid="paste-fallback-textarea"></textarea>

		<div class="flex justify-end gap-2">
			<button
				onclick={handleCancel}
				class="rounded border border-gray-300 px-3 py-1.5 text-xs font-medium"
				data-testid="paste-fallback-cancel"
			>
				Cancel
			</button>
			<button
				onclick={handleSubmit}
				disabled={!text}
				class="rounded bg-blue-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50"
				data-testid="paste-fallback-submit"
			>
				Submit
			</button>
		</div>
	</div>
</div>
