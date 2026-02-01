<script lang="ts">
	import type { PackedRGB } from '$lib/state/editor';
	import { packedToCss, extractPaletteFromPixels } from '$lib/state/editor';

	type Props = {
		pixels: PackedRGB[];
		selectedColor: PackedRGB;
		onSelectColor: (color: PackedRGB) => void;
	};

	let { pixels, selectedColor, onSelectColor }: Props = $props();

	const uniqueColors = $derived(extractPaletteFromPixels(pixels));

	// Convert packed RGB to hex string for display
	function colorToHex(color: PackedRGB): string {
		return '#' + color.toString(16).padStart(6, '0').toUpperCase();
	}
</script>

<div class="flex flex-col gap-3">
	<div class="text-sm font-medium">Palette</div>

	{#if uniqueColors.length === 0}
		<div class="text-sm text-gray-500">Draw to see colors</div>
	{:else}
		<div class="grid grid-cols-6 gap-2">
			{#each uniqueColors as color (color)}
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
