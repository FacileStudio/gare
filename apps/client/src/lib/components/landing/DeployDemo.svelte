<script lang="ts">
	import { tick } from 'svelte';
	import { fade } from 'svelte/transition';
	import { Badge, Button, StatusDot, icons } from '@facile/muse';
	import { Bar, Cursor, Roll, Stage, Terminal, Window, cursorAt, type Act } from '@facile/mise-en-scene';
	import { t } from '$lib/i18n/index.svelte';
	import DemoNav from './DemoNav.svelte';
	import { BAD, FAIL_LOG, LOG, NEW, OLD, OTHERS, type Line } from './deploy-log';

	let deploying = $state(false);
	let starting = $state(false);
	let failing = $state(false);
	let commit = $state(OLD);
	let lines = $state<Line[]>([]);
	let cursor = $state(cursorAt());
	let button = $state<HTMLElement>();

	function reset() {
		deploying = false;
		starting = false;
		failing = false;
		commit = OLD;
		lines = [];
		cursor.visible = false;
	}

	function poster() {
		commit = NEW;
		failing = true;
		lines = [...FAIL_LOG];
	}

	async function stream(act: Act, log: Line[], onLine: (i: number) => void) {
		for (const [i, line] of log.entries()) {
			lines.push(line);
			onLine(i);
			await act.wait(i === 1 ? 1300 : 700);
		}
	}

	async function firstDeploy(act: Act) {
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
		await stream(act, LOG, (i) => (starting = i === 3));
		starting = false;
		deploying = false;
		commit = NEW;
	}

	async function script(act: Act) {
		await firstDeploy(act);
		await act.wait(1800);
		lines = [];
		await act.wait(400);
		deploying = true;
		await stream(act, FAIL_LOG, (i) => {
			if (i === 2) {
				starting = true;
				commit = BAD;
			}
			if (i === 3) failing = true;
			if (i === 5) commit = NEW;
		});
		starting = false;
		deploying = false;
		await act.wait(2200);
		reset();
		await act.wait(500);
	}
</script>

{#snippet nav()}<DemoNav />{/snippet}

{#snippet shop(compact: boolean)}
	<div class="gd-card">
		<div class="flex items-center justify-between gap-2">
			<span class="text-[16px] font-bold">shop</span>
			<span class="flex items-center gap-1.5">
				<StatusDot tone={starting ? 'info' : 'success'} pulse={starting} />
				<Roll value={starting ? t('demo.starting') : t('demo.running')} class="gd-status" />
			</span>
		</div>
		<div class="gd-meta">
			<span class="text-fc-fg-muted">{t('demo.type')}</span><span><Badge>Container</Badge></span>
			<span class="text-fc-fg-muted">{t('demo.commit')}</span><span class="gd-commit"><Roll value={commit} /></span>
		</div>
		<div class="mt-auto flex gap-2">
			<span bind:this={button}>
				<Button size={compact ? 'sm' : 'md'} icon={icons.refresh} disabled={deploying}>
					{deploying ? t('demo.deploying') : t('demo.deploy')}
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
	label={t('demo.label')}
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
					<span class="font-fc-title text-fc-lg font-bold">{t('demo.apps')}</span>
					{#if !compact}<Button icon={icons.plus}>{t('demo.newApp')}</Button>{/if}
				</div>
				<div class="gd-grid">
					{@render shop(compact)}
					{#each compact ? [] : OTHERS as app (app.name)}
						<div class="gd-card">
							<div class="flex items-center justify-between gap-2">
								<span class="text-[16px] font-bold">{app.name}</span>
								<span class="flex items-center gap-1.5">
									<StatusDot tone="success" /><span class="gd-status">{t('demo.running')}</span>
								</span>
							</div>
							<div class="gd-meta">
								<span class="text-fc-fg-muted">{t('demo.type')}</span><span><Badge>{app.type}</Badge></span>
								<span class="text-fc-fg-muted">{t('demo.commit')}</span><Bar w="{app.widths[0]}%" />
							</div>
							<div class="mt-auto"><Bar w="{app.widths[1]}%" /></div>
						</div>
					{/each}
				</div>
			</div>
		</Window>

		<Terminal style={compact ? 'top:280px;left:4px;width:350px;height:176px' : 'top:324px;left:4px;width:968px;height:196px'}>
			<div class="mb-3 flex items-center gap-2 text-fc-on-inverse-muted">
				<span class="size-2 rounded-fc-pill {deploying ? 'bg-fc-info' : failing ? 'bg-fc-warning' : 'bg-fc-success'}"></span>{t('demo.log')} · shop
			</div>
			{#each lines as line, i (i)}
				<div class="gd-line" class:gd-wrap={compact} in:fade={{ duration: 160 }} out:fade={{ duration: 200 }}>
					<span class:gd-ok={line.mark === '✓'} class:gd-dim={line.mark === '▸'} class:gd-warn={line.mark === '!'}>{line.mark}</span>
					<span class:gd-ok={line.final} class:gd-warn={line.mark === '!'}>{line.text}</span>
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
	.gd-warn {
		color: color-mix(in oklab, var(--color-fc-warning), var(--color-fc-on-inverse) 35%);
	}
	.gd-ok {
		color: color-mix(in oklab, var(--color-fc-success), var(--color-fc-on-inverse) 35%);
	}
</style>
