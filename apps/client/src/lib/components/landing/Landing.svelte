<script lang="ts">
	import { Button, DownloadSection, Icon, LandingFooter, LandingHeader, SignInWithFacile, icons } from '@facile/muse';
	import { logoIcon } from '$lib/brand';
	import { CHECKSUMS_URL, DOWNLOADS, INSTALL_COMMANDS, RELEASES_URL } from '$lib/downloads';
	import DeployDemo from './DeployDemo.svelte';
	import DomainTile from './DomainTile.svelte';
	import PushTile from './PushTile.svelte';
	import RollbackTile from './RollbackTile.svelte';

	const SIGN_IN = '/api/auth/oidc';

	const features = [
		{ icon: icons.bolt, title: 'Push to deploy', body: 'A signed GitHub or GitLab webhook pulls, builds and restarts the app.' },
		{ icon: icons.history, title: 'Rolls back by itself', body: 'When a health check fails, the last good build runs again.' },
		{ icon: icons.shield, title: 'Rootless, no daemon', body: 'Each app is a systemd user unit running rootless Podman.' },
		{ icon: icons.globe, title: 'Domains with HTTPS', body: 'Add a hostname and Caddy routes it with a certificate.' }
	];

	const links = [
		{ href: '#features', label: 'Features' },
		{ href: '#download', label: 'Download' }
	];

	const columns = [
		{
			title: 'Gare',
			links: [
				{ href: '#features', label: 'Features' },
				{ href: '/downloads', label: 'Downloads' },
				{ href: 'https://github.com/FacileStudio/gare', label: 'Source code' }
			]
		}
	];
</script>

<svelte:head>
	<title>Gare · Deploy from git to your own server</title>
</svelte:head>

<div class="flex min-h-dvh flex-col overflow-x-clip bg-fc-page text-fc-fg">
	<LandingHeader appName="Gare" icon={logoIcon} {links} skipHref="#main">
		{#snippet actions()}
			<SignInWithFacile href={SIGN_IN} />
		{/snippet}
	</LandingHeader>

	<main id="main" class="flex-1">
		<section class="mx-auto max-w-5xl px-4 pt-20 pb-12 text-center sm:px-6 sm:pt-24">
			<h1 class="font-fc-title text-fc-3xl font-bold tracking-tight sm:text-5xl lg:text-6xl">
				<span class="block">Deploy from git</span>
				<span class="block">to your own server</span>
			</h1>
			<p class="mx-auto mt-6 max-w-xl text-fc-md text-fc-fg-muted">
				Push to git. Gare builds the app, runs it with Podman and checks its health.
			</p>
			<div class="mt-10 flex flex-wrap items-center justify-center gap-3">
				<SignInWithFacile href={SIGN_IN} size="lg" />
				<Button href="#download" variant="secondary" size="lg" icon={icons.download}>Install the CLI</Button>
			</div>
		</section>

		<section class="mx-auto max-w-5xl px-4 pb-8 sm:px-6">
			<DeployDemo />
		</section>

		<section class="relative mx-auto max-w-5xl px-4 py-16 sm:px-6 sm:py-20">
			<h2 class="mb-10 text-center font-fc-title text-2xl font-bold tracking-tight text-fc-fg sm:text-3xl">
				Everything after git push
			</h2>
			<div class="grid gap-6 md:grid-cols-3">
				<PushTile />
				<RollbackTile />
				<DomainTile />
			</div>
		</section>

		<section id="features" class="mx-auto grid max-w-5xl scroll-mt-20 gap-10 px-4 py-16 sm:px-6 md:grid-cols-[2fr_3fr] md:items-center md:gap-16 md:py-20">
			<div>
				<h2 class="font-fc-title text-2xl font-bold tracking-tight text-fc-fg sm:text-3xl">Plain tools on your server</h2>
				<p class="mt-4 text-fc-sm text-fc-fg-muted">Podman, systemd and Caddy do the work. Gare wires them together.</p>
			</div>
			<ul class="flex flex-col gap-6">
				{#each features as feature (feature.title)}
					<li class="flex items-start gap-4">
						<div class="flex size-10 shrink-0 items-center justify-center rounded-fc-md bg-fc-component" aria-hidden="true">
							<Icon icon={feature.icon} size={20} />
						</div>
						<div>
							<h3 class="text-fc-md font-bold">{feature.title}</h3>
							<p class="mt-1 text-fc-sm text-fc-fg-muted">{feature.body}</p>
						</div>
					</li>
				{/each}
			</ul>
		</section>

		<section id="download" class="mx-auto flex max-w-5xl scroll-mt-20 flex-col gap-8 px-4 py-20 sm:px-6">
			<div class="text-center">
				<h2 class="font-fc-title text-fc-2xl font-bold tracking-tight">Install Gare</h2>
				<p class="mx-auto mt-3 max-w-md text-fc-sm text-fc-fg-muted">
					One binary for Linux. It needs Podman 5, systemd, Caddy and git.
				</p>
			</div>
			<DownloadSection
				mode="device"
				allHref="/downloads"
				downloads={DOWNLOADS}
				commands={INSTALL_COMMANDS}
				releasesHref={RELEASES_URL}
				checksumsHref={CHECKSUMS_URL}
				headingLevel={3}
			/>
		</section>

		<section class="mx-auto max-w-5xl px-4 py-20 text-center sm:px-6">
			<h2 class="font-fc-title text-fc-2xl font-bold tracking-tight">Open your server's console</h2>
			<p class="mt-4 text-fc-sm text-fc-fg-muted">Sign in to see every app, its deploys and its logs.</p>
			<div class="mt-8 flex justify-center">
				<SignInWithFacile href={SIGN_IN} size="lg" />
			</div>
		</section>
	</main>

	<LandingFooter appName="Gare" icon={logoIcon} tagline="Deploy and run your apps on this server." {columns} />
</div>
