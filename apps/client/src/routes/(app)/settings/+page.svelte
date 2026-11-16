<script lang="ts">
	import { Page, PageHeader, PreferencesSection, type Theme } from '@facile/muse';
	import { theme } from '$lib/session';
</script>

<svelte:head>
	<title>Settings · Gare</title>
</svelte:head>

<Page width="md">
	<PageHeader title="Settings" back={{ href: '/', label: 'Apps' }} />

	<PreferencesSection showLanguage={false} theme={theme.mode} onThemeChange={(next: Theme) => theme.set(next)} />
</Page>
