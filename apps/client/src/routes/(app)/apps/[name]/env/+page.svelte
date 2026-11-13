<script lang="ts">
	import { Alert, ConfirmModal, EmptyState, Section, Skeleton, icons, toast } from '@facile/muse';
	import { api, type EnvEntry, type EnvPatch } from '$lib/api';
	import { currentApp } from '$lib/current-app.svelte';
	import { errorText } from '$lib/status';
	import EnvAdd from '$lib/components/EnvAdd.svelte';
	import EnvPaste from '$lib/components/EnvPaste.svelte';
	import EnvRow from '$lib/components/EnvRow.svelte';

	const current = currentApp();

	let entries = $state<EnvEntry[] | null>(null);
	let revealed = $state<Record<string, string>>({});
	let failure = $state<string | null>(null);
	let removing = $state<string | null>(null);

	const keys = $derived((entries ?? []).map((entry) => entry.key));

	/** Loads the keys with their masked values; no plain value leaves the server here. */
	async function load(): Promise<void> {
		try {
			entries = await api.env(current.name);
			revealed = {};
			failure = null;
		} catch (error) {
			failure = errorText(error);
		}
	}

	/** Fetches one plain value on request and keeps it on screen until hidden or changed. */
	async function reveal(key: string): Promise<string | undefined> {
		try {
			revealed[key] = await api.reveal(current.name, key);
			return revealed[key];
		} catch (error) {
			toast.danger(errorText(error), { title: `Could not reveal ${key}` });
			return undefined;
		}
	}

	/** Sends a merge patch: only the named keys change, every other variable stays as it was. */
	async function patch(change: EnvPatch, message: string): Promise<boolean> {
		try {
			entries = await api.patchEnv(current.name, change);
			for (const key of [...Object.keys(change.set ?? {}), ...(change.unset ?? [])]) delete revealed[key];
			toast.success(message, { title: 'Restart or deploy to apply it' });
			return true;
		} catch (error) {
			toast.danger(errorText(error), { title: 'The environment was not changed' });
			return false;
		}
	}

	/** Sets the given variables, saying how many changed. */
	function set(values: Record<string, string>): Promise<boolean> {
		const count = Object.keys(values).length;
		return patch({ set: values }, count === 1 ? `${Object.keys(values)[0]} saved.` : `${count} variables saved.`);
	}

	/** Removes the variable the person confirmed. */
	async function remove(): Promise<void> {
		if (removing) await patch({ unset: [removing] }, `${removing} removed.`);
		removing = null;
	}

	$effect(() => {
		void current.name;
		void load();
	});
</script>

<Section title="Variables" description="Values stay masked until you reveal one. Changes apply on the next restart or deploy.">
	{#if failure}
		<Alert tone="danger" title="Could not load the environment">{failure}</Alert>
	{/if}

	{#if entries === null}
		<Skeleton class="h-40 rounded-fc-lg" />
	{:else if entries.length === 0}
		<EmptyState icon={icons.key} title="No variables yet" description="Add one below, or paste a .env block." />
	{:else}
		<ul class="flex flex-col rounded-fc-lg bg-fc-component px-5 py-1">
			{#each entries as entry (entry.key)}
				<EnvRow
					{entry}
					value={revealed[entry.key]}
					onReveal={() => reveal(entry.key)}
					onHide={() => delete revealed[entry.key]}
					onSave={(value) => set({ [entry.key]: value })}
					onRemove={() => (removing = entry.key)}
				/>
			{/each}
		</ul>
	{/if}
</Section>

<Section title="Add a variable" card>
	<EnvAdd existing={keys} onSet={set} />
</Section>

<Section title="Paste a .env block" description="Merges into the variables above. It never removes one." card>
	<EnvPaste existing={keys} onSet={set} />
</Section>

<ConfirmModal
	open={removing !== null}
	title="Remove {removing}?"
	description="{current.name} loses this variable on its next restart or deploy."
	confirmLabel="Remove"
	tone="danger"
	icon={icons.remove}
	onConfirm={remove}
	onCancel={() => (removing = null)}
/>
