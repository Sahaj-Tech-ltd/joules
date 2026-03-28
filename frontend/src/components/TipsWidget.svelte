<script lang="ts">
  let { tips }: { tips: string | null } = $props();

  let displayTips = $derived(() => {
    if (!tips) return [];
    const lines = tips.split('\n').filter(l => l.trim());
    return lines.slice(0, 3);
  });
</script>

<div class="rounded-xl border border-slate-800 bg-surface-light p-6">
  <div class="mb-4 flex items-center gap-2">
    <svg class="h-4 w-4 text-joule-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 18v-5.25m0 0a6.01 6.01 0 001.5-.189m-1.5.189a6.01 6.01 0 01-1.5-.189m3.75 7.478a12.06 12.06 0 01-4.5 0m3.75 2.383a14.406 14.406 0 01-3 0M14.25 18v-.192c0-.983.658-1.823 1.508-2.316a7.5 7.5 0 10-7.517 0c.85.493 1.509 1.333 1.509 2.316V18" />
    </svg>
    <h3 class="text-sm font-semibold text-white">Daily Tips</h3>
  </div>

  {#if tips === null}
    <div class="flex items-center justify-center py-6">
      <div class="h-6 w-6 animate-spin rounded-full border-2 border-slate-700 border-t-joule-500"></div>
    </div>
  {:else if !tips.trim()}
    <p class="text-sm text-slate-400">Ask the coach for personalized tips</p>
  {:else}
    <div class="space-y-2">
      {#each displayTips() as tip, i}
        <div class="flex gap-2">
          <span class="mt-0.5 h-1.5 w-1.5 shrink-0 rounded-full bg-joule-500"></span>
          <p class="whitespace-pre-wrap text-sm text-slate-300 leading-relaxed">{tip.replace(/^[-•*]\s*/, '')}</p>
        </div>
      {/each}
    </div>
  {/if}

  <a
    href="/coach"
    class="mt-4 inline-block text-xs text-joule-500 hover:text-joule-400 transition"
  >
    Chat with Coach →
  </a>
</div>
