<script lang="ts">
	import ColorPicker from 'svelte-awesome-color-picker';
	import { packRGB, unpackRGB, type PackedRGB } from '$lib/state/editor';
	import type { RgbaColor } from 'svelte-awesome-color-picker';

	type Props = {
		color: PackedRGB;
		onChange: (color: PackedRGB) => void;
	};

	let { color, onChange }: Props = $props();

	let rgb: RgbaColor | null = $state({ r: 255, g: 0, b: 0, a: 1 });

	// Sync incoming color prop to rgb state
	$effect(() => {
		const { r, g, b } = unpackRGB(color);
		rgb = { r, g, b, a: 1 };
	});

	// Handle color changes from picker
	function handleColorChange(pickerColor: { rgb: RgbaColor | null; hex: string | null }) {
		if (pickerColor.rgb) {
			const packed = packRGB(pickerColor.rgb.r, pickerColor.rgb.g, pickerColor.rgb.b);
			onChange(packed);
		}
	}
</script>

<div class="flex flex-col gap-3">
	<div class="text-sm font-medium">Color</div>
	<ColorPicker
		bind:rgb
		onInput={handleColorChange}
		position="responsive"
		isDialog={false}
		isAlpha={false}
	/>
</div>
