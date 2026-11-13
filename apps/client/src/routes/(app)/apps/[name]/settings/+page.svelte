<script lang="ts">
	import { goto } from '$app/navigation';
	import { Alert, Button, Checkbox, DangerZone, Field, Input, Modal, icons, toast } from '@facile/muse';
	import { api } from '$lib/api';
	import { currentApp } from '$lib/current-app.svelte';
	import { errorText } from '$lib/status';

	const current = currentApp();

	let open = $state(false);
	let typed = $state('');
	let volumes = $state(false);
	let busy = $state(false);
	let failure = $state<string | null>(null);

	const confirmed = $derived(typed === current.name);
	const compose = $derived(current.app?.type === 'compose');

	/** Opens the confirmation with an empty name, so nobody confirms by habit. */
	function ask(): void {
		typed = '';
		volumes = false;
		failure = null;
		open = true;
	}

	/** Destroys the app once its name was typed back, then returns to the list. */
	async function destroy(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (!confirmed) return;
		busy = true;
		try {
			await api.destroyApp(current.name, compose && volumes);
			open = false;
			toast.success(`${current.name} destroyed.`);
			await goto('/', { replaceState: true });
		} catch (error) {
			failure = errorText(error);
		} finally {
			busy = false;
		}
	}
</script>

<DangerZone description="Destroying stops the app and removes its unit, ingress, files and image. It cannot be undone.">
	<Button size="lg" variant="danger" icon={icons.remove} onclick={ask}>Destroy {current.name}</Button>
</DangerZone>

<Modal {open} title="Destroy {current.name}?" onClose={() => (open = false)}>
	<form id="destroy" class="flex flex-col gap-4" onsubmit={destroy}>
		<p class="text-fc-sm text-fc-fg-muted">Type the app name to confirm. Its environment and deploy history go too.</p>
		<Field label="App name">
			{#snippet children({ id, describedBy })}
				<Input {id} aria-describedby={describedBy} class="font-fc-mono" autocomplete="off" autocapitalize="off" spellcheck="false" placeholder={current.name} bind:value={typed} />
			{/snippet}
		</Field>
		{#if compose}
			<Checkbox label="Also delete the compose volumes and their data" bind:checked={volumes} />
		{/if}
		{#if failure}
			<Alert tone="danger" title="The destroy failed">{failure}</Alert>
		{/if}
	</form>
	{#snippet footer()}
		<Button size="lg" variant="ghost" onclick={() => (open = false)}>Cancel</Button>
		<Button size="lg" variant="danger" type="submit" form="destroy" disabled={!confirmed || busy}>
			{busy ? 'Destroying' : 'Destroy'}
		</Button>
	{/snippet}
</Modal>
