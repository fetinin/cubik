<script lang="ts">
	type Props = {
		open: boolean;
		currentAnimationName: string | null;
		onsave: (detail: { name: string; overwrite: boolean }) => void;
	};

	let { open = $bindable(), currentAnimationName, onsave }: Props = $props();

	let name = $state('');
	const showOverwriteOption = $derived(currentAnimationName !== null);

	// Pre-fill name when editing existing animation
	$effect(() => {
		if (open && currentAnimationName && !name) {
			name = currentAnimationName;
		}
	});

	// Reset name when modal closes
	$effect(() => {
		if (!open) {
			name = '';
		}
	});

	function handleSave(overwrite: boolean) {
		if (!name.trim()) return;
		onsave({ name: name.trim(), overwrite });
		name = '';
		open = false;
	}
</script>

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		onclick={() => (open = false)}
		data-testid="save-modal-backdrop"
	>
		<div
			class="mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl"
			onclick={(e) => e.stopPropagation()}
			data-testid="save-modal-content"
		>
			<h2 class="mb-4 text-xl font-semibold">Save Animation</h2>

			<label class="mb-4 block">
				<span class="mb-2 block text-sm font-medium text-gray-700">Animation Name:</span>
				<input
					type="text"
					bind:value={name}
					placeholder="Enter name..."
					class="w-full rounded border border-gray-300 px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:outline-none"
					data-testid="animation-name-input"
				/>
			</label>

			<div class="flex gap-2">
				{#if showOverwriteOption}
					<button
						onclick={() => handleSave(true)}
						disabled={!name.trim()}
						class="rounded bg-blue-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50"
						data-testid="update-button"
					>
						Update "{currentAnimationName}"
					</button>
				{/if}
				<button
					onclick={() => handleSave(false)}
					disabled={!name.trim()}
					class="rounded bg-blue-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50"
					data-testid="save-new-button"
				>
					Save as New
				</button>
				<button
					onclick={() => {
						open = false;
						name = '';
					}}
					class="rounded border border-gray-300 px-3 py-1.5 text-xs font-medium"
					data-testid="cancel-button"
				>
					Cancel
				</button>
			</div>
		</div>
	</div>
{/if}
