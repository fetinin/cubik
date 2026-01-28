<script lang="ts">
	import { packedToCss, type EditorMode, type PackedRGB } from '$lib/state/editor';
	import { onDestroy } from 'svelte';

	type Props = {
		width: number;
		height: number;
		pixels: PackedRGB[];
		paintColor: PackedRGB;
		mode: EditorMode;
		selection: Set<number>;
		onPaint: (index: number, color: PackedRGB) => void;
		onSelect: (indices: Set<number>) => void;
		onMove: (deltaX: number, deltaY: number) => void;
	};

	let { width, height, pixels, paintColor, mode, selection, onPaint, onSelect, onMove }: Props =
		$props();

	let isPainting = $state(false);
	let isSelecting = $state(false);
	let isMoving = $state(false);
	let moveStartIndex = $state<number | null>(null);
	let selectionStart = $state<{ x: number; y: number } | null>(null);
	let selectionEnd = $state<{ x: number; y: number } | null>(null);

	const rows = $derived([...Array(height).keys()]);
	const cols = $derived([...Array(width).keys()]);

	function paint(index: number): void {
		onPaint(index, paintColor);
	}

	function indexToCoords(index: number): { x: number; y: number } {
		return { x: index % width, y: Math.floor(index / width) };
	}

	function getSelectionRingClass(index: number): string {
		if (selection.has(index)) return 'ring-2 ring-blue-500';
		if (previewSelection.has(index)) return 'ring-2 ring-blue-300';
		return '';
	}

	function computeBoxSelection(
		start: { x: number; y: number },
		end: { x: number; y: number }
	): Set<number> {
		const minX = Math.min(start.x, end.x);
		const maxX = Math.max(start.x, end.x);
		const minY = Math.min(start.y, end.y);
		const maxY = Math.max(start.y, end.y);

		const indices: number[] = [];
		for (let y = minY; y <= maxY; y++) {
			for (let x = minX; x <= maxX; x++) {
				indices.push(y * width + x);
			}
		}
		return new Set(indices);
	}

	const previewSelection = $derived.by(() => {
		if (!isSelecting || !selectionStart || !selectionEnd) return new Set<number>();
		return computeBoxSelection(selectionStart, selectionEnd);
	});

	function handlePointerDown(index: number) {
		if (mode === 'paint') {
			isPainting = true;
			paint(index);
		} else if (mode === 'select') {
			if (selection.has(index)) {
				// Start moving
				isMoving = true;
				moveStartIndex = index;
			} else {
				// Start new box selection
				isSelecting = true;
				const coords = indexToCoords(index);
				selectionStart = coords;
				selectionEnd = coords;
			}
		}
	}

	function handlePointerEnter(index: number) {
		if (mode === 'paint' && isPainting) {
			paint(index);
		} else if (mode === 'select' && isSelecting) {
			selectionEnd = indexToCoords(index);
		}
	}

	function handlePointerUp(index: number) {
		if (mode === 'select' && isSelecting && selectionStart && selectionEnd) {
			const selected = computeBoxSelection(selectionStart, selectionEnd);
			onSelect(selected);
		} else if (mode === 'select' && isMoving && moveStartIndex !== null) {
			const start = indexToCoords(moveStartIndex);
			const end = indexToCoords(index);
			const deltaX = end.x - start.x;
			const deltaY = end.y - start.y;
			if (deltaX !== 0 || deltaY !== 0) {
				onMove(deltaX, deltaY);
			}
		}
		resetState();
	}

	function resetState() {
		isPainting = false;
		isSelecting = false;
		isMoving = false;
		moveStartIndex = null;
		selectionStart = null;
		selectionEnd = null;
	}

	function onGlobalPointerUp() {
		resetState();
	}

	if (typeof window !== 'undefined') {
		window.addEventListener('pointerup', onGlobalPointerUp);
	}

	onDestroy(() => {
		if (typeof window !== 'undefined') {
			window.removeEventListener('pointerup', onGlobalPointerUp);
		}
	});
</script>

<div class="inline-flex flex-shrink-0 flex-col gap-1" data-testid="matrix">
	{#each rows as y (y)}
		<div class="flex gap-1">
			{#each cols as x (x)}
				{@const i = y * width + x}
				<button
					type="button"
					class="h-7 w-7 rounded border border-gray-300 {getSelectionRingClass(i)}"
					style:background-color={packedToCss(pixels[i] ?? 0)}
					onpointerdown={() => handlePointerDown(i)}
					onpointerenter={() => handlePointerEnter(i)}
					onpointerup={() => handlePointerUp(i)}
					aria-label={`pixel ${x},${y}`}
					data-testid={`matrix-cell-${i}`}
				></button>
			{/each}
		</div>
	{/each}
</div>
