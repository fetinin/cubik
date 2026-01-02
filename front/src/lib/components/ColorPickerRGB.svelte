<script lang="ts">
	import { packRGB, unpackRGB, type PackedRGB } from '$lib/state/editor';

	type Props = {
		color: PackedRGB;
		onChange: (color: PackedRGB) => void;
	};

	let { color, onChange }: Props = $props();

	let r = $state(0);
	let g = $state(0);
	let b = $state(0);

	$effect(() => {
		// Keep sliders in sync if parent updates the color programmatically
		const { r: rr, g: gg, b: bb } = unpackRGB(color);
		r = rr;
		g = gg;
		b = bb;
	});

	$effect(() => {
		const next = packRGB(r, g, b);
		if (next !== color) onChange(next);
	});
</script>

<div class="flex flex-col gap-3">
	<div class="flex items-center gap-3">
		<div class="text-sm font-medium">Color</div>
		<div
			class="h-6 w-10 rounded border border-gray-300"
			style:background-color={`rgb(${r}, ${g}, ${b})`}
			aria-label="selected color"
		></div>
		<div class="text-xs text-gray-500">{r},{g},{b}</div>
	</div>

	<div class="grid grid-cols-[auto_1fr_auto] items-center gap-x-3 gap-y-2">
		<label class="text-xs text-gray-600" for="r">R</label>
		<input
			id="r"
			type="range"
			min="0"
			max="255"
			bind:value={r}
			class="w-full"
			data-testid="slider-r"
		/>
		<div class="w-10 text-right text-xs tabular-nums">{r}</div>

		<label class="text-xs text-gray-600" for="g">G</label>
		<input id="g" type="range" min="0" max="255" bind:value={g} class="w-full" />
		<div class="w-10 text-right text-xs tabular-nums">{g}</div>

		<label class="text-xs text-gray-600" for="b">B</label>
		<input id="b" type="range" min="0" max="255" bind:value={b} class="w-full" />
		<div class="w-10 text-right text-xs tabular-nums">{b}</div>
	</div>
</div>
