<script lang="ts">
	import { Alert, Button, Section, Spinner, icons } from '@facile/muse';
	import { runtimeLogUrl } from '$lib/api';
	import { currentApp } from '$lib/current-app.svelte';
	import { follow } from '$lib/sse';
	import LogView from '$lib/components/LogView.svelte';

	const MAX_LINES = 5_000;

	const current = currentApp();

	let lines = $state<string[]>([]);
	let broken = $state(false);
	let attempt = $state(0);

	$effect(() => {
		void attempt;
		lines = [];
		broken = false;
		return follow(runtimeLogUrl(current.name), {
			onLine: (line) => {
				lines.push(line);
				if (lines.length > MAX_LINES) lines.splice(0, lines.length - MAX_LINES);
			},
			onBroken: () => {
				broken = true;
				void current.refresh();
			}
		});
	});
</script>

<Section title="Runtime logs" description="The last 200 journal lines of this app's unit, then everything new.">
	{#snippet actions()}
		{#if broken}
			<Button size="lg" variant="secondary" icon={icons.refresh} onclick={() => attempt++}>Reconnect</Button>
		{:else}
			<Spinner size="sm" label="Following the journal" />
		{/if}
	{/snippet}

	{#if broken}
		<Alert tone="warning">The log stream stopped. Reconnect to read it again from the last 200 lines.</Alert>
	{/if}

	<LogView {lines} label="Runtime logs" empty="No journal lines yet." />
</Section>
