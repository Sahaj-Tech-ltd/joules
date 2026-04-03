<script lang="ts">
  import { api } from '$lib/api';

  let { totalMl = 0 }: { totalMl: number } = $props();

  const TARGET = 2500;

  let customAmount = $state('');
  let loading = $state(false);

  let pct = $derived(TARGET > 0 ? Math.min(100, Math.round((totalMl / TARGET) * 100)) : 0);
  let fillWidth = $derived(Math.min(100, (totalMl / TARGET) * 100));

  function dispatchLogged() {
    const event = new CustomEvent('water-logged');
    window.dispatchEvent(event);
  }

  async function addWater(amount: number) {
    if (loading || amount <= 0) return;
    loading = true;
    try {
      await api.post('/water', { amount_ml: amount });
      dispatchLogged();
    } catch {
    } finally {
      loading = false;
    }
  }

  function handleCustomAdd() {
    const amount = parseInt(customAmount, 10);
    if (isNaN(amount) || amount <= 0) return;
    addWater(amount);
    customAmount = '';
  }
</script>

<div class="rounded-2xl border border-border bg-card p-5">
  <div class="flex items-center justify-between mb-4">
    <div class="flex items-center gap-2">
      <svg class="w-4 h-4 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 2.25c-3 4.5-6.75 7.5-6.75 12a6.75 6.75 0 1013.5 0c0-4.5-3.75-7.5-6.75-12z" />
      </svg>
      <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Water Intake</p>
    </div>
    <span class="text-[11px] text-muted-foreground">{totalMl.toLocaleString()} / {TARGET.toLocaleString()} ml</span>
  </div>

  <div class="mb-4">
    <div class="flex items-baseline justify-between mb-2">
      <span class="font-display text-2xl font-bold text-foreground">{pct}%</span>
      <span class="text-xs text-blue-600 dark:text-blue-400 font-medium">{pct >= 100 ? 'Goal reached!' : `${TARGET - totalMl > 0 ? (TARGET - totalMl).toLocaleString() : 0} ml to go`}</span>
    </div>
    <div class="w-full h-2.5 rounded-full bg-accent/50 overflow-hidden">
      <div
        class="h-full rounded-full bg-gradient-to-r from-blue-500 to-cyan-400 transition-all duration-500 ease-out"
        style="width: {fillWidth}%"
      ></div>
    </div>
  </div>

  <div class="grid grid-cols-3 gap-2 mb-3">
    {#each [250, 500, 750] as amount}
      <button
        class="w-full text-center bg-blue-500/15 dark:bg-blue-500/8 border border-blue-500/30 dark:border-blue-500/20 hover:bg-blue-500/25 dark:hover:bg-blue-500/15 rounded-xl px-3 py-2 text-xs font-medium text-blue-600 dark:text-blue-400 transition-colors"
        onclick={() => addWater(amount)}
        disabled={loading}
      >
        +{amount} ml
      </button>
    {/each}
  </div>

  <div class="flex items-center gap-2">
    <input
      type="number"
      bind:value={customAmount}
      placeholder="Custom ml"
      min="1"
      class="flex-1 bg-secondary border border-border rounded-xl px-3 py-2 text-xs text-foreground/80 placeholder:text-muted-foreground focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20 transition-colors"
    />
    <button
      class="bg-blue-500 hover:bg-blue-400 disabled:opacity-40 text-foreground rounded-xl px-4 py-2 text-xs font-semibold transition-colors"
      onclick={handleCustomAdd}
      disabled={loading || !customAmount}
    >
      Add
    </button>
  </div>
</div>
