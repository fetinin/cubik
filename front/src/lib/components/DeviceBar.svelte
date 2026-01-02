<script lang="ts">
	import type { Device } from '$lib/api/mock';

	type Props = {
		devices: Device[];
		selectedDeviceId: string | null;
		onSelect: (deviceId: string) => void;
	};

	let { devices, selectedDeviceId, onSelect }: Props = $props();
</script>

<div class="flex items-center gap-3">
	<div class="text-sm font-medium">Device</div>
	{#if devices.length === 0}
		<div class="text-sm text-gray-500">Searching…</div>
	{:else}
		<select
			class="rounded border border-gray-300 bg-white px-2 py-1 text-sm"
			value={selectedDeviceId ?? ''}
			onchange={(e) => onSelect((e.currentTarget as HTMLSelectElement).value)}
			data-testid="device-select"
		>
			{#each devices as d (d.id)}
				<option value={d.id}>{d.name}</option>
			{/each}
		</select>
	{/if}
</div>
