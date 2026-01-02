<script lang="ts">
	import type { Frame } from '$lib/state/editor';

	type Props = {
		frames: Frame[];
		selectedFrameId: string | null;
		onSelectFrame: (frameId: string) => void;
		onSaveNewFrame: () => void;
		onReplaceSelected: () => void;
		onDeleteFrame: (frameId: string) => void;
		onMoveFrame: (frameId: string, dir: -1 | 1) => void;
	};

	let {
		frames,
		selectedFrameId,
		onSelectFrame,
		onSaveNewFrame,
		onReplaceSelected,
		onDeleteFrame,
		onMoveFrame
	}: Props = $props();

	const canReplace = () => selectedFrameId !== null;
</script>

<div class="flex flex-col gap-3">
	<div class="flex items-center justify-between gap-3">
		<div class="text-sm font-medium">Frames</div>
		<div class="flex items-center gap-2">
			<button
				type="button"
				class="rounded bg-gray-900 px-3 py-1.5 text-xs font-medium text-white"
				onclick={onSaveNewFrame}
				data-testid="save-frame"
			>
				Save frame
			</button>
			<button
				type="button"
				class="rounded border border-gray-300 px-3 py-1.5 text-xs font-medium disabled:opacity-50"
				onclick={onReplaceSelected}
				disabled={!canReplace()}
			>
				Replace selected
			</button>
		</div>
	</div>

	{#if frames.length === 0}
		<div class="text-sm text-gray-500">No frames yet. Draw something and click “Save frame”.</div>
	{:else}
		<ul class="flex flex-col gap-2">
			{#each frames as f, idx (f.id)}
				<li
					class="flex items-center justify-between gap-2 rounded border px-2 py-2"
					class:border-gray-300={selectedFrameId !== f.id}
					class:border-gray-900={selectedFrameId === f.id}
				>
					<button
						type="button"
						class="flex-1 text-left text-sm"
						onclick={() => onSelectFrame(f.id)}
						data-testid={`frame-${idx}`}
					>
						{f.name}
					</button>
					<div class="flex items-center gap-1">
						<button
							type="button"
							class="rounded border border-gray-300 px-2 py-1 text-xs disabled:opacity-50"
							onclick={() => onMoveFrame(f.id, -1)}
							disabled={idx === 0}
							aria-label="move up"
						>
							↑
						</button>
						<button
							type="button"
							class="rounded border border-gray-300 px-2 py-1 text-xs disabled:opacity-50"
							onclick={() => onMoveFrame(f.id, 1)}
							disabled={idx === frames.length - 1}
							aria-label="move down"
						>
							↓
						</button>
						<button
							type="button"
							class="rounded border border-red-300 px-2 py-1 text-xs text-red-700"
							onclick={() => onDeleteFrame(f.id)}
							aria-label="delete"
						>
							Delete
						</button>
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</div>
