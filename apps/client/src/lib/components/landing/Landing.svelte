<script lang="ts">
	import {
		Button,
		DownloadSection,
		FeatureSplit,
		ForDevelopers,
		LANDING_HEADING,
		LandingFooter,
		LandingHeader,
		SignInWithFacile,
		WorksWith,
		icons,
		provideLocale,
		type DeveloperRow,
		type FeatureItem,
		type WorksWithApp
	} from '@facile/muse';
	import { antenneIcon, logoIcon } from '$lib/brand';
	import { CHECKSUMS_URL, DOWNLOADS, RELEASES_URL, installCommands } from '$lib/downloads';
	import { bindLocale, localeSource, rememberLocale, t } from '$lib/i18n/index.svelte';
	import DeployDemo from './DeployDemo.svelte';
	import DomainTile from './DomainTile.svelte';
	import LandingHero from './LandingHero.svelte';
	import LandingPlan from './LandingPlan.svelte';
	import PushTile from './PushTile.svelte';
	import RollbackTile from './RollbackTile.svelte';

	const SIGN_IN = '/api/auth/oidc';
	const SOURCE = 'https://github.com/FacileStudio/gare';
	const WIDTH = 'mx-auto w-full max-w-5xl px-4 sm:px-6';

	const i18n = provideLocale(localeSource);
	bindLocale(() => i18n.locale);

	const features: FeatureItem[] = $derived([
		{ icon: icons.layers, title: t('landing.feature.kinds.title'), text: t('landing.feature.kinds.text') },
		{ icon: icons.history, title: t('landing.feature.history.title'), text: t('landing.feature.history.text') },
		{ icon: icons.key, title: t('landing.feature.env.title'), text: t('landing.feature.env.text') },
		{ icon: icons.server, title: t('landing.feature.rootless.title'), text: t('landing.feature.rootless.text') }
	]);

	const connections: WorksWithApp[] = $derived([
		{ name: 'Antenne', icon: antenneIcon, line: t('landing.with.antenne'), href: 'https://antenne.facile.studio' }
	]);

	const developers: DeveloperRow[] = $derived([
		{ kind: 'cli', text: t('landing.dev.cli'), command: 'gare deploy shop' },
		{ kind: 'source', text: t('landing.dev.source'), href: SOURCE }
	]);

	const links = $derived([
		{ href: '#features', label: t('landing.nav.features') },
		{ href: '#download', label: t('landing.nav.install') }
	]);

	const columns = $derived([
		{
			title: 'Gare',
			links: [
				{ href: '#features', label: t('landing.nav.features') },
				{ href: '/downloads', label: t('landing.footer.downloads') },
				{ href: SOURCE, label: t('landing.footer.source') },
				{ href: 'https://suite.facile.studio/', label: t('landing.footer.suite') }
			]
		}
	]);
</script>

<svelte:head>
	<title>{t('landing.title')}</title>
	<meta name="description" content={t('landing.description')} />
</svelte:head>

<div class="flex min-h-dvh flex-col overflow-x-clip bg-fc-page text-fc-fg">
	<LandingHeader appName="Gare" icon={logoIcon} {links} skipHref="#main">
		{#snippet actions()}
			<Button href={SIGN_IN} variant="ghost" size="sm">{t('landing.header.console')}</Button>
		{/snippet}
	</LandingHeader>

	<main id="main" class="flex-1">
		<LandingHero signInHref={SIGN_IN} />

		<section class="{WIDTH} pb-8">
			<DeployDemo />
		</section>

		<section class="relative {WIDTH} py-16 sm:py-20">
			<h2 class="mb-10 text-center {LANDING_HEADING}">{t('landing.tiles.title')}</h2>
			<div class="grid gap-6 md:grid-cols-3">
				<PushTile />
				<RollbackTile />
				<DomainTile />
			</div>
		</section>

		<FeatureSplit
			id="features"
			class="{WIDTH} scroll-mt-20 py-16 md:py-20"
			title={t('landing.features.title')}
			description={t('landing.features.line')}
			{features}
		/>

		<WorksWith class="{WIDTH} py-16" apps={connections} />

		<section id="download" class="{WIDTH} flex scroll-mt-20 flex-col gap-8 py-20">
			<div class="text-center">
				<h2 class={LANDING_HEADING}>{t('landing.install.title')}</h2>
				<p class="mx-auto mt-3 max-w-md text-fc-sm text-fc-fg-muted">{t('landing.install.text')}</p>
			</div>
			<DownloadSection
				mode="device"
				allHref="/downloads"
				downloads={DOWNLOADS}
				commands={installCommands()}
				releasesHref={RELEASES_URL}
				checksumsHref={CHECKSUMS_URL}
				headingLevel={3}
			/>
		</section>

		<LandingPlan />

		<div class="{WIDTH} py-8">
			<ForDevelopers rows={developers} description={t('landing.dev.description')} />
		</div>

		<section class="{WIDTH} py-20 text-center">
			<h2 class={LANDING_HEADING}>{t('landing.cta.title')}</h2>
			<p class="mx-auto mt-4 max-w-lg text-fc-sm text-fc-fg-muted">{t('landing.cta.text')}</p>
			<div class="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
				<Button href="#download" size="lg" icon={icons.download} class="w-full sm:w-auto">{t('landing.hero.install')}</Button>
				<SignInWithFacile href={SIGN_IN} size="lg" variant="secondary" />
			</div>
		</section>
	</main>

	<LandingFooter
		appName="Gare"
		icon={logoIcon}
		tagline={t('landing.footer.tagline')}
		{columns}
		legal={{ onLocaleChange: rememberLocale }}
	/>
</div>
