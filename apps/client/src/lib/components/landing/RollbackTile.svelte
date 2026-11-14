<script lang="ts">
	import { Badge, StatusDot } from '@facile/muse';
	import { Roll, Stage, Tile, grow, type Act } from '@facile/mise-en-scene';

	let started = $state(false);
	let failed = $state(false);

	function reset() {
		started = false;
		failed = false;
	}

	function poster() {
		started = true;
		failed = true;
	}

	async function script(act: Act) {
		await act.wait(900);
		started = true;
		await act.wait(2000);
		failed = true;
		await act.wait(3200);
		reset();
		await act.wait(600);
	}
</script>

<Tile title="Undoes a bad deploy" body="When a health check fails, the last good build runs again.">
	<Stage width={320} height={240} {script} {reset} {poster}>
		{#snippet children()}
			<div class="panel absolute top-5 right-5 left-5 flex h-[188px] flex-col gap-3 rounded-fc-lg bg-fc-page p-4">
				<div class="flex items-center justify-between gap-2">
					<span class="font-fc-title text-fc-sm font-bold text-fc-fg">shop</span>
					<span class="flex items-center gap-1.5 text-fc-xs text-fc-fg-muted"><StatusDot tone="success" />Running</span>
				</div>
				<ol class="flex flex-col rounded-fc-md bg-fc-component px-3">
					{#if started}
						<li class="flex flex-col gap-1 pt-2.5" in:grow={{ duration: 350 }} out:grow={{ duration: 250 }}>
							<div class="flex items-center gap-2">
								<Badge tone={failed ? 'warning' : 'info'}><Roll value={failed ? 'Rolled back' : 'Running'} class="h-4 w-[4.6rem] leading-4" /></Badge>
								<span class="font-fc-mono text-fc-xs text-fc-fg">9e3b7d0</span>
							</div>
							<Roll value={failed ? 'Health check failed on /' : 'Checking health...'} class="h-4 text-[11px] leading-4 text-fc-fg-muted" />
						</li>
					{/if}
					<li class="flex flex-col gap-1 py-2.5">
						<div class="flex items-center gap-2">
							<Badge tone="success">Succeeded</Badge>
							<span class="font-fc-mono text-fc-xs text-fc-fg">4f2c1a9</span>
							<Badge tone="accent">Live</Badge>
						</div>
						<span class="text-[11px] text-fc-fg-muted">2 hours ago by webhook</span>
					</li>
				</ol>
			</div>
		{/snippet}
	</Stage>
</Tile>

<style>
	.panel {
		box-shadow: 0 14px 28px -14px rgb(0 0 0 / 0.25);
	}
</style>
