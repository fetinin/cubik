import { describe, it, expect, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { SavedAnimation } from '$lib/api/generated';
import LoadAnimationModal from './LoadAnimationModal.svelte';

function makeSavedAnimation(overrides: Partial<SavedAnimation> = {}): SavedAnimation {
	return {
		id: 'anim-1',
		deviceId: 'device-1',
		name: 'Palm Island',
		frames: [],
		createdAt: new Date('2026-01-01T00:00:00Z'),
		updatedAt: new Date('2026-01-01T00:00:00Z'),
		...overrides
	};
}

describe('LoadAnimationModal', () => {
	it('Play button calls onplay with the id and leaves the modal open', async () => {
		const onplay = vi.fn();
		const onload = vi.fn();
		const screen = await render(LoadAnimationModal, {
			animations: [makeSavedAnimation({ id: 'anim-7' })],
			open: true,
			onload,
			onplay,
			ondelete: vi.fn()
		});

		await screen.getByTestId('play-button').click();

		expect(onplay).toHaveBeenCalledWith('anim-7');
		expect(onload).not.toHaveBeenCalled();
		await expect.element(screen.getByTestId('load-modal-content')).toBeVisible();
	});
});
