<script lang="ts">
	import { Button, DownloadSection, LandingFooter, LandingHeader, provideLocale } from '@facile/muse';
	import { logoIcon } from '$lib/brand';
	import { CHECKSUMS_URL, DOWNLOADS, RELEASES_URL, installCommands } from '$lib/downloads';
	import { bindLocale, localeSource, rememberLocale, t } from '$lib/i18n/index.svelte';

	const i18n = provideLocale(localeSource);
	bindLocale(() => i18n.locale);
</script>

<svelte:head>
	<title>{t('downloads.title')}</title>
</svelte:head>

<div class="flex min-h-dvh flex-col overflow-x-clip bg-fc-page text-fc-fg">
	<LandingHeader appName="Gare" icon={logoIcon} homeHref="/" skipHref="#main">
		{#snippet actions()}
			<Button href="/api/auth/oidc" variant="ghost" size="sm">{t('landing.header.console')}</Button>
		{/snippet}
	</LandingHeader>

	<main id="main" class="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-8 px-4 py-16 sm:px-6 sm:py-20">
		<div class="text-center">
			<h1 class="font-fc-title text-fc-3xl font-bold tracking-tight">{t('downloads.heading')}</h1>
			<p class="mx-auto mt-3 max-w-md text-fc-sm text-fc-fg-muted">{t('landing.install.text')}</p>
		</div>
		<DownloadSection
			downloads={DOWNLOADS}
			commands={installCommands()}
			releasesHref={RELEASES_URL}
			checksumsHref={CHECKSUMS_URL}
			headingLevel={2}
		/>
	</main>

	<LandingFooter
		appName="Gare"
		icon={logoIcon}
		tagline={t('landing.footer.tagline')}
		legal={{ onLocaleChange: rememberLocale }}
	/>
</div>
