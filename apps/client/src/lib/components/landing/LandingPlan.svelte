<script lang="ts">
	import { Button, LANDING_HEADING, icons } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';

	const SUITE = 'https://suite.facile.studio/';
</script>

<section id="pricing" class="mx-auto max-w-5xl scroll-mt-20 px-4 py-16 sm:px-6 sm:py-20">
	<div class="grid gap-8 md:grid-cols-2 md:items-center md:gap-12">
		<div>
			<h2 class={LANDING_HEADING}>{t('landing.plan.title')}</h2>
			<p class="mt-4 text-fc-md text-fc-fg">{t('landing.plan.note')}</p>
		</div>
		<div class="flex flex-col items-start gap-4">
			<p class="text-fc-sm text-fc-fg-muted">{t('landing.plan.suite')}</p>
			<Button href={SUITE} variant="secondary" icon={icons.external}>{t('landing.plan.suiteLink')}</Button>
		</div>
	</div>
</section>
