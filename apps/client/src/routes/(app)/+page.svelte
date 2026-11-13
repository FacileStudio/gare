<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Alert, Button, EmptyState, Inline, Page, PageHeader, Skeleton, icons } from '@facile/muse';
	import { api, type App } from '$lib/api';
	import { poll, startDeploy } from '$lib/actions';
	import { errorText } from '$lib/status';
	import AppCard from '$lib/components/AppCard.svelte';

	let apps = $state<App[] | null>(null);
	let failure = $state<string | null>(null);
	let deploying = $state<string | null>(null);

	const tag = $derived(page.url.searchParams.get('tag'));
	const tags = $derived([...new Set((apps ?? []).flatMap((app) => app.tags))].sort());
	const shown = $derived((apps ?? []).filter((app) => !tag || app.tags.includes(tag)));

	/** Reloads the list, keeping the last good one on screen when the reload fails. */
	async function load(): Promise<void> {
		try {
			apps = (await api.apps()).sort((a, b) => a.name.localeCompare(b.name));
			failure = null;
		} catch (error) {
			failure = errorText(error);
		}
	}

	/** Puts the tag filter in the URL, so a filtered list survives a reload and can be shared. */
	function filter(next: string | null): void {
		const url = new URL(page.url);
		if (next) url.searchParams.set('tag', next);
		else url.searchParams.delete('tag');
		void goto(url, { replaceState: true, keepFocus: true, noScroll: true });
	}

	/** Starts one deploy; the button stays busy until the log page opens or the API refuses. */
	async function deploy(name: string): Promise<void> {
		deploying = name;
		await startDeploy(name);
		deploying = null;
	}

	$effect(() => {
		void load();
		return poll(10_000, () => void load());
	});
</script>

<svelte:head>
	<title>Apps · Gare</title>
</svelte:head>

<Page width="xl">
	<PageHeader title="Apps" description="Everything this server deploys, and how it is doing.">
		{#snippet actions()}
			<Button size="lg" icon={icons.plus} href="/new">New app</Button>
		{/snippet}
	</PageHeader>

	{#if failure}
		<Alert tone="danger" title="Could not load the apps">{failure}</Alert>
	{/if}

	{#if apps === null}
		<div class="grid gap-4 lg:grid-cols-2">
			<Skeleton class="h-56 rounded-fc-lg" />
			<Skeleton class="h-56 rounded-fc-lg" />
		</div>
	{:else if apps.length === 0}
		<EmptyState icon={icons.server} title="No apps yet" description="Point Gare at a repository to deploy it.">
			<Button size="lg" icon={icons.plus} href="/new">New app</Button>
		</EmptyState>
	{:else}
		{#if tags.length > 0}
			<Inline gap="tight" role="group" aria-label="Filter by tag">
				<Button size="lg" variant={tag ? 'secondary' : 'primary'} aria-pressed={!tag} onclick={() => filter(null)}>
					All
				</Button>
				{#each tags as name (name)}
					<Button
						size="lg"
						variant={tag === name ? 'primary' : 'secondary'}
						aria-pressed={tag === name}
						onclick={() => filter(tag === name ? null : name)}
					>
						{name}
					</Button>
				{/each}
			</Inline>
		{/if}

		{#if shown.length === 0}
			<EmptyState icon={icons.filter} title="No app has this tag">
				<Button size="lg" variant="secondary" onclick={() => filter(null)}>Show all apps</Button>
			</EmptyState>
		{:else}
			<div class="grid gap-4 lg:grid-cols-2">
				{#each shown as app (app.name)}
					<AppCard {app} deploying={deploying === app.name} onDeploy={() => deploy(app.name)} />
				{/each}
			</div>
		{/if}
	{/if}
</Page>
