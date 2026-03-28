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

<div class="rounded-xl border border-slate-800 bg-surface-light p-6">
  <div class="flex items-center gap-2 mb-4">
    <svg class="w-4 h-4 text-blue-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" d="M12 2.25c-3 4.5-6.75 7.5-6.75 12a6.75 6.75 0 1013.5 0c0-4.5-3.75-7.5-6.75-12z" />
    </svg>
    <h3 class="text-sm font-semibold text-slate-100">Water Intake</h3>
  </div>

  <div class="mb-4">
    <div class="flex items-end justify-between mb-1.5">
      <span class="text-2xl font-bold text-slate-100">{pct}%</span>
      <span class="text-xs text-slate-400">{totalMl.toLocaleString()} / {TARGET.toLocaleString()} ml</span>
    </div>
    <div class="w-full h-2.5 rounded-full bg-slate-700 overflow-hidden">
      <div
        class="h-full rounded-full bg-blue-500 transition-all duration-500 ease-out"
        style="width: {fillWidth}%"
      ></div>
    </div>
  </div>

  <div class="flex items-center gap-2 mb-3">
    {#each [250, 500, 750] as amount}
      <button
        class="bg-surface border border-slate-700 hover:bg-surface-lighter rounded-lg px-3 py-1.5 text-xs text-slate-300 transition-colors"
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
      placeholder="ml"
      min="1"
      class="w-20 bg-surface border border-slate-700 rounded-lg px-3 py-1.5 text-xs text-slate-200 placeholder:text-slate-500 focus:outline-none focus:border-blue-500 transition-colors"
    />
    <button
      class="bg-blue-500 hover:bg-blue-600 disabled:bg-slate-700 disabled:text-slate-500 text-white rounded-lg px-3 py-1.5 text-xs font-medium transition-colors"
      onclick={handleCustomAdd}
      disabled={loading || !customAmount}
    >
      +
    </button>
  </div>
</div>
