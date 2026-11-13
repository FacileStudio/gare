<script lang="ts">
	import { Alert, Button, Field, Textarea, icons } from '@facile/muse';
	import { parseDotenv } from '$lib/dotenv';

	let { existing, onSet }: { existing: string[]; onSet: (set: Record<string, string>) => Promise<boolean> } = $props();

	let text = $state('');
	let busy = $state(false);

	const parsed = $derived(parseDotenv(text));
	const keys = $derived(Object.keys(parsed.entries));
	const replaced = $derived(keys.filter((key) => existing.includes(key)));
	const summary = $derived(
		`${keys.length} variable${keys.length === 1 ? '' : 's'}: ${keys.length - replaced.length} new, ${replaced.length} replaced. Variables not in the block stay as they are.`
	);

	/** Merges the block into the environment; nothing outside it is removed. */
	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (keys.length === 0 || parsed.invalid.length > 0) return;
		busy = true;
		if (await onSet(parsed.entries)) text = '';
		busy = false;
	}
</script>

<form class="flex flex-col gap-4" onsubmit={submit}>
	<Field label=".env block" helper="One KEY=value per line. Comments, export and quotes are understood.">
		{#snippet children({ id, describedBy })}
			<Textarea {id} aria-describedby={describedBy} class="font-fc-mono text-fc-sm" rows={6} spellcheck="false" placeholder={'API_URL=https://example.com\nSECRET="with spaces"'} bind:value={text} />
		{/snippet}
	</Field>

	{#if parsed.invalid.length > 0}
		<Alert tone="danger" title="Some lines are not KEY=value">
			Fix line{parsed.invalid.length === 1 ? '' : 's'} {parsed.invalid.join(', ')} before merging.
		</Alert>
	{:else if keys.length > 0}
		<p class="text-fc-sm text-fc-fg-muted" aria-live="polite">{summary}</p>
	{/if}

	<div>
		<Button size="lg" type="submit" icon={icons.upload} disabled={busy || keys.length === 0 || parsed.invalid.length > 0}>
			{busy ? 'Merging' : 'Merge into environment'}
		</Button>
	</div>
</form>
