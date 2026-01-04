<script lang="ts">
	import { SvelteMap } from 'svelte/reactivity';
	import type { PackedRGB } from '$lib/state/editor';
	import { packedToCss } from '$lib/state/editor';

	type Props = {
		pixels: PackedRGB[];
		selectedColor: PackedRGB;
		onSelectColor: (color: PackedRGB) => void;
	};

	let { pixels, selectedColor, onSelectColor }: Props = $props();

	// Extract unique colors from pixel buffer
	const uniqueColors = $derived(() => {
		// Deduplicate using Set
		const uniqueSet = new Set<PackedRGB>(pixels);
		const colors = Array.from(uniqueSet);

		// Sort by first appearance
		const indexMap = new SvelteMap<PackedRGB, number>();
		pixels.forEach((color, idx) => {
			if (!indexMap.has(color)) indexMap.set(color, idx);
		});

		const sorted = colors.sort((a, b) => {
			const idxA = indexMap.get(a) ?? Infinity;
			const idxB = indexMap.get(b) ?? Infinity;
			return idxA - idxB;
		});

		// Limit to 12 colors
		return sorted.slice(0, 12);
	});

	// Convert packed RGB to hex string for display
	function colorToHex(color: PackedRGB): string {
		return '#' + color.toString(16).padStart(6, '0').toUpperCase();
	}
</script>

<div class="flex flex-col gap-3">
	<div class="text-sm font-medium">Palette</div>

	{#if uniqueColors().length === 0}
		<div class="text-sm text-gray-500">Draw to see colors</div>
	{:else}
		<div class="grid grid-cols-6 gap-2">
			{#each uniqueColors() as color (color)}
				<button
					type="button"
					class="h-8 w-8 rounded border-2 transition-transform hover:scale-110"
					class:border-gray-300={selectedColor !== color}
					class:border-gray-900={selectedColor === color}
					style:background-color={packedToCss(color)}
					onclick={() => onSelectColor(color)}
					title={colorToHex(color)}
					aria-label={`Select color ${colorToHex(color)}`}
				></button>
			{/each}
		</div>
	{/if}
</div>
