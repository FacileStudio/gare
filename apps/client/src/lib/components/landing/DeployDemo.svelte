<script lang="ts">
	import { tick } from 'svelte';
	import { fade } from 'svelte/transition';
	import { Badge, Button, Icon, StatusDot, icons } from '@facile/muse';
	import { Bar, Cursor, Roll, Stage, Terminal, Window, cursorAt, type Act } from '@facile/mise-en-scene';
	import { logoIcon } from '$lib/brand';

	type Line = { mark: '▸' | '✓'; text: string };

	const OLD = 'a41c9e2';
	const NEW = '4f2c1a9';
	const LOG: Line[] = [
		{ mark: '▸', text: 'Syncing shop with its git remote...' },
		{ mark: '▸', text: `Building container image localhost/shop:${NEW}...` },
		{ mark: '✓', text: 'Synced manifest.yaml from repository' },
		{ mark: '▸', text: 'Verifying health probe port 8003 at http://127.0.0.1:8003/...' },
		{ mark: '✓', text: '1 health probe(s) passed' },
		{ mark: '✓', text: `Successfully deployed container app shop (${NEW}) -> shop.example.com` }
	];
	const OTHERS = [
		{ name: 'blog', type: 'Static', widths: [54, 40] },
		{ name: 'api', type: 'Compose', widths: [46, 58] }
	];

	let deploying = $state(false);
	let starting = $state(false);
	let commit = $state(OLD);
	let lines = $state<Line[]>([]);
	let cursor = $state(cursorAt());
	let button = $state<HTMLElement>();

	function reset() {
		deploying = false;
		starting = false;
		commit = OLD;
		lines = [];
		cursor.visible = false;
	}

	function poster() {
		commit = NEW;
		lines = [...LOG];
	}

	async function script(act: Act) {
		cursor.x = 640;
		cursor.y = 420;
		await act.wait(700);
		cursor.visible = true;
		await tick();
		await act.move(cursor, act.point(button, 0.45, 0.6));
		await act.click(cursor);
		deploying = true;
		await act.wait(300);
		cursor.visible = false;
		for (const [i, line] of LOG.entries()) {
			lines.push(line);
			if (i === 3) starting = true;
			await act.wait(i === 1 ? 1400 : 750);
		}
		starting = false;
		deploying = false;
		commit = NEW;
		await act.wait(2200);
		reset();
		await act.wait(500);
	}
</script>

{#snippet nav()}
	<div class="flex items-center gap-2 px-5 pt-3 pb-5 font-fc-title text-[15px] font-bold text-fc-fg">
		<Icon icon={logoIcon} size={20} />
		<span>Gare</span>
	</div>
	<div class="grid gap-1 px-3">
		<div class="flex h-[34px] items-center gap-2.5 rounded-fc-md bg-fc-fg/5 px-2.5 text-[13px] font-bold text-fc-fg">
			<Icon icon={icons.server} size={16} />Apps
		</div>
		<div class="flex h-[34px] items-center gap-2.5 rounded-fc-md px-2.5 text-[13px] text-fc-fg-muted">
			<Icon icon={icons.plus} size={16} />New app
		</div>
	</div>
{/snippet}

{#snippet shop(compact: boolean)}
	<div class="gd-card">
		<div class="flex items-center justify-between gap-2">
			<span class="text-[16px] font-bold">shop</span>
			<span class="flex items-center gap-1.5">
				<StatusDot tone={starting ? 'info' : 'success'} pulse={starting} />
				<Roll value={starting ? 'Starting' : 'Running'} class="gd-status" />
			</span>
		</div>
		<div class="gd-meta">
			<span class="text-fc-fg-muted">Type</span><span><Badge>Container</Badge></span>
			<span class="text-fc-fg-muted">Commit</span><span class="gd-commit"><Roll value={commit} /></span>
		</div>
		<div class="mt-auto flex gap-2">
			<span bind:this={button}>
				<Button size={compact ? 'sm' : 'md'} icon={icons.refresh} disabled={deploying}>
					{deploying ? 'Starting deploy' : 'Deploy'}
				</Button>
			</span>
		</div>
	</div>
{/snippet}

<Stage
	width={976}
	height={564}
	compact={{ below: 640, width: 358, height: 500 }}
	class="gd-host"
	label="Gare deploys an app: a click on Deploy streams the build and health check log, then the new commit runs."
	{script}
	{reset}
	{poster}
>
	{#snippet children({ compact })}
		<Window
			style={compact ? 'top:8px;left:4px;width:350px;height:256px' : 'top:8px;left:4px;width:968px;height:300px'}
			sidebar={compact ? undefined : nav}
		>
			<div class="gd-body" class:gd-compact={compact}>
				<div class="flex items-center justify-between gap-3">
					<span class="font-fc-title text-fc-lg font-bold">Apps</span>
					{#if !compact}<Button icon={icons.plus}>New app</Button>{/if}
				</div>
				<div class="gd-grid">
					{@render shop(compact)}
					{#each compact ? [] : OTHERS as app (app.name)}
						<div class="gd-card">
							<div class="flex items-center justify-between gap-2">
								<span class="text-[16px] font-bold">{app.name}</span>
								<span class="flex items-center gap-1.5">
									<StatusDot tone="success" /><span class="gd-status">Running</span>
								</span>
							</div>
							<div class="gd-meta">
								<span class="text-fc-fg-muted">Type</span><span><Badge>{app.type}</Badge></span>
								<span class="text-fc-fg-muted">Commit</span><Bar w="{app.widths[0]}%" />
							</div>
							<div class="mt-auto"><Bar w="{app.widths[1]}%" /></div>
						</div>
					{/each}
				</div>
			</div>
		</Window>

		<Terminal style={compact ? 'top:280px;left:4px;width:350px;height:176px' : 'top:324px;left:4px;width:968px;height:196px'}>
			<div class="mb-3 flex items-center gap-2 text-fc-on-inverse-muted">
				<span class="size-2 rounded-fc-pill {deploying ? 'bg-fc-info' : 'bg-fc-success'}"></span>Deploy log · shop
			</div>
			{#each lines as line, i (i)}
				<div class="gd-line" class:gd-wrap={compact} in:fade={{ duration: 160 }} out:fade={{ duration: 200 }}>
					<span class:gd-ok={line.mark === '✓'} class:gd-dim={line.mark === '▸'}>{line.mark}</span>
					<span class:gd-ok={i === LOG.length - 1}>{line.text}</span>
				</div>
			{/each}
		</Terminal>
		<Cursor {cursor} />
	{/snippet}
</Stage>

<style>
	:global(.gd-host) {
		--mes-shadow-hero: 0 14px 32px -14px rgb(0 0 0 / 0.2), 0 4px 12px -6px rgb(0 0 0 / 0.08);
	}
	.gd-body {
		display: flex;
		flex: 1;
		flex-direction: column;
		gap: 18px;
		min-height: 0;
		padding: 18px 28px 24px;
		font-family: var(--font-fc-body);
		color: var(--color-fc-fg);
	}
	.gd-compact {
		gap: 12px;
		padding: 12px 14px 14px;
	}
	.gd-grid {
		display: grid;
		flex: 1;
		min-height: 0;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 14px;
	}
	.gd-compact .gd-grid {
		grid-template-columns: minmax(0, 1fr);
	}
	.gd-card {
		display: flex;
		flex-direction: column;
		gap: 14px;
		padding: 16px;
		border-radius: var(--radius-fc-lg);
		background: var(--color-fc-surface);
	}
	.gd-compact .gd-card {
		gap: 10px;
		min-height: 0;
		padding: 12px;
	}
	.gd-meta {
		display: grid;
		grid-template-columns: auto 1fr;
		align-items: center;
		gap: 6px 14px;
		font-size: 13px;
	}
	.gd-compact .gd-meta {
		gap: 6px 8px;
		font-size: 12px;
	}
	.gd-commit {
		display: inline-flex;
		width: 8ch;
		font-family: var(--font-fc-mono);
	}
	:global(.gd-status) {
		min-width: 4.5rem;
		font-size: 13px;
		color: var(--color-fc-fg-muted);
		white-space: nowrap;
	}
	.gd-line {
		display: flex;
		gap: 8px;
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
	}
	.gd-line > span:last-child {
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.gd-wrap {
		font-size: 11px;
		line-height: 17px;
	}
	.gd-dim {
		color: var(--color-fc-on-inverse-muted);
	}
	.gd-ok {
		color: color-mix(in oklab, var(--color-fc-success), var(--color-fc-on-inverse) 35%);
	}
</style>
