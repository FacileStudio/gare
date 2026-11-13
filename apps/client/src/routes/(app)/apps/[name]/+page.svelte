<script lang="ts">
	import { Badge, Button, Inline, Section, icons, toast } from '@facile/muse';
	import { api, type App, type Lifecycle } from '$lib/api';
	import { deployHref, startDeploy } from '$lib/actions';
	import { currentApp } from '$lib/current-app.svelte';
	import { ago, deployLabel, deployTone, errorText, shortCommit } from '$lib/status';

	const current = currentApp();
	const app = $derived(current.app as App);
	const running = $derived(app.status === 'active' || app.status === 'activating');

	let busy = $state<Lifecycle | 'deploy' | null>(null);

	const done: Record<Lifecycle, string> = { start: 'started', stop: 'stopped', restart: 'restarted' };

	/** Runs start, stop or restart and shows the state the API answers with. */
	async function run(action: Lifecycle): Promise<void> {
		busy = action;
		try {
			current.app = await api.lifecycle(app.name, action);
			toast.success(`${app.name} ${done[action]}.`);
		} catch (error) {
			toast.danger(errorText(error), { title: `Could not ${action} ${app.name}` });
		} finally {
			busy = null;
		}
	}

	/** Queues a deploy and opens its live log. */
	async function deploy(): Promise<void> {
		busy = 'deploy';
		await startDeploy(app.name);
		busy = null;
	}
</script>

<Section title="Actions">
	<Inline gap="tight">
		<Button size="lg" icon={icons.refresh} disabled={busy !== null} onclick={deploy}>
			{busy === 'deploy' ? 'Starting deploy' : 'Deploy now'}
		</Button>
		<Button size="lg" variant="secondary" disabled={busy !== null} onclick={() => run('restart')}>
			{busy === 'restart' ? 'Restarting' : 'Restart'}
		</Button>
		{#if running}
			<Button size="lg" variant="secondary" disabled={busy !== null} onclick={() => run('stop')}>
				{busy === 'stop' ? 'Stopping' : 'Stop'}
			</Button>
		{:else}
			<Button size="lg" variant="secondary" disabled={busy !== null} onclick={() => run('start')}>
				{busy === 'start' ? 'Starting' : 'Start'}
			</Button>
		{/if}
	</Inline>
</Section>

<Section title="Details" card>
	<dl class="grid grid-cols-1 gap-x-6 gap-y-1 text-fc-sm sm:grid-cols-[auto_1fr] sm:gap-y-3">
		<dt class="text-fc-fg-muted">Repository</dt>
		<dd class="pb-3 font-fc-mono break-all text-fc-fg sm:pb-0">{app.repo_url}</dd>
		<dt class="text-fc-fg-muted">Branch</dt>
		<dd class="pb-3 font-fc-mono text-fc-fg sm:pb-0">{app.branch}</dd>
		<dt class="text-fc-fg-muted">Commit</dt>
		<dd class="pb-3 font-fc-mono text-fc-fg sm:pb-0">{shortCommit(app.commit) || 'Not deployed yet'}</dd>
		<dt class="text-fc-fg-muted">Port</dt>
		<dd class="pb-3 font-fc-mono text-fc-fg sm:pb-0">{app.port}</dd>
		<dt class="text-fc-fg-muted">Domains</dt>
		<dd class="pb-3 break-all text-fc-fg sm:pb-0">{app.domains.length ? app.domains.join(', ') : 'None'}</dd>
		{#if app.tags.length}
			<dt class="text-fc-fg-muted">Tags</dt>
			<dd class="flex flex-wrap gap-1 pb-3 sm:pb-0">
				{#each app.tags as tag (tag)}<Badge>{tag}</Badge>{/each}
			</dd>
		{/if}
		<dt class="text-fc-fg-muted">Created</dt>
		<dd class="text-fc-fg">{ago(app.created_at)}</dd>
	</dl>
</Section>

<Section title="Last deploy" card>
	{#if app.last_deploy}
		{@const last = app.last_deploy}
		<div class="flex flex-wrap items-center justify-between gap-4">
			<Inline gap="tight">
				<Badge tone={deployTone[last.status]}>{deployLabel[last.status]}</Badge>
				<span class="font-fc-mono text-fc-sm text-fc-fg">{shortCommit(last.commit)}</span>
				<span class="text-fc-sm text-fc-fg-muted">{ago(last.started_at)}, by {last.trigger}</span>
			</Inline>
			<Button size="lg" variant="secondary" href={deployHref(last)}>Open log</Button>
		</div>
		{#if last.error}
			<p class="font-fc-mono text-fc-sm break-words text-fc-danger">{last.error}</p>
		{/if}
	{:else}
		<p class="text-fc-sm text-fc-fg-muted">Never deployed. Deploy it to start it for the first time.</p>
	{/if}
</Section>
