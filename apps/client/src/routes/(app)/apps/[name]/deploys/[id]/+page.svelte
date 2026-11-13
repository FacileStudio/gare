<script lang="ts">
	import { page } from '$app/state';
	import { Alert, Badge, Button, Inline, Section, Spinner, icons } from '@facile/muse';
	import { api, deployLogUrl, type Deploy } from '$lib/api';
	import { currentApp } from '$lib/current-app.svelte';
	import { follow } from '$lib/sse';
	import { ago, deployLabel, deployTone, errorText, isLive, shortCommit, took } from '$lib/status';
	import LogView from '$lib/components/LogView.svelte';

	const MAX_LINES = 10_000;

	const current = currentApp();
	const id = $derived(page.params.id ?? '');

	let deploy = $state<Deploy | null>(null);
	let lines = $state<string[]>([]);
	let failure = $state<string | null>(null);
	let reconnecting = $state(false);

	/** Keeps the newest lines when a runaway build prints more than the page should hold. */
	function push(line: string): void {
		lines.push(line);
		if (lines.length > MAX_LINES) lines.splice(0, lines.length - MAX_LINES);
	}

	$effect(() => {
		const name = current.name;
		const wanted = id;
		let stop = () => {};
		let retry: ReturnType<typeof setTimeout> | undefined;
		let gone = false;

		async function attach(): Promise<void> {
			const found = (await api.deploys(name)).find((item) => item.id === wanted) ?? null;
			if (gone) return;
			deploy = found;
			reconnecting = false;
			if (!found) {
				failure = 'This deploy is not in the history any more.';
			} else {
				lines = [];
				stop = follow(deployLogUrl(name, wanted), {
					onLine: push,
					onDone: (finished) => {
						deploy = finished;
						void current.refresh();
					},
					onBroken: () => {
						if (gone) return;
						reconnecting = true;
						retry = setTimeout(() => void attach().catch(fail), 2000);
					}
				});
			}
		}

		function fail(error: unknown): void {
			if (!gone) failure = errorText(error);
		}

		failure = null;
		void attach().catch(fail);
		return () => {
			gone = true;
			clearTimeout(retry);
			stop();
		};
	});
</script>

<Section title="Deploy log">
	{#snippet actions()}
		<Button size="lg" variant="ghost" icon={icons.chevronLeft} href="/apps/{encodeURIComponent(current.name)}/deploys">
			All deploys
		</Button>
	{/snippet}

	{#if deploy}
		<Inline gap="tight">
			<Badge tone={deployTone[deploy.status]}>{deployLabel[deploy.status]}</Badge>
			<span class="font-fc-mono text-fc-sm text-fc-fg">{shortCommit(deploy.commit) || 'commit pending'}</span>
			<span class="text-fc-sm text-fc-fg-muted">
				Started {ago(deploy.started_at)} by {deploy.trigger}{took(deploy) ? `, took ${took(deploy)}` : ''}
			</span>
			{#if isLive(deploy) && !reconnecting}
				<Spinner size="sm" label="Following the log" />
			{/if}
		</Inline>
		{#if deploy.error}
			<Alert tone="danger" title="The deploy failed">{deploy.error}</Alert>
		{/if}
	{/if}

	{#if reconnecting}
		<Alert tone="warning">The log stream dropped. Reconnecting.</Alert>
	{/if}
	{#if failure}
		<Alert tone="danger" title="Could not read the log">{failure}</Alert>
	{/if}

	<LogView {lines} label="Deploy log" empty={deploy && isLive(deploy) ? 'Waiting for the first line.' : 'This deploy printed nothing.'} />
</Section>
