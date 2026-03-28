<script lang="ts">
  import Sidebar from '$components/Sidebar.svelte';
  import ThemeToggle from '$components/ThemeToggle.svelte';
  import { authToken, userGoals } from '$lib/stores';
  import { api } from '$lib/api';
  import { goto } from '$app/navigation';
  import { onMount, onDestroy } from 'svelte';
  import {
    Chart, CategoryScale, LinearScale, PointElement, LineElement,
    BarElement, BarController, Tooltip, Filler, LineController, Legend
  } from 'chart.js';
  Chart.register(CategoryScale, LinearScale, PointElement, LineElement, BarElement, BarController, Tooltip, Filler, LineController, Legend);

  interface WeightEntry {
    date: string;
    weight_kg: number;
  }

  interface DailySummary {
    date: string;
    total_calories: number;
    total_protein: number;
    total_carbs: number;
    total_fat: number;
    total_fiber: number;
    total_burned: number;
    total_water_ml: number;
  }

  let weightHistory = $state<WeightEntry[]>([]);
  let weekSummaries = $state<DailySummary[]>([]);
  let goals = $state($userGoals);
  let loading = $state(true);
  let logWeight = $state('');
  let loggingWeight = $state(false);
  let weightSuccess = $state(false);
  let weightError = $state('');

  let weightCanvas: HTMLCanvasElement;
  let calorieCanvas: HTMLCanvasElement;
  let weightChart: Chart | undefined;
  let calorieChart: Chart | undefined;

  const MIN_WEIGHT_ENTRIES = 3;
  const MIN_CALORIE_ENTRIES = 3;

  let latestWeight = $derived(
    weightHistory.length > 0 ? weightHistory[weightHistory.length - 1].weight_kg : null
  );
  let firstWeight = $derived(
    weightHistory.length > 0 ? weightHistory[0].weight_kg : null
  );
  let weightChange = $derived(
    latestWeight !== null && firstWeight !== null ? +(latestWeight - firstWeight).toFixed(1) : null
  );

  let avgCalories = $derived(
    weekSummaries.length > 0
      ? Math.round(weekSummaries.reduce((s, d) => s + d.total_calories, 0) / weekSummaries.length)
      : null
  );
  let avgWater = $derived(
    weekSummaries.length > 0
      ? Math.round(weekSummaries.reduce((s, d) => s + d.total_water_ml, 0) / weekSummaries.length)
      : null
  );

  function buildWeightChart() {
    weightChart?.destroy();
    if (!weightCanvas || weightHistory.length < MIN_WEIGHT_ENTRIES) return;
    weightChart = new Chart(weightCanvas, {
      type: 'line',
      data: {
        labels: weightHistory.map(d => {
          const dt = new Date(d.date);
          return dt.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
        }),
        datasets: [{
          label: 'Weight (kg)',
          data: weightHistory.map(d => d.weight_kg),
          borderColor: '#f59e0b',
          backgroundColor: 'rgba(245,158,11,0.08)',
          fill: true,
          tension: 0.35,
          pointBackgroundColor: '#f59e0b',
          pointBorderColor: '#f59e0b',
          pointRadius: 4,
          pointHoverRadius: 6,
          borderWidth: 2
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          x: { grid: { color: 'rgba(51,65,85,0.4)' }, ticks: { color: '#64748b', font: { size: 11 } } },
          y: { grid: { color: 'rgba(51,65,85,0.4)' }, ticks: { color: '#64748b', font: { size: 11 } }, beginAtZero: false }
        },
        plugins: {
          legend: { display: false },
          tooltip: { backgroundColor: '#1e293b', titleColor: '#e2e8f0', bodyColor: '#e2e8f0', borderColor: '#334155', borderWidth: 1 }
        }
      }
    });
  }

  function buildCalorieChart() {
    calorieChart?.destroy();
    if (!calorieCanvas || weekSummaries.length < MIN_CALORIE_ENTRIES) return;
    const target = goals?.daily_calorie_target ?? 0;
    calorieChart = new Chart(calorieCanvas, {
      type: 'bar',
      data: {
        labels: weekSummaries.map(d => {
          const dt = new Date(d.date);
          return dt.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' });
        }),
        datasets: [
          {
            label: 'Calories eaten',
            data: weekSummaries.map(d => d.total_calories),
            backgroundColor: weekSummaries.map(d =>
              target > 0 && d.total_calories <= target ? 'rgba(245,158,11,0.7)' : 'rgba(239,68,68,0.6)'
            ),
            borderRadius: 6,
            borderSkipped: false
          }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          x: { grid: { display: false }, ticks: { color: '#64748b', font: { size: 10 } } },
          y: {
            grid: { color: 'rgba(51,65,85,0.4)' },
            ticks: { color: '#64748b', font: { size: 11 } },
            beginAtZero: true
          }
        },
        plugins: {
          legend: { display: false },
          tooltip: { backgroundColor: '#1e293b', titleColor: '#e2e8f0', bodyColor: '#e2e8f0', borderColor: '#334155', borderWidth: 1 }
        }
      }
    });
  }

  async function loadData() {
    const now = new Date();
    const from30 = new Date();
    from30.setDate(now.getDate() - 30);
    const from7 = new Date();
    from7.setDate(now.getDate() - 7);

    const fmt = (d: Date) => d.toISOString().split('T')[0];

    try {
      const [weights] = await Promise.all([
        api.get<WeightEntry[]>(`/weight?from=${fmt(from30)}&to=${fmt(now)}`)
      ]);
      weightHistory = (weights ?? []).sort((a, b) => a.date.localeCompare(b.date));

      // Fetch last 7 daily summaries
      const summaryPromises = [];
      for (let i = 6; i >= 0; i--) {
        const d = new Date();
        d.setDate(d.getDate() - i);
        summaryPromises.push(
          api.get<DailySummary>(`/dashboard/summary?date=${fmt(d)}`).catch(() => null)
        );
      }
      const results = await Promise.all(summaryPromises);
      weekSummaries = results.filter((r): r is DailySummary => r !== null && (r.total_calories > 0 || r.total_water_ml > 0));
    } catch {
      // partial load is fine
    } finally {
      loading = false;
      setTimeout(() => {
        buildWeightChart();
        buildCalorieChart();
      }, 50);
    }
  }

  async function handleLogWeight() {
    const w = parseFloat(logWeight);
    if (isNaN(w) || w <= 0) return;
    loggingWeight = true;
    weightError = '';
    try {
      await api.post('/weight', { weight_kg: w });
      weightSuccess = true;
      logWeight = '';
      setTimeout(() => (weightSuccess = false), 2500);
      weightChart?.destroy();
      calorieChart?.destroy();
      await loadData();
    } catch (err) {
      weightError = err instanceof Error ? err.message : 'Failed to log weight';
    } finally {
      loggingWeight = false;
    }
  }

  onMount(() => {
    const unsub = authToken.subscribe(token => {
      if (!token) goto('/login');
    });
    loadData();
    return unsub;
  });

  onDestroy(() => {
    weightChart?.destroy();
    calorieChart?.destroy();
  });
</script>

<div class="flex min-h-screen">
  <Sidebar activePage="progress" />

  <main class="flex-1 p-4 pb-20 lg:p-10 lg:pb-10">
    <div class="mb-8 flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Progress</h1>
        <p class="mt-1 text-sm text-slate-400">Your trends over time</p>
      </div>
      <div class="flex items-center gap-2">
        <ThemeToggle />
        <button
          onclick={() => { authToken.set(null); goto('/login'); }}
          class="rounded-lg border border-slate-700 px-3 py-1.5 text-sm text-slate-400 hover:text-white transition"
        >
          Sign out
        </button>
      </div>
    </div>

    {#if loading}
      <div class="flex h-64 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-slate-700 border-t-joule-500"></div>
      </div>
    {:else}
      <div class="space-y-6">

        <!-- Stats row -->
        <div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <div class="rounded-xl border border-slate-800 bg-surface-light p-4">
            <p class="text-xs text-slate-400">Current Weight</p>
            {#if latestWeight !== null}
              <p class="mt-1 text-2xl font-bold text-white">{latestWeight}<span class="text-sm font-normal text-slate-400"> kg</span></p>
            {:else}
              <p class="mt-1 text-sm text-slate-500">Not logged yet</p>
            {/if}
          </div>
          <div class="rounded-xl border border-slate-800 bg-surface-light p-4">
            <p class="text-xs text-slate-400">30-day change</p>
            {#if weightChange !== null}
              <p class="mt-1 text-2xl font-bold {weightChange <= 0 ? 'text-emerald-400' : 'text-red-400'}">
                {weightChange > 0 ? '+' : ''}{weightChange}<span class="text-sm font-normal text-slate-400"> kg</span>
              </p>
            {:else}
              <p class="mt-1 text-sm text-slate-500">Not enough data</p>
            {/if}
          </div>
          <div class="rounded-xl border border-slate-800 bg-surface-light p-4">
            <p class="text-xs text-slate-400">7-day avg calories</p>
            {#if avgCalories !== null}
              <p class="mt-1 text-2xl font-bold text-white">{avgCalories.toLocaleString()}<span class="text-sm font-normal text-slate-400"> kcal</span></p>
              {#if goals?.daily_calorie_target}
                <p class="mt-0.5 text-xs text-slate-500">Target: {goals.daily_calorie_target.toLocaleString()} kcal</p>
              {/if}
            {:else}
              <p class="mt-1 text-sm text-slate-500">No data yet</p>
            {/if}
          </div>
          <div class="rounded-xl border border-slate-800 bg-surface-light p-4">
            <p class="text-xs text-slate-400">7-day avg water</p>
            {#if avgWater !== null}
              <p class="mt-1 text-2xl font-bold text-blue-400">{avgWater.toLocaleString()}<span class="text-sm font-normal text-slate-400"> ml</span></p>
              <p class="mt-0.5 text-xs text-slate-500">Target: 2,500 ml</p>
            {:else}
              <p class="mt-1 text-sm text-slate-500">No data yet</p>
            {/if}
          </div>
        </div>

        <!-- Weight chart -->
        <div class="rounded-xl border border-slate-800 bg-surface-light p-6">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-sm font-semibold text-slate-100">Weight Trend (30 days)</h2>
            <span class="text-xs text-slate-500">{weightHistory.length} entries</span>
          </div>
          {#if weightHistory.length < MIN_WEIGHT_ENTRIES}
            <div class="flex h-48 flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-slate-700">
              <svg class="h-8 w-8 text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
              <p class="text-sm text-slate-400">Not enough data yet</p>
              <p class="text-xs text-slate-500">Log at least {MIN_WEIGHT_ENTRIES} weigh-ins to see your trend · You have {weightHistory.length}</p>
            </div>
          {:else}
            <div class="h-56">
              <canvas bind:this={weightCanvas}></canvas>
            </div>
          {/if}
        </div>

        <!-- Calorie chart -->
        <div class="rounded-xl border border-slate-800 bg-surface-light p-6">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-sm font-semibold text-slate-100">Daily Calories (last 7 days)</h2>
            {#if goals?.daily_calorie_target}
              <div class="flex items-center gap-3 text-xs">
                <span class="flex items-center gap-1"><span class="inline-block h-2.5 w-2.5 rounded-sm bg-joule-500/70"></span>On target</span>
                <span class="flex items-center gap-1"><span class="inline-block h-2.5 w-2.5 rounded-sm bg-red-500/60"></span>Over target</span>
              </div>
            {/if}
          </div>
          {#if weekSummaries.length < MIN_CALORIE_ENTRIES}
            <div class="flex h-48 flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-slate-700">
              <svg class="h-8 w-8 text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" />
              </svg>
              <p class="text-sm text-slate-400">Not enough data yet</p>
              <p class="text-xs text-slate-500">Log meals for at least {MIN_CALORIE_ENTRIES} days to see the chart · <a href="/log" class="text-joule-400 hover:underline">Log a meal</a></p>
            </div>
          {:else}
            <div class="h-56">
              <canvas bind:this={calorieCanvas}></canvas>
            </div>
          {/if}
        </div>

        <!-- Log weight widget -->
        <div class="rounded-xl border border-slate-800 bg-surface-light p-6">
          <h2 class="mb-4 text-sm font-semibold text-slate-100">Log Today's Weight</h2>
          <div class="flex items-center gap-3">
            <input
              type="number"
              bind:value={logWeight}
              placeholder="e.g. 72.5"
              min="1"
              step="0.1"
              class="w-40 rounded-lg border border-slate-700 bg-surface px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
            />
            <span class="text-sm text-slate-400">kg</span>
            <button
              onclick={handleLogWeight}
              disabled={loggingWeight || !logWeight}
              class="rounded-lg bg-joule-500 px-5 py-2.5 text-sm font-semibold text-slate-900 hover:bg-joule-400 transition disabled:opacity-50"
            >
              {loggingWeight ? 'Saving...' : 'Log'}
            </button>
            {#if weightSuccess}
              <span class="text-sm text-emerald-400">Saved!</span>
            {/if}
            {#if weightError}
              <span class="text-sm text-red-400">{weightError}</span>
            {/if}
          </div>
        </div>

      </div>
    {/if}
  </main>
</div>

<!-- Mobile bottom nav -->
<nav class="fixed bottom-0 left-0 right-0 z-10 flex border-t border-slate-800 bg-surface lg:hidden">
  <a href="/dashboard" class="flex flex-1 flex-col items-center gap-0.5 py-3 text-slate-400 hover:text-white transition">
    <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" /></svg>
    <span class="text-[10px]">Home</span>
  </a>
  <a href="/log" class="flex flex-1 flex-col items-center gap-0.5 py-3 text-slate-400 hover:text-white transition">
    <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
    <span class="text-[10px]">Log</span>
  </a>
  <a href="/coach" class="flex flex-1 flex-col items-center gap-0.5 py-3 text-slate-400 hover:text-white transition">
    <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" /></svg>
    <span class="text-[10px]">Coach</span>
  </a>
  <a href="/progress" class="flex flex-1 flex-col items-center gap-0.5 py-3 text-joule-400">
    <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" /></svg>
    <span class="text-[10px]">Progress</span>
  </a>
  <a href="/achievements" class="flex flex-1 flex-col items-center gap-0.5 py-3 text-slate-400 hover:text-white transition">
    <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" /></svg>
    <span class="text-[10px]">Awards</span>
  </a>
</nav>
