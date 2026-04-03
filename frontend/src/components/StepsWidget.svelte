<script lang="ts">
  import { api } from '$lib/api';
  import { onMount } from 'svelte';

  interface StepLog {
    date: string;
    step_count: number;
    source: string;
  }

  const STEP_GOAL = 10000;

  let stepData = $state<StepLog | null>(null);
  let googleConnected = $state(false);
  let loading = $state(true);
  let syncing = $state(false);
  let logging = $state(false);
  let stepInput = $state('');
  let syncSuccess = $state(false);
  let logSuccess = $state(false);

  const today = new Date().toISOString().split('T')[0];

  function dispatchStepsUpdated() {
    window.dispatchEvent(new CustomEvent('steps-updated'));
  }

  onMount(async () => {
    try {
      const [steps, status] = await Promise.all([
        api.get<StepLog | null>(`/steps?date=${today}`),
        api.get<{ connected: boolean }>('/steps/google/status'),
      ]);
      stepData = steps;
      googleConnected = status.connected;
    } catch {
    } finally {
      loading = false;
    }
  });

  async function logSteps() {
    const count = parseInt(stepInput, 10);
    if (logging || isNaN(count) || count <= 0) return;
    logging = true;
    try {
      const result = await api.post<StepLog>('/steps', { step_count: count, date: today });
      stepData = result;
      stepInput = '';
      logSuccess = true;
      setTimeout(() => { logSuccess = false; }, 2000);
      dispatchStepsUpdated();
    } catch {
    } finally {
      logging = false;
    }
  }

  async function syncGoogleFit() {
    if (syncing) return;
    syncing = true;
    try {
      const result = await api.post<{ synced_steps: number; date: string }>('/steps/google/sync');
      if (stepData) {
        stepData = { ...stepData, step_count: result.synced_steps, source: 'Google Fit' };
      } else {
        stepData = { date: result.date, step_count: result.synced_steps, source: 'Google Fit' };
      }
      syncSuccess = true;
      setTimeout(() => { syncSuccess = false; }, 2000);
      dispatchStepsUpdated();
    } catch {
    } finally {
      syncing = false;
    }
  }

  let stepCount = $derived(stepData?.step_count ?? 0);
  let progressPct = $derived(Math.min(100, Math.round((stepCount / STEP_GOAL) * 100)));
  let source = $derived(stepData?.source ?? '');
</script>

<div class="rounded-2xl border border-border bg-card p-5">
  <div class="flex items-center justify-between mb-4">
    <div class="flex items-center gap-2">
      <svg class="w-4 h-4 text-orange-600 dark:text-orange-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
      </svg>
      <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Steps</p>
    </div>
    {#if source}
      <span class="text-[10px] font-medium px-2 py-0.5 rounded-full {source === 'Google Fit' ? 'bg-blue-500/20 dark:bg-blue-500/15 text-blue-600 dark:text-blue-400' : 'bg-accent/50 text-muted-foreground'}">
        {source === 'Google Fit' ? 'Google Fit' : 'manual'}
      </span>
    {/if}
  </div>

  {#if loading}
    <div class="flex items-center justify-center h-12">
      <div class="h-5 w-5 animate-spin rounded-full border-2 border-border border-t-orange-600 dark:border-t-orange-400"></div>
    </div>
  {:else}
    <div class="mb-4">
      <div class="flex items-baseline justify-between mb-2">
        <span class="font-display text-2xl font-bold text-foreground">{stepCount.toLocaleString()}</span>
        <span class="text-xs text-muted-foreground">/ {STEP_GOAL.toLocaleString()}</span>
      </div>
      <div class="w-full bg-accent/50 rounded-full h-2.5 overflow-hidden">
        <div
          class="h-full rounded-full transition-all duration-500 {progressPct >= 100 ? 'bg-gradient-to-r from-emerald-500 to-emerald-400' : 'bg-gradient-to-r from-orange-500 to-orange-400'}"
          style="width: {progressPct}%"
        ></div>
      </div>
      <p class="text-[11px] text-muted-foreground mt-1.5">{progressPct}% of daily goal</p>
    </div>

    <div class="flex items-center gap-2 mb-2.5">
      <input
        type="number"
        bind:value={stepInput}
        placeholder="Steps"
        min="1"
        class="w-28 bg-secondary border border-border rounded-xl px-3 py-2 text-xs text-foreground/80 placeholder:text-muted-foreground focus:outline-none focus:border-orange-500/50 focus:ring-2 focus:ring-orange-500/20 transition-colors"
      />
      <button
        class="bg-orange-500 hover:bg-orange-400 disabled:opacity-40 text-foreground rounded-xl px-3 py-2 text-xs font-semibold transition-colors flex-shrink-0"
        onclick={logSteps}
        disabled={logging || !stepInput}
      >
        {logging ? '...' : logSuccess ? '✓' : 'Log'}
      </button>
    </div>

    {#if googleConnected}
      <button
        class="w-full flex items-center justify-center gap-2 border border-border hover:border-blue-500/40 dark:hover:border-blue-500/30 hover:bg-blue-500/20 dark:hover:bg-blue-500/8 text-foreground hover:text-blue-600 dark:hover:text-blue-400 rounded-xl px-3 py-2 text-xs font-medium transition-all disabled:opacity-50"
        onclick={syncGoogleFit}
        disabled={syncing}
      >
        <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor">
          <path d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"/>
        </svg>
        {syncing ? 'Syncing...' : syncSuccess ? 'Synced!' : 'Sync Google Fit'}
      </button>
    {/if}
  {/if}
</div>
