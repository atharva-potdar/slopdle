<script lang="ts">
  /** Brand mark: terminal prompt `›_` — the palette's grammar as logo.
   * Zinc box, mono glyph. Same mark navbar + login, sized via prop. */
  export let size: number = 24;
</script>

<span
  aria-hidden="true"
  style="width: {size}px; height: {size}px; font-size: {Math.round(size * 0.62)}px;"
  class="rounded-md bg-zinc-100 text-zinc-950 inline-flex items-center justify-center font-mono font-bold leading-none select-none shrink-0"
>
  ›_
</span>
