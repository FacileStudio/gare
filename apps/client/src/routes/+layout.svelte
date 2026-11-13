<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Toaster, provideLocale } from '@facile/muse';
	import { theme } from '$lib/session';
	import '../app.css';

	let { children }: { children: Snippet } = $props();

	provideLocale('en');
	theme.restore();
</script>

{@render children()}

<Toaster class="pb-28 md:pb-6" />
