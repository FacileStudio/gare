<script lang="ts">
	import type { Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { redirectToLogin } from '@facile/client';
	import {
		Alert,
		Button,
		MobileNav,
		PageTransition,
		SideBar,
		Spinner,
		Topbar,
		icons,
		type Theme,
		type UserMenuOptions
	} from '@facile/muse';
	import { session, theme } from '$lib/session';
	import { errorText } from '$lib/status';

	let { children }: { children: Snippet } = $props();

	let collapsed = $state(false);
	let failure = $state<string | null>(null);
	let scroller: HTMLElement | null = $state(null);

	const path = $derived(page.url.pathname);
	const pages = $derived([
		{ href: '/', label: 'Apps', icon: icons.server, active: path === '/' || path.startsWith('/apps/') },
		{ href: '/new', label: 'New app', icon: icons.plus, active: path === '/new' }
	]);

	const user = $derived({
		name: session.user?.name?.trim() || session.user?.email || 'Signed in',
		email: session.user?.email,
		avatar: session.user?.picture ?? undefined
	});

	const userMenu: UserMenuOptions = $derived({
		settingsHref: null,
		helpHref: 'https://github.com/FacileStudio/gare',
		showLanguage: false,
		theme: theme.mode,
		onThemeChange: (next: Theme) => theme.set(next),
		onSignOut: () => void signOut()
	});

	/** Ends the porte session, then shows the sign-in page whatever the server answered. */
	async function signOut(): Promise<void> {
		await session.logout().catch(() => undefined);
		await goto('/login', { replaceState: true });
	}

	/** Reads the session; nobody signed in goes to the sign-in page, a broken API shows why. */
	async function load(): Promise<void> {
		failure = null;
		try {
			if (!(await session.refresh())) redirectToLogin();
		} catch (error) {
			failure = errorText(error);
		}
	}

	$effect(() => {
		void load();
	});

	$effect(() => {
		if (path) scroller?.scrollTo({ top: 0 });
	});
</script>

<div class="flex h-dvh w-full overflow-hidden bg-fc-page">
	<div class="hidden h-full shrink-0 p-3 md:block">
		<SideBar icon={icons.server} title="Gare" bind:collapsed {pages} {user} {userMenu} class="h-full" />
	</div>

	<main bind:this={scroller} class="min-w-0 flex-1 overflow-auto overscroll-contain pb-28 md:pb-0">
		<Topbar class="md:hidden">
			<span class="text-fc-md font-semibold text-fc-fg">Gare</span>
		</Topbar>

		{#if failure}
			<div class="mx-auto flex max-w-fc-lg flex-col items-start gap-4 px-4 py-8">
				<Alert tone="danger" title="Gare's API did not answer">{failure}</Alert>
				<Button size="lg" variant="secondary" icon={icons.refresh} onclick={load}>Try again</Button>
			</div>
		{:else if session.loading || !session.user}
			<div class="flex justify-center py-20">
				<Spinner label="Loading" />
			</div>
		{:else}
			<PageTransition key={path.split('/').slice(0, 3).join('/')}>
				{@render children()}
			</PageTransition>
		{/if}
	</main>

	<MobileNav items={pages} {user} {userMenu} />
</div>
