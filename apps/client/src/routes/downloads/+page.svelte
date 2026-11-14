<script lang="ts">
	import { DownloadSection, LandingFooter, LandingHeader, SignInWithFacile } from '@facile/muse';
	import { logoIcon } from '$lib/brand';
	import { CHECKSUMS_URL, DOWNLOADS, INSTALL_COMMANDS, RELEASES_URL } from '$lib/downloads';
</script>

<svelte:head>
	<title>Downloads · Gare</title>
</svelte:head>

<div class="flex min-h-dvh flex-col overflow-x-clip bg-fc-page text-fc-fg">
	<LandingHeader appName="Gare" icon={logoIcon} homeHref="/" skipHref="#main">
		{#snippet actions()}
			<SignInWithFacile href="/api/auth/oidc" />
		{/snippet}
	</LandingHeader>

	<main id="main" class="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-8 px-4 py-16 sm:px-6 sm:py-20">
		<div class="text-center">
			<h1 class="font-fc-title text-fc-3xl font-bold tracking-tight">Downloads</h1>
			<p class="mx-auto mt-3 max-w-md text-fc-sm text-fc-fg-muted">
				One binary for Linux. It needs Podman 5, systemd, Caddy and git.
			</p>
		</div>
		<DownloadSection
			downloads={DOWNLOADS}
			commands={INSTALL_COMMANDS}
			releasesHref={RELEASES_URL}
			checksumsHref={CHECKSUMS_URL}
			headingLevel={2}
		/>
	</main>

	<LandingFooter appName="Gare" icon={logoIcon} tagline="Deploy and run your apps on this server." />
</div>
