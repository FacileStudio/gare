<script lang="ts">
	import { Alert, Badge, Button, ConfirmModal, EmptyState, Section, Skeleton, icons } from '@facile/muse';
	import { api, type Deploy } from '$lib/api';
	import { deployHref, poll, startDeploy } from '$lib/actions';
	import { currentApp } from '$lib/current-app.svelte';
	import { ago, deployLabel, deployTone, errorText, shortCommit, took } from '$lib/status';

	const current = currentApp();

	let deploys = $state<Deploy[] | null>(null);
	let failure = $state<string | null>(null);
	let target = $state<Deploy | null>(null);

	const canRollBack = $derived(current.app?.type === 'container');
	const live = $derived(deploys?.find((deploy) => deploy.status === 'succeeded')?.id);

	/** Reloads the history, newest first as the API sends it. */
	async function load(): Promise<void> {
		try {
			deploys = await api.deploys(current.name);
			failure = null;
		} catch (error) {
			failure = errorText(error);
		}
	}

	/** Rolls back to the chosen deploy's image and opens the rollback's own log. */
	async function rollBack(): Promise<void> {
		if (target) await startDeploy(current.name, target.id);
		target = null;
	}

	$effect(() => {
		void current.name;
		void load();
		return poll(5_000, () => void load());
	});
</script>

<Section title="Deploys" description={canRollBack ? 'Roll back to any earlier successful deploy.' : 'Rollback is available for container apps only.'}>
	{#if failure}
		<Alert tone="danger" title="Could not load the deploys">{failure}</Alert>
	{/if}

	{#if deploys === null}
		<Skeleton class="h-40 rounded-fc-lg" />
	{:else if deploys.length === 0}
		<EmptyState icon={icons.history} title="No deploys yet" description="Each deploy and its log show up here." />
	{:else}
		<ol class="flex flex-col rounded-fc-lg bg-fc-component px-5 py-1">
			{#each deploys as deploy (deploy.id)}
				<li class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 py-4">
					<div class="flex min-w-0 flex-col gap-1">
						<div class="flex flex-wrap items-center gap-2">
							<Badge tone={deployTone[deploy.status]}>{deployLabel[deploy.status]}</Badge>
							<span class="font-fc-mono text-fc-sm text-fc-fg">{shortCommit(deploy.commit) || 'commit pending'}</span>
							{#if deploy.id === live}<Badge tone="accent">Live</Badge>{/if}
						</div>
						<span class="text-fc-xs text-fc-fg-muted">
							{ago(deploy.started_at)} by {deploy.trigger}{took(deploy) ? `, took ${took(deploy)}` : ''}
						</span>
					</div>
					<div class="flex flex-wrap gap-2">
						<Button size="lg" variant="secondary" href={deployHref(deploy)}>Log</Button>
						{#if canRollBack && deploy.status === 'succeeded' && deploy.id !== live}
							<Button size="lg" variant="ghost" icon={icons.history} onclick={() => (target = deploy)}>Roll back</Button>
						{/if}
					</div>
				</li>
			{/each}
		</ol>
	{/if}
</Section>

<ConfirmModal
	open={target !== null}
	title="Roll back {current.name}?"
	description={target ? `Gare restarts ${current.name} on commit ${shortCommit(target.commit)} from ${ago(target.started_at)}. The repository is not changed.` : ''}
	confirmLabel="Roll back"
	tone="neutral"
	icon={icons.history}
	onConfirm={rollBack}
	onCancel={() => (target = null)}
/>
