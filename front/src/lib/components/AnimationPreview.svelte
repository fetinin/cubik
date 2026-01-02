<script lang="ts">
	import { onDestroy } from 'svelte';
	import { packedToCss, type PackedRGB } from '$lib/state/editor';

	type Props = {
		width: number;
		height: number;
		frames: PackedRGB[][];
	};

	let { width, height, frames }: Props = $props();

	let frameIndex = $state(0);
	let timer: number | null = null;

	$effect(() => {
		// reset when frames change
		frameIndex = 0;

		if (timer !== null) {
			window.clearInterval(timer);
			timer = null;
		}

		if (frames.length <= 1) return;

		// 1 FPS - cube limitation
		timer = window.setInterval(() => {
			frameIndex = (frameIndex + 1) % frames.length;
		}, 1000);

		return () => {
			if (timer !== null) {
				window.clearInterval(timer);
				timer = null;
			}
		};
	});

	onDestroy(() => {
		if (timer !== null) {
			window.clearInterval(timer);
			timer = null;
		}
	});

	const activeFrame = $derived(frames[frameIndex] ?? []);
</script>

<div class="flex flex-col gap-3">
	<div class="flex items-center justify-between gap-3">
		<div class="text-sm font-medium">Preview</div>
		<div class="text-xs text-gray-500">1 FPS</div>
	</div>

	{#if frames.length === 0}
		<div class="text-sm text-gray-500">Add frames to preview an animation.</div>
	{:else}
		<div
			class="grid gap-1"
			style:grid-template-columns={`repeat(${width}, 1fr)`}
			data-testid="preview"
		>
			{#each Array.from({ length: width * height }) as _cell, i (i)}
				<div
					class="h-4 w-4 rounded border border-gray-300"
					style:background-color={packedToCss(activeFrame[i] ?? 0)}
				></div>
			{/each}
		</div>
	{/if}
</div>
