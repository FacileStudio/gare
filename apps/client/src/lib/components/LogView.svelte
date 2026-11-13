<script lang="ts">
	let { lines, label, empty = 'No output yet.' }: { lines: string[]; label: string; empty?: string } = $props();

	let box: HTMLElement | null = $state(null);
	let pinned = $state(true);

	/** Follows new lines only while the reader sits at the bottom, so scrolling back to read is never yanked away. */
	function onscroll(): void {
		if (box) pinned = box.scrollHeight - box.scrollTop - box.clientHeight < 24;
	}

	$effect(() => {
		void lines.length;
		if (pinned && box) box.scrollTop = box.scrollHeight;
	});
</script>

<div
	bind:this={box}
	{onscroll}
	class="max-h-[65dvh] min-h-48 overflow-auto rounded-fc-lg bg-fc-surface p-4 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
>
	<div role="log" aria-live="off" aria-label={label}>
		{#if lines.length === 0}
			<p class="text-fc-sm text-fc-fg-muted">{empty}</p>
		{:else}
			<pre class="font-fc-mono text-fc-xs break-all whitespace-pre-wrap text-fc-fg">{lines.join('\n')}</pre>
		{/if}
	</div>
</div>
