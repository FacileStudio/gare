<script lang="ts">
	import { goto } from '$app/navigation';
	import { Alert, Button, Field, Input, Page, PageHeader, Section, Select, icons, toast } from '@facile/muse';
	import { api, type NewApp, type WorkloadType } from '$lib/api';
	import { errorText } from '$lib/status';

	const NAME = /^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$/;

	const types = [
		{ value: '', label: 'Read it from gare.yml' },
		{ value: 'container', label: 'Container' },
		{ value: 'static', label: 'Static' },
		{ value: 'compose', label: 'Compose' }
	];

	let repo = $state('');
	let name = $state('');
	let branch = $state('');
	let type = $state('');
	let port = $state('');
	let saving = $state(false);
	let failure = $state<string | null>(null);
	let tried = $state(false);

	const errors = $derived({
		repo: !repo.trim() ? 'A repository URL is required.' : repo.trim().startsWith('-') ? 'A URL cannot start with "-".' : '',
		name: name.trim() && !NAME.test(name.trim()) ? 'Letters, digits, "-" and "_", starting with a letter or digit, 63 at most.' : '',
		branch: branch.trim().startsWith('-') ? 'A branch cannot start with "-".' : '',
		port: port.trim() && !(Number.isInteger(Number(port)) && Number(port) > 0 && Number(port) < 65536) ? 'A port is a whole number from 1 to 65535.' : ''
	});
	const valid = $derived(Object.values(errors).every((message) => !message));

	/** Builds the request body, leaving out every field the person left empty so the API applies its defaults. */
	function body(): NewApp {
		return {
			repo_url: repo.trim(),
			...(name.trim() && { name: name.trim() }),
			...(branch.trim() && { branch: branch.trim() }),
			...(type && { type: type as WorkloadType }),
			...(port.trim() && { port: Number(port) })
		};
	}

	/** Creates the app, then opens it so the first deploy is one tap away. */
	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		tried = true;
		if (!valid) return;
		saving = true;
		failure = null;
		try {
			const app = await api.createApp(body());
			toast.success(`${app.name} is ready to deploy.`);
			await goto(`/apps/${encodeURIComponent(app.name)}`);
		} catch (error) {
			failure = errorText(error);
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>New app · Gare</title>
</svelte:head>

<Page width="md">
	<PageHeader title="New app" description="Gare clones the repository and reads its gare.yml." back={{ href: '/', label: 'Apps' }} />

	<form onsubmit={submit} novalidate>
		<Section card>
			<Field label="Repository URL" error={tried ? errors.repo : ''}>
				{#snippet children({ id, describedBy })}
					<Input {id} aria-describedby={describedBy} aria-invalid={tried && !!errors.repo} type="url" inputmode="url" autocomplete="off" placeholder="https://github.com/acme/site.git" bind:value={repo} />
				{/snippet}
			</Field>
			<Field label="Name" helper="Optional. Defaults to the repository name." error={tried ? errors.name : ''}>
				{#snippet children({ id, describedBy })}
					<Input {id} aria-describedby={describedBy} aria-invalid={tried && !!errors.name} autocomplete="off" autocapitalize="off" spellcheck="false" bind:value={name} />
				{/snippet}
			</Field>
			<Field label="Branch" helper="Optional. Defaults to the repository's default branch." error={tried ? errors.branch : ''}>
				{#snippet children({ id, describedBy })}
					<Input {id} aria-describedby={describedBy} aria-invalid={tried && !!errors.branch} autocomplete="off" autocapitalize="off" spellcheck="false" placeholder="main" bind:value={branch} />
				{/snippet}
			</Field>
			<Field label="Workload type" helper="gare.yml decides when you leave this alone.">
				{#snippet children({ id, describedBy })}
					<Select {id} aria-describedby={describedBy} options={types} bind:value={type} />
				{/snippet}
			</Field>
			<Field label="Port" helper="Optional. Gare picks a free one from 8000 up." error={tried ? errors.port : ''}>
				{#snippet children({ id, describedBy })}
					<Input {id} aria-describedby={describedBy} aria-invalid={tried && !!errors.port} inputmode="numeric" autocomplete="off" bind:value={port} />
				{/snippet}
			</Field>

			{#if failure}
				<Alert tone="danger" title="The app was not created">{failure}</Alert>
			{/if}

			<div class="flex flex-wrap gap-2">
				<Button size="lg" type="submit" icon={icons.plus} disabled={saving}>{saving ? 'Creating' : 'Create app'}</Button>
				<Button size="lg" variant="ghost" href="/">Cancel</Button>
			</div>
		</Section>
	</form>
</Page>
