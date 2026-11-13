<script lang="ts">
	import { logoIcon } from '$lib/brand';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Alert, AuthScreen, Icon } from '@facile/muse';
	import { session } from '$lib/session';

	const failed = $derived(page.url.searchParams.has('error'));

	/** Only a same-site path is followed after sign-in, so `?next=` cannot send anyone off-site. */
	function nextPath(): string {
		const next = page.url.searchParams.get('next') ?? '/';
		return next.startsWith('/') && !next.startsWith('//') ? next : '/';
	}

	$effect(() => {
		session
			.refresh()
			.then((user) => {
				if (user) void goto(nextPath(), { replaceState: true });
			})
			.catch(() => undefined);
	});
</script>

<svelte:head>
	<title>Sign in · Gare</title>
</svelte:head>

<AuthScreen appName="Gare" href="/api/auth/oidc" signUpHref={null}>
	{#snippet logo()}
		<Icon icon={logoIcon} size={40} class="text-fc-fg" />
	{/snippet}
	{#snippet tagline()}
		Deploy and run your apps on this server.
	{/snippet}
	{#snippet notice()}
		{#if failed}
			<Alert tone="danger">Sign-in failed, or this account is not allowed to use Gare.</Alert>
		{/if}
	{/snippet}
</AuthScreen>
