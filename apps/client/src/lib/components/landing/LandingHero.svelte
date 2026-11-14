<script lang="ts">
	import { Button, icons } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';

	let { signInHref }: { signInHref: string } = $props();
</script>

<section class="mx-auto max-w-5xl px-4 pt-20 pb-12 text-center sm:px-6 sm:pt-24">
	<h1 class="font-fc-title text-fc-3xl font-bold tracking-tight sm:text-5xl lg:text-6xl">
		<span class="block">{t('landing.hero.line1')}</span>
		<span class="block">{t('landing.hero.line2')}</span>
	</h1>
	<p class="mx-auto mt-6 max-w-xl text-fc-md text-fc-fg-muted">{t('landing.description')}</p>
	<div class="mt-10 flex flex-col items-center gap-4">
		<Button href="#download" size="lg" icon={icons.download} class="w-full sm:w-auto">{t('landing.hero.install')}</Button>
		<a
			href={signInHref}
			class="rounded-fc-sm text-fc-sm text-fc-fg-muted underline underline-offset-4 hover:text-fc-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
		>
			{t('landing.hero.console')}
		</a>
	</div>
</section>
