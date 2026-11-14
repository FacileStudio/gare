<script lang="ts">
	import { Badge, Button, Icon, Input, icons } from '@facile/muse';
	import { Stage, Tile, grow, type Act } from '@facile/mise-en-scene';
	import { t } from '$lib/i18n/index.svelte';

	const HOST = 'www.example.com';

	let typed = $state('');
	let busy = $state(false);
	let added = $state(false);

	function reset() {
		typed = '';
		busy = false;
		added = false;
	}

	function poster() {
		added = true;
	}

	async function script(act: Act) {
		await act.wait(700);
		await act.type(HOST, (v) => (typed = v), 70);
		await act.wait(400);
		busy = true;
		await act.wait(1300);
		busy = false;
		typed = '';
		added = true;
		await act.wait(3000);
		reset();
		await act.wait(500);
	}
</script>

{#snippet row(domain: string)}
	<div class="flex items-center justify-between gap-2 py-2">
		<span class="min-w-0 truncate text-fc-xs text-fc-fg underline underline-offset-4">{domain}</span>
		<span class="flex shrink-0 items-center gap-1 text-fc-success"><Icon icon={icons.shield} size={14} /><Badge tone="success">HTTPS</Badge></span>
	</div>
{/snippet}

<Tile title={t('demo.domain.title')} body={t('demo.domain.body')}>
	<Stage width={320} height={240} {script} {reset} {poster}>
		{#snippet children()}
			<div class="panel absolute top-5 right-5 left-5 flex h-[188px] flex-col gap-3 rounded-fc-lg bg-fc-page p-4">
				<div class="flex items-center gap-2">
					<Input value={typed} placeholder="app.example.com" class="h-9 min-w-0 flex-1 text-fc-xs" tabindex={-1} readonly />
					<Button size="sm" icon={icons.plus} disabled={busy}>{busy ? t('demo.adding') : t('demo.add')}</Button>
				</div>
				<div class="flex flex-col rounded-fc-md bg-fc-component px-3">
					{@render row('example.com')}
					{@render row('shop.example.com')}
					{#if added}
						<div in:grow={{ duration: 350 }} out:grow={{ duration: 250 }}>{@render row(HOST)}</div>
					{/if}
				</div>
			</div>
		{/snippet}
	</Stage>
</Tile>

<style>
	.panel {
		box-shadow: 0 14px 28px -14px rgb(0 0 0 / 0.25);
	}
</style>
