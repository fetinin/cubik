<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { applyAnimation, getDevices, getMatrixSize, stopAnimation } from '$lib/api/mock';
	import DeviceBar from '$lib/components/DeviceBar.svelte';
	import AnimationPreview from '$lib/components/AnimationPreview.svelte';
	import ColorPickerRGB from '$lib/components/ColorPickerRGB.svelte';
	import ColorSelector from '$lib/components/ColorSelector.svelte';
	import FramesPanel from '$lib/components/FramesPanel.svelte';
	import MatrixGrid from '$lib/components/MatrixGrid.svelte';
	import {
		buildAnimationPayload,
		createEditorState,
		createFrameFromPixels,
		initPixelsForSize,
		loadFrameIntoPixels,
		paintPixel,
		packRGB,
		replaceSelectedFramePixels,
		type PackedRGB
	} from '$lib/state/editor';

	const editor = createEditorState();
	const devices = editor.devices;
	const selectedDeviceId = editor.selectedDeviceId;
	const selectedDevice = editor.selectedDevice;
	const matrix = editor.matrix;
	const pixels = editor.pixels;
	const frames = editor.frames;
	const selectedFrameId = editor.selectedFrameId;
	const applyStatus = editor.applyStatus;

	let paintColor: PackedRGB = packRGB(255, 0, 0);
	let loading = true;
	let error: string | null = null;
	let stoppedNotice = false;
	let stoppedNoticeTimeout: ReturnType<typeof setTimeout> | null = null;
	let appliedNoticeTimeout: ReturnType<typeof setTimeout> | null = null;

	async function selectDevice(deviceId: string) {
		editor.selectedDeviceId.set(deviceId);
		const size = await getMatrixSize(deviceId);
		editor.matrix.set(size);
		editor.pixels.set(initPixelsForSize(size, 0x000000));
		editor.frames.set([]);
		editor.selectedFrameId.set(null);
	}

	onMount(async () => {
		try {
			loading = true;
			const devices = await getDevices();
			editor.devices.set(devices);

			const first = devices[0];
			if (first) await selectDevice(first.id);
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	});

	function onPaint(index: number, color: PackedRGB) {
		editor.pixels.update((p) => paintPixel(p, index, color));
	}

	function saveNewFrame() {
		const n = get(frames).length + 1;
		const frame = createFrameFromPixels(`Frame ${n}`, get(pixels));
		editor.frames.update((list) => [...list, frame]);
		loadFrameIntoPixels(editor, frame.id);
	}

	function selectFrame(frameId: string) {
		loadFrameIntoPixels(editor, frameId);
	}

	function deleteFrame(frameId: string) {
		editor.frames.update((list) => list.filter((f) => f.id !== frameId));
		const selected = get(selectedFrameId);
		if (selected === frameId) editor.selectedFrameId.set(null);
	}

	function moveFrame(frameId: string, dir: -1 | 1) {
		editor.frames.update((list) => {
			const idx = list.findIndex((f) => f.id === frameId);
			if (idx === -1) return list;
			const nextIdx = idx + dir;
			if (nextIdx < 0 || nextIdx >= list.length) return list;
			const next = list.slice();
			const [item] = next.splice(idx, 1);
			next.splice(nextIdx, 0, item);
			return next;
		});
	}

	async function applyCurrentAnimation() {
		const deviceLocation = get(selectedDevice)?.location ?? null;
		const size = get(matrix);
		const framesList = get(frames);
		if (!deviceLocation || !size || framesList.length === 0) return;

		editor.applyStatus.set({ state: 'applying' });
		if (appliedNoticeTimeout) {
			clearTimeout(appliedNoticeTimeout);
			appliedNoticeTimeout = null;
		}
		try {
			const payload = buildAnimationPayload(size, framesList);
			await applyAnimation(deviceLocation, payload);
			editor.applyStatus.set({ state: 'success' });
			appliedNoticeTimeout = setTimeout(() => {
				editor.applyStatus.set({ state: 'idle' });
				appliedNoticeTimeout = null;
			}, 2000);
		} catch (e) {
			editor.applyStatus.set({
				state: 'error',
				message: e instanceof Error ? e.message : String(e)
			});
		}
	}

	async function stopCurrentAnimation() {
		const deviceLocation = get(selectedDevice)?.location ?? null;
		if (!deviceLocation) return;
		await stopAnimation(deviceLocation);
		stoppedNotice = true;
		if (stoppedNoticeTimeout) clearTimeout(stoppedNoticeTimeout);
		stoppedNoticeTimeout = setTimeout(() => {
			stoppedNotice = false;
			stoppedNoticeTimeout = null;
		}, 2000);
	}
</script>

<main class="mx-auto max-w-6xl p-6">
	<div class="flex items-center justify-between gap-4">
		<h1 class="text-xl font-semibold">Cubik</h1>
		<DeviceBar devices={$devices} selectedDeviceId={$selectedDeviceId} onSelect={selectDevice} />
	</div>

	{#if error}
		<div class="mt-4 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="mt-6 text-sm text-gray-500">Loading…</div>
	{:else if $matrix === null}
		<div class="mt-6 text-sm text-gray-500">No devices found.</div>
	{:else}
		{@const size = $matrix!}

		<div class="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-[auto_320px]">
			<div class="flex flex-col gap-6">
				<section class="rounded border border-gray-200 p-4">
					<div class="flex items-center justify-between gap-4">
						<div class="text-sm text-gray-600">Matrix: {size.width}×{size.height}</div>
						<div class="flex items-center gap-2">
							<button
								type="button"
								class="rounded bg-gray-900 px-3 py-1.5 text-xs font-medium text-white"
								onclick={saveNewFrame}
								data-testid="save-frame"
							>
								Save frame
							</button>
							<button
								type="button"
								class="rounded border border-gray-300 px-3 py-1.5 text-xs font-medium disabled:opacity-50"
								onclick={() => replaceSelectedFramePixels(editor)}
								disabled={$selectedFrameId === null}
							>
								Replace selected
							</button>
							<button
								type="button"
								class="rounded border border-gray-300 px-3 py-1.5 text-xs font-medium"
								onclick={() => pixels.set(initPixelsForSize(size, 0x000000))}
							>
								Clear
							</button>
						</div>
					</div>

					<div class="mt-4">
						<MatrixGrid
							width={size.width}
							height={size.height}
							pixels={$pixels}
							{paintColor}
							{onPaint}
						/>
					</div>
				</section>

				<section class="rounded border border-gray-200 p-4">
					<AnimationPreview
						width={size.width}
						height={size.height}
						frames={$frames.map((f) => f.pixels)}
					/>
				</section>
			</div>

			<aside class="flex flex-col gap-6">
				<section class="rounded border border-gray-200 p-4">
					<ColorPickerRGB color={paintColor} onChange={(c) => (paintColor = c)} />
				</section>

				<section class="rounded border border-gray-200 p-4">
					<ColorSelector
						pixels={$pixels}
						selectedColor={paintColor}
						onSelectColor={(c) => (paintColor = c)}
					/>
				</section>

				<section class="rounded border border-gray-200 p-4">
					<FramesPanel
						frames={$frames}
						selectedFrameId={$selectedFrameId}
						onSelectFrame={selectFrame}
						onDeleteFrame={deleteFrame}
						onMoveFrame={moveFrame}
					/>
				</section>

				<section class="rounded border border-gray-200 p-4">
					<div class="flex items-center justify-between gap-3">
						<div class="text-sm font-medium">Apply</div>
						<div class="flex items-center gap-2">
							<button
								type="button"
								class="rounded bg-blue-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50"
								onclick={applyCurrentAnimation}
								disabled={$frames.length === 0}
								data-testid="apply-animation"
							>
								Apply animation
							</button>

							<button
								type="button"
								class="rounded border border-gray-300 px-3 py-1.5 text-xs font-medium disabled:opacity-50"
								onclick={stopCurrentAnimation}
								disabled={!$selectedDeviceId}
								data-testid="stop-animation"
							>
								Stop animation
							</button>
						</div>
					</div>

					{#if $applyStatus.state === 'applying'}
						<div class="mt-3 text-sm text-gray-600">Applying…</div>
					{:else if $applyStatus.state === 'success'}
						<div class="mt-3 text-sm text-green-700">Applied.</div>
					{:else if $applyStatus.state === 'error'}
						<div class="mt-3 text-sm text-red-700">
							Failed: {$applyStatus.message}
						</div>
					{/if}

					{#if stoppedNotice}
						<div class="mt-3 text-sm text-gray-600">Animation stopped.</div>
					{/if}
				</section>
			</aside>
		</div>
	{/if}
</main>
