<script lang="ts">
	import { Badge } from '@facile/muse';
	import { Caret, Roll, Stage, Terminal, Tile, grow, type Act } from '@facile/mise-en-scene';

	const COMMAND = 'git push origin main';

	let typed = $state('');
	let pushed = $state(false);
	let done = $state(false);

	function reset() {
		typed = '';
		pushed = false;
		done = false;
	}

	function poster() {
		typed = COMMAND;
		pushed = true;
		done = true;
	}

	async function script(act: Act) {
		await act.wait(700);
		await act.type(COMMAND, (v) => (typed = v), 65);
		await act.wait(400);
		pushed = true;
		await act.wait(2200);
		done = true;
		await act.wait(2600);
		reset();
		await act.wait(500);
	}
</script>

{#snippet live()}
	<span in:grow={{ duration: 250 }} out:grow={{ duration: 200 }}><Badge tone="accent">Live</Badge></span>
{/snippet}

<Tile title="Deploys on git push" body="A signed GitHub or GitLab webhook starts the deploy.">
	<Stage width={320} height={240} {script} {reset} {poster}>
		{#snippet children()}
			<Terminal style="top:20px;left:20px;right:20px;height:64px">
				<div class="whitespace-nowrap">
					<span class="dim">~/shop $</span>
					{typed}{#if !pushed}<Caret />{/if}
				</div>
				{#if pushed}
					<div class="dim whitespace-nowrap" in:grow={{ duration: 300 }}>→ main 4f2c1a9</div>
				{/if}
			</Terminal>

			<ol class="panel absolute top-24 right-5 left-5 flex h-[120px] flex-col justify-end rounded-fc-lg bg-fc-page px-4 py-2">
				{#if pushed}
					<li class="flex flex-col gap-1 py-1.5" in:grow={{ duration: 350 }} out:grow={{ duration: 250 }}>
						<div class="flex items-center gap-2">
							<Badge tone={done ? 'success' : 'info'}><Roll value={done ? 'Succeeded' : 'Running'} class="h-4 w-16 leading-4" /></Badge>
							<span class="font-fc-mono text-fc-xs text-fc-fg">4f2c1a9</span>
							{#if done}{@render live()}{/if}
						</div>
						<span class="text-[11px] leading-4 text-fc-fg-muted">just now by webhook</span>
					</li>
				{/if}
				<li class="flex flex-col gap-1 py-1.5">
					<div class="flex items-center gap-2">
						<Badge tone="success">Succeeded</Badge>
						<span class="font-fc-mono text-fc-xs text-fc-fg">a41c9e2</span>
						{#if !done}{@render live()}{/if}
					</div>
					<span class="text-[11px] leading-4 text-fc-fg-muted">yesterday by webhook</span>
				</li>
			</ol>
		{/snippet}
	</Stage>
</Tile>

<style>
	.dim {
		color: var(--color-fc-on-inverse-muted);
	}
	.panel {
		box-shadow: 0 14px 28px -14px rgb(0 0 0 / 0.25);
	}
</style>
