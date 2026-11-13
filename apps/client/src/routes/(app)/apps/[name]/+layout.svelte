<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { Alert, Badge, Inline, Page, PageHeader, Skeleton, StatusDot, Tabs, icons } from '@facile/muse';
	import { poll } from '$lib/actions';
	import { provideCurrentApp } from '$lib/current-app.svelte';
	import { appLabel, appTone, isTransient, workloadLabel } from '$lib/status';

	let { children }: { children: Snippet } = $props();

	const current = provideCurrentApp();
	const name = $derived(page.params.name ?? '');
	const base = $derived(`/apps/${encodeURIComponent(name)}`);

	const tabs = $derived([
		{ id: 'overview', label: 'Overview', href: base, icon: icons.dashboard },
		{ id: 'deploys', label: 'Deploys', href: `${base}/deploys`, icon: icons.history },
		{ id: 'logs', label: 'Logs', href: `${base}/logs`, icon: icons.text },
		{ id: 'env', label: 'Environment', href: `${base}/env`, icon: icons.key },
		{ id: 'domains', label: 'Domains', href: `${base}/domains`, icon: icons.globe },
		{ id: 'settings', label: 'Settings', href: `${base}/settings`, icon: icons.settings }
	]);
	const active = $derived(page.url.pathname.slice(base.length).split('/')[1] || 'overview');

	$effect(() => {
		void current.open(name);
		return poll(10_000, () => void current.refresh());
	});
</script>

<svelte:head>
	<title>{name} · Gare</title>
</svelte:head>

<Page width="lg" gap="content">
	<PageHeader title={name} description={current.app ? `${current.app.repo_url} on ${current.app.branch}` : undefined} back={{ href: '/', label: 'Apps' }} />

	{#if current.app}
		<Inline gap="tight">
			<StatusDot tone={appTone[current.app.status]} label={appLabel[current.app.status]} pulse={isTransient(current.app.status)} />
			<Badge>{workloadLabel[current.app.type]}</Badge>
		</Inline>
	{/if}

	<Tabs items={tabs} value={active} label="App sections" />

	{#if current.error}
		<Alert tone="danger" title="Could not load {name}">{current.error}</Alert>
	{/if}

	{#if current.app}
		{@render children()}
	{:else if !current.error}
		<Skeleton class="h-48 rounded-fc-lg" />
	{/if}
</Page>
