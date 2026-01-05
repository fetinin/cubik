<script lang="ts">
	import type { SavedAnimation } from '$lib/api/generated';

	type Props = {
		animations: SavedAnimation[];
		open: boolean;
		onload: (id: string) => void;
		ondelete: (id: string) => void;
	};

	let { animations, open = $bindable(), onload, ondelete }: Props = $props();

	function handleLoad(id: string) {
		onload(id);
		open = false;
	}

	function handleDelete(id: string) {
		ondelete(id);
	}
</script>

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		onclick={() => (open = false)}
		data-testid="load-modal-backdrop"
	>
		<div
			class="mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl"
			onclick={(e) => e.stopPropagation()}
			data-testid="load-modal-content"
		>
			<h2 class="mb-4 text-xl font-semibold">Load Animation</h2>

			{#if animations.length === 0}
				<p class="mb-4 text-sm text-gray-600">No saved animations for this device.</p>
			{:else}
				<ul class="mb-4 max-h-96 space-y-2 overflow-y-auto" data-testid="animations-list">
					{#each animations as anim (anim.id)}
						<li class="rounded border border-gray-200 p-3">
							<div class="flex items-center justify-between gap-3">
								<div class="flex-1">
									<div class="font-medium text-gray-900">{anim.name}</div>
									<div class="text-xs text-gray-600">{anim.frames.length} frames</div>
								</div>
								<div class="flex gap-2">
									<button
										onclick={() => handleLoad(anim.id)}
										class="rounded bg-blue-600 px-3 py-1.5 text-xs font-medium text-white"
										data-testid="load-button"
									>
										Load
									</button>
									<button
										onclick={() => handleDelete(anim.id)}
										class="rounded border border-red-300 px-2 py-1 text-xs text-red-700"
										data-testid="delete-button"
									>
										Delete
									</button>
								</div>
							</div>
						</li>
					{/each}
				</ul>
			{/if}

			<button
				onclick={() => (open = false)}
				class="w-full rounded border border-gray-300 px-3 py-1.5 text-xs font-medium"
				data-testid="close-button"
			>
				Close
			</button>
		</div>
	</div>
{/if}
