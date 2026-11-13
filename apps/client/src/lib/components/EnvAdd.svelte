<script lang="ts">
	import { Button, Field, Input, Textarea, icons } from '@facile/muse';
	import { isEnvKey } from '$lib/dotenv';

	let { existing, onSet }: { existing: string[]; onSet: (set: Record<string, string>) => Promise<boolean> } = $props();

	let key = $state('');
	let value = $state('');
	let tried = $state(false);
	let busy = $state(false);

	const name = $derived(key.trim());
	const error = $derived(isEnvKey(name) ? '' : 'Letters, digits and "_", not starting with a digit.');
	const replaces = $derived(existing.includes(name));

	/** Sets one variable and clears the form once the API accepted it. */
	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		tried = true;
		if (error) return;
		busy = true;
		if (await onSet({ [name]: value })) {
			key = '';
			value = '';
			tried = false;
		}
		busy = false;
	}
</script>

<form class="flex flex-col gap-4" onsubmit={submit} novalidate>
	<Field label="Name" error={tried ? error : ''} helper={replaces ? 'This name exists: saving replaces its value.' : undefined}>
		{#snippet children({ id, describedBy })}
			<Input {id} aria-describedby={describedBy} aria-invalid={tried && !!error} class="font-fc-mono" autocomplete="off" autocapitalize="characters" spellcheck="false" placeholder="DATABASE_URL" bind:value={key} />
		{/snippet}
	</Field>
	<Field label="Value">
		{#snippet children({ id, describedBy })}
			<Textarea {id} aria-describedby={describedBy} class="font-fc-mono text-fc-sm" rows={1} autoGrow spellcheck="false" bind:value />
		{/snippet}
	</Field>
	<div>
		<Button size="lg" type="submit" icon={icons.plus} disabled={busy}>{replaces ? 'Replace value' : 'Add variable'}</Button>
	</div>
</form>
