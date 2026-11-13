<script lang="ts">
	import { Badge, Button, Card, StatusDot, icons } from '@facile/muse';
	import type { App } from '$lib/api';
	import { appLabel, appTone, isTransient, shortCommit, workloadLabel } from '$lib/status';

	let { app, deploying, onDeploy }: { app: App; deploying: boolean; onDeploy: () => void } = $props();

	const href = $derived(`/apps/${encodeURIComponent(app.name)}`);
</script>

<Card class="flex flex-col gap-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<a
			{href}
			class="inline-flex min-h-11 min-w-0 items-center truncate text-fc-lg font-medium text-fc-fg underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
		>
			{app.name}
		</a>
		<StatusDot tone={appTone[app.status]} label={appLabel[app.status]} pulse={isTransient(app.status)} />
	</div>

	<dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-fc-sm">
		<dt class="text-fc-fg-muted">Type</dt>
		<dd><Badge>{workloadLabel[app.type]}</Badge></dd>
		<dt class="text-fc-fg-muted">Commit</dt>
		<dd class="font-fc-mono text-fc-fg">{shortCommit(app.commit) || 'Not deployed yet'}</dd>
		<dt class="text-fc-fg-muted">Domains</dt>
		<dd class="flex min-w-0 flex-wrap gap-x-3">
			{#each app.domains as domain (domain)}
				<a
					href="https://{domain}"
					target="_blank"
					rel="noreferrer"
					class="truncate text-fc-fg underline underline-offset-4 focus-visible:outline-2 focus-visible:outline-fc-ring"
				>
					{domain}
				</a>
			{:else}
				<span class="text-fc-fg-muted">Port {app.port} only</span>
			{/each}
		</dd>
		{#if app.tags.length > 0}
			<dt class="text-fc-fg-muted">Tags</dt>
			<dd class="flex flex-wrap gap-1">
				{#each app.tags as tag (tag)}
					<Badge>{tag}</Badge>
				{/each}
			</dd>
		{/if}
	</dl>

	<div class="flex flex-wrap gap-2">
		<Button size="lg" icon={icons.refresh} disabled={deploying} onclick={onDeploy}>
			{deploying ? 'Starting deploy' : 'Deploy'}
		</Button>
		<Button size="lg" variant="ghost" {href}>Details</Button>
	</div>
</Card>
