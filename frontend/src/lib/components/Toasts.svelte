<script lang="ts">
  import { toasts } from '../toast';
</script>

<div aria-live="polite" class="fixed bottom-4 left-1/2 -translate-x-1/2 z-[60] flex flex-col items-center gap-2 w-max max-w-[92vw]">
  {#each $toasts as t (t.id)}
    <div class="flex items-center gap-3 rounded-md border border-zinc-700 bg-zinc-900 px-3.5 py-2 shadow-xl text-xs text-zinc-100">
      <span>{t.msg}</span>
      {#if t.actionLabel && t.onAction}
        <button
          on:click={t.onAction}
          class="rounded bg-zinc-100 text-zinc-950 px-2 py-0.5 text-xs font-medium hover:bg-white transition-colors"
        >
          {t.actionLabel}
        </button>
      {/if}
    </div>
  {/each}
</div>
