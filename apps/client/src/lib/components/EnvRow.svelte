<script lang="ts">
	import { Button, Textarea, icons } from '@facile/muse';
	import type { EnvEntry } from '$lib/api';

	type Props = {
		entry: EnvEntry;
		value: string | undefined;
		onReveal: () => Promise<string | undefined>;
		onHide: () => void;
		onSave: (value: string) => Promise<boolean>;
		onRemove: () => void;
	};

	let { entry, value, onReveal, onHide, onSave, onRemove }: Props = $props();

	let editing = $state(false);
	let draft = $state('');
	let busy = $state(false);

	/** Opens the editor on the current value, which has to be revealed first to be edited. */
	async function edit(): Promise<void> {
		busy = true;
		const current = value ?? (await onReveal());
		busy = false;
		if (current === undefined) return;
		draft = current;
		editing = true;
	}

	/** Saves the draft; the editor stays open when the API refuses it. */
	async function save(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		busy = true;
		editing = !(await onSave(draft));
		busy = false;
	}
</script>

<li class="flex flex-col gap-2 py-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<span class="min-w-0 font-fc-mono text-fc-sm font-medium break-all text-fc-fg">{entry.key}</span>
		{#if !editing}
			<div class="flex flex-wrap gap-1">
				{#if value === undefined}
					<Button size="lg" variant="ghost" icon={icons.eye} disabled={busy} onclick={() => void onReveal()} aria-label="Reveal {entry.key}">Reveal</Button>
				{:else}
					<Button size="lg" variant="ghost" icon={icons.eyeClosed} onclick={onHide} aria-label="Hide {entry.key}">Hide</Button>
				{/if}
				<Button size="lg" variant="ghost" icon={icons.edit} disabled={busy} onclick={edit} aria-label="Edit {entry.key}">Edit</Button>
				<Button size="lg" variant="ghost-danger" icon={icons.remove} onclick={onRemove} aria-label="Remove {entry.key}">Remove</Button>
			</div>
		{/if}
	</div>

	{#if editing}
		<form class="flex flex-col gap-2" onsubmit={save}>
			<Textarea class="font-fc-mono text-fc-sm" rows={1} autoGrow aria-label="New value for {entry.key}" spellcheck="false" bind:value={draft} />
			<div class="flex flex-wrap gap-2">
				<Button size="lg" type="submit" disabled={busy}>{busy ? 'Saving' : 'Save'}</Button>
				<Button size="lg" variant="ghost" onclick={() => (editing = false)}>Cancel</Button>
			</div>
		</form>
	{:else}
		<span class="font-fc-mono text-fc-sm break-all whitespace-pre-wrap text-fc-fg-muted">{value ?? entry.masked}</span>
	{/if}
</li>
