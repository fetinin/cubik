<script lang="ts">
	import { packedToCss, type PackedRGB } from '$lib/state/editor';
	import { onDestroy } from 'svelte';

	type Props = {
		width: number;
		height: number;
		pixels: PackedRGB[];
		paintColor: PackedRGB;
		onPaint: (index: number, color: PackedRGB) => void;
	};

	let { width, height, pixels, paintColor, onPaint }: Props = $props();

	let isPainting = $state(false);
	const rows = $derived([...Array(height).keys()]);
	const cols = $derived([...Array(width).keys()]);

	function paint(index: number) {
		onPaint(index, paintColor);
	}

	function onPointerUp() {
		isPainting = false;
	}

	if (typeof window !== 'undefined') {
		window.addEventListener('pointerup', onPointerUp);
	}

	onDestroy(() => {
		if (typeof window !== 'undefined') {
			window.removeEventListener('pointerup', onPointerUp);
		}
	});
</script>

<div class="inline-flex flex-col gap-1 flex-shrink-0" data-testid="matrix">
	{#each rows as y (y)}
		<div class="flex gap-1">
			{#each cols as x (x)}
				{@const i = y * width + x}
				<button
					type="button"
					class="h-7 w-7 rounded border border-gray-300"
					style:background-color={packedToCss(pixels[i] ?? 0)}
					onpointerdown={() => {
						isPainting = true;
						paint(i);
					}}
					onpointerenter={() => {
						if (isPainting) paint(i);
					}}
					aria-label={`pixel ${x},${y}`}
					data-testid={`matrix-cell-${i}`}
				></button>
			{/each}
		</div>
	{/each}
</div>
