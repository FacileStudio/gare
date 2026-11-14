<script lang="ts">
	import { Icon, icons } from '@facile/muse';
	import { logoIcon } from '$lib/brand';
	import { t } from '$lib/i18n/index.svelte';
</script>

<div class="flex items-center gap-2 px-5 pt-3 pb-5 font-fc-title text-[15px] font-bold text-fc-fg">
	<Icon icon={logoIcon} size={20} />
	<span>Gare</span>
</div>
<div class="grid gap-1 px-3">
	<div class="flex h-[34px] items-center gap-2.5 rounded-fc-md bg-fc-fg/5 px-2.5 text-[13px] font-bold text-fc-fg">
		<Icon icon={icons.server} size={16} />{t('demo.apps')}
	</div>
	<div class="flex h-[34px] items-center gap-2.5 rounded-fc-md px-2.5 text-[13px] text-fc-fg-muted">
		<Icon icon={icons.plus} size={16} />{t('demo.newApp')}
	</div>
</div>
