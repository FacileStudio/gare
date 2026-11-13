<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Toaster, provideLocale, registerIcons } from '@facile/muse';
	import { theme } from '$lib/session';
	import iconData from '$lib/icons-data.json';
	import '../app.css';

	let { children }: { children: Snippet } = $props();

	registerIcons(iconData);
	provideLocale('en');
	theme.restore();
</script>

{@render children()}

<Toaster class="pb-28 md:pb-6" />
