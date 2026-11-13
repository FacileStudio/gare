<script lang="ts">
	import { Alert, Button, ConfirmModal, EmptyState, Field, Input, Section, Skeleton, icons, toast } from '@facile/muse';
	import { api } from '$lib/api';
	import { currentApp } from '$lib/current-app.svelte';
	import { errorText } from '$lib/status';

	const HOST = /^(?=.{1,253}$)([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$/;

	const current = currentApp();

	let domains = $state<string[] | null>(null);
	let failure = $state<string | null>(null);
	let hostname = $state('');
	let tried = $state(false);
	let busy = $state(false);
	let removing = $state<string | null>(null);

	const host = $derived(hostname.trim().toLowerCase().replace(/^https?:\/\//, '').replace(/\/.*$/, ''));
	const error = $derived(
		!HOST.test(host) ? 'A hostname such as app.example.com, without a path.' : domains?.includes(host) ? 'This app already answers on it.' : ''
	);

	/** Loads the hostnames the app answers on. */
	async function load(): Promise<void> {
		try {
			domains = await api.domains(current.name);
			failure = null;
		} catch (error) {
			failure = errorText(error);
		}
	}

	/** Adds a hostname; the API answers only once Caddy serves it. */
	async function add(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		tried = true;
		if (error) return;
		busy = true;
		try {
			domains = await api.addDomain(current.name, host);
			toast.success(`${host} now routes to ${current.name}.`);
			hostname = '';
			tried = false;
			void current.refresh();
		} catch (failed) {
			toast.danger(errorText(failed), { title: `Could not add ${host}` });
		} finally {
			busy = false;
		}
	}

	/** Removes the hostname the person confirmed. */
	async function remove(): Promise<void> {
		const target = removing;
		removing = null;
		if (!target) return;
		try {
			domains = await api.removeDomain(current.name, target);
			toast.success(`${target} removed.`);
			void current.refresh();
		} catch (failed) {
			toast.danger(errorText(failed), { title: `Could not remove ${target}` });
		}
	}

	$effect(() => {
		void current.name;
		void load();
	});
</script>

<Section title="Domains" description="Caddy routes each hostname to port {current.app?.port} and gets its certificate.">
	{#if failure}
		<Alert tone="danger" title="Could not load the domains">{failure}</Alert>
	{/if}

	{#if domains === null}
		<Skeleton class="h-24 rounded-fc-lg" />
	{:else if domains.length === 0}
		<EmptyState icon={icons.globe} title="No domains" description="The app answers on its port only." />
	{:else}
		<ul class="flex flex-col rounded-fc-lg bg-fc-component px-5 py-1">
			{#each domains as domain (domain)}
				<li class="flex flex-wrap items-center justify-between gap-2 py-3">
					<a href="https://{domain}" target="_blank" rel="noreferrer" class="inline-flex min-h-11 min-w-0 items-center break-all text-fc-sm text-fc-fg underline underline-offset-4 focus-visible:outline-2 focus-visible:outline-fc-ring">
						{domain}
					</a>
					<Button size="lg" variant="ghost-danger" icon={icons.remove} onclick={() => (removing = domain)} aria-label="Remove {domain}">Remove</Button>
				</li>
			{/each}
		</ul>
	{/if}
</Section>

<Section title="Add a domain" description="Point its DNS at this server first." card>
	<form class="flex flex-col gap-4" onsubmit={add} novalidate>
		<Field label="Hostname" error={tried ? error : ''}>
			{#snippet children({ id, describedBy })}
				<Input {id} aria-describedby={describedBy} aria-invalid={tried && !!error} inputmode="url" autocomplete="off" autocapitalize="off" spellcheck="false" placeholder="app.example.com" bind:value={hostname} />
			{/snippet}
		</Field>
		<div>
			<Button size="lg" type="submit" icon={icons.plus} disabled={busy}>{busy ? 'Adding' : 'Add domain'}</Button>
		</div>
	</form>
</Section>

<ConfirmModal
	open={removing !== null}
	title="Remove {removing}?"
	description="Caddy stops routing this hostname to {current.name}."
	confirmLabel="Remove"
	tone="danger"
	icon={icons.remove}
	onConfirm={remove}
	onCancel={() => (removing = null)}
/>
