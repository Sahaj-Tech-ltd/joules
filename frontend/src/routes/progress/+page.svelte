<script lang="ts">
  import Sidebar from '$components/Sidebar.svelte';
  import ThemeToggle from '$components/ThemeToggle.svelte';
  import type { UserProfile } from '$lib/stores';
  import { authToken, userGoals } from '$lib/stores';
  import { api } from '$lib/api';
  import { goto } from '$app/navigation';
  import { onMount, onDestroy } from 'svelte';
  import { get } from 'svelte/store';
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
    is_cheat_day: boolean;
  }

  let weightHistory = $state<WeightEntry[]>([]);
  let weekSummaries = $state<DailySummary[]>([]);
  let goals = $state(get(userGoals));
  let profile = $state<UserProfile | null>(null);
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

  // BMI
  let bmi = $derived(
    latestWeight !== null && profile?.height_cm
      ? +(latestWeight / Math.pow(profile.height_cm / 100, 2)).toFixed(1)
      : null
  );
  let bmiCategory = $derived(
    bmi === null ? null
      : bmi < 18.5 ? { label: 'Underweight', color: 'text-blue-400', bg: 'bg-blue-500' }
      : bmi < 25   ? { label: 'Healthy',      color: 'text-emerald-400', bg: 'bg-emerald-500' }
      : bmi < 30   ? { label: 'Overweight',   color: 'text-yellow-400', bg: 'bg-yellow-500' }
                   : { label: 'Obese',         color: 'text-red-400', bg: 'bg-red-500' }
  );
  // BMI bar position (15–40 scale → 0–100%)
  let bmiBarPct = $derived(bmi !== null ? Math.min(100, Math.max(0, ((bmi - 15) / 25) * 100)) : null);

  // Weight goal progress
  let weightGoalPct = $derived(
    (() => {
      if (!profile?.target_weight_kg || firstWeight === null || latestWeight === null) return null;
      const target = profile.target_weight_kg;
      const start = firstWeight;
      if (start === target) return 100;
      const pct = ((start - latestWeight) / (start - target)) * 100;
      return Math.min(100, Math.max(0, Math.round(pct)));
    })()
  );

  // Days on calorie target (last 7 days)
  let daysOnTarget = $derived(
    goals?.daily_calorie_target
      ? weekSummaries.filter(d => d.total_calories > 0 && d.total_calories <= goals!.daily_calorie_target).length
      : null
  );

  // 7-day macro averages
  let avgProtein = $derived(
    weekSummaries.length > 0
      ? Math.round(weekSummaries.reduce((s, d) => s + d.total_protein, 0) / weekSummaries.length)
      : null
  );
  let avgCarbs = $derived(
    weekSummaries.length > 0
      ? Math.round(weekSummaries.reduce((s, d) => s + d.total_carbs, 0) / weekSummaries.length)
      : null
  );
  let avgFat = $derived(
    weekSummaries.length > 0
      ? Math.round(weekSummaries.reduce((s, d) => s + d.total_fat, 0) / weekSummaries.length)
      : null
  );

  function buildWeightChart() {
    weightChart?.destroy();
    if (!weightCanvas || weightHistory.length < MIN_WEIGHT_ENTRIES) return;
    const style = getComputedStyle(document.documentElement);
    const chart1 = style.getPropertyValue('--chart-1').trim();
    const borderVar = style.getPropertyValue('--border').trim();
    const mutedFg = style.getPropertyValue('--muted-foreground').trim();
    const lineColor = `oklch(${chart1})`;
    const gridColor = `oklch(${borderVar} / 0.4)`;
    const tickColor = `oklch(${mutedFg})`;
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
          borderColor: lineColor,
          backgroundColor: `oklch(${chart1} / 0.08)`,
          fill: true,
          tension: 0.35,
          pointBackgroundColor: lineColor,
          pointBorderColor: lineColor,
          pointRadius: 4,
          pointHoverRadius: 6,
          borderWidth: 2
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          x: { grid: { color: gridColor }, ticks: { color: tickColor, font: { size: 11 } } },
          y: { grid: { color: gridColor }, ticks: { color: tickColor, font: { size: 11 } }, beginAtZero: false }
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
    const style = getComputedStyle(document.documentElement);
    const chart1 = style.getPropertyValue('--chart-1').trim();
    const destructive = style.getPropertyValue('--destructive').trim();
    const borderVar = style.getPropertyValue('--border').trim();
    const mutedFg = style.getPropertyValue('--muted-foreground').trim();
    const gridColor = `oklch(${borderVar} / 0.4)`;
    const tickColor = `oklch(${mutedFg})`;
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
            backgroundColor: weekSummaries.map(d => {
              if (target > 0 && d.total_calories <= target) return `oklch(${chart1} / 0.7)`;
              if (d.is_cheat_day) return `oklch(${chart1} / 0.45)`;
              return `oklch(${destructive} / 0.6)`;
            }),
            borderRadius: 6,
            borderSkipped: false
          }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          x: { grid: { display: false }, ticks: { color: tickColor, font: { size: 10 } } },
          y: {
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { size: 11 } },
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

    const fmt = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

    try {
      const [weights, profileData] = await Promise.all([
        api.get<WeightEntry[]>(`/weight?from=${fmt(from30)}&to=${fmt(now)}`),
        api.get<UserProfile>('/user/profile')
      ]);
      weightHistory = (weights ?? []).sort((a, b) => a.date.localeCompare(b.date));
      profile = profileData;

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

<div class="flex min-h-screen overflow-x-hidden">
  <Sidebar activePage="progress" />

  <main class="flex-1 min-w-0 overflow-x-hidden p-4 lg:p-8" style="padding-bottom: calc(5rem + env(safe-area-inset-bottom, 0px));">
    <div class="mb-6 flex items-center justify-between">
      <div>
        <h1 class="font-display text-2xl font-bold text-foreground">Progress</h1>
        <p class="mt-0.5 text-xs text-muted-foreground">Your trends over time</p>
      </div>
      <div class="flex items-center gap-2">
        <ThemeToggle />
        <button
          onclick={() => { authToken.set(null); goto('/login'); }}
          class="rounded-xl border border-border bg-accent/50 px-3 py-1.5 text-xs font-medium text-foreground hover:text-foreground hover:bg-accent/50 transition"
        >
          Sign out
        </button>
      </div>
    </div>

    {#if loading}
      <div class="flex h-64 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-border border-t-primary"></div>
      </div>
    {:else}
      <div class="space-y-5">

        <!-- Stats row -->
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
          <div class="rounded-2xl border border-border bg-card p-5">
            <p class="text-xs text-foreground">Current Weight</p>
            {#if latestWeight !== null}
              <p class="mt-1.5 font-display text-3xl font-bold text-foreground">{latestWeight}<span class="text-sm font-normal text-muted-foreground ml-1">kg</span></p>
            {:else}
              <p class="mt-1 text-sm text-muted-foreground">Not logged yet</p>
            {/if}
          </div>
          <div class="rounded-2xl border border-border bg-card p-5">
            <p class="text-xs text-foreground">30-day change</p>
            {#if weightChange !== null}
              <p class="mt-1.5 font-display text-3xl font-bold {weightChange <= 0 ? 'text-emerald-400' : 'text-red-400'}">
                {weightChange > 0 ? '+' : ''}{weightChange}<span class="text-sm font-normal text-muted-foreground ml-1">kg</span>
              </p>
            {:else}
              <p class="mt-1 text-sm text-muted-foreground">Not enough data</p>
            {/if}
          </div>
          <div class="rounded-2xl border border-border bg-card p-5">
            <p class="text-xs text-foreground">7-day avg calories</p>
            {#if avgCalories !== null}
              <p class="mt-1.5 font-display text-3xl font-bold text-foreground">{avgCalories.toLocaleString()}<span class="text-sm font-normal text-muted-foreground ml-1">kcal</span></p>
              {#if goals?.daily_calorie_target}
                <p class="mt-0.5 text-xs text-muted-foreground">Target: {goals.daily_calorie_target.toLocaleString()} kcal</p>
              {/if}
            {:else}
              <p class="mt-1 text-sm text-muted-foreground">No data yet</p>
            {/if}
          </div>
          <div class="rounded-2xl border border-border bg-card p-5">
            <p class="text-xs text-foreground">7-day avg water</p>
            {#if avgWater !== null}
              <p class="mt-1.5 font-display text-3xl font-bold text-blue-400">{avgWater.toLocaleString()}<span class="text-sm font-normal text-muted-foreground ml-1">ml</span></p>
              <p class="mt-0.5 text-xs text-muted-foreground">Target: 2,500 ml</p>
            {:else}
              <p class="mt-1 text-sm text-muted-foreground">No data yet</p>
            {/if}
          </div>
          <div class="rounded-2xl border border-border bg-card p-5">
            <p class="text-xs text-foreground">BMI</p>
            {#if bmi !== null && bmiCategory !== null}
              <p class="mt-1.5 font-display text-3xl font-bold {bmiCategory.color}">{bmi}</p>
              <span class="mt-1 inline-block text-[10px] font-semibold px-2 py-0.5 rounded-full {bmiCategory.bg}/20 {bmiCategory.color}">{bmiCategory.label}</span>
            {:else}
              <p class="mt-1 text-sm text-muted-foreground">Log weight + height</p>
            {/if}
          </div>
        </div>

        <!-- BMI detail -->
        {#if bmi !== null && bmiCategory !== null && bmiBarPct !== null}
          <div class="rounded-2xl border border-border bg-card p-5">
            <div class="flex items-center justify-between mb-3">
              <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Body Mass Index</p>
              <span class="font-display text-3xl font-bold {bmiCategory.color}">{bmi} <span class="text-sm font-normal text-foreground">BMI</span></span>
            </div>
            <!-- Gradient bar -->
            <div class="relative h-3 rounded-full overflow-hidden mb-2">
              <div class="absolute inset-0" style="background: linear-gradient(to right, oklch(0.7 0.15 250) 0%, oklch(0.75 0.15 160) 25%, oklch(0.8 0.15 85) 58%, oklch(0.65 0.2 25) 100%);"></div>
              <!-- Marker -->
              <div class="absolute top-0 h-full w-1 rounded-full bg-white shadow-lg" style="left: calc({bmiBarPct}% - 2px);"></div>
            </div>
            <div class="flex justify-between text-[10px] text-muted-foreground mb-3">
              <span>Under</span><span>Healthy</span><span>Over</span><span>Obese</span>
            </div>
            <p class="text-xs text-foreground">
              {#if bmiCategory.label === 'Healthy'}
                You're in the healthy range.
              {:else if bmiCategory.label === 'Underweight'}
                Your BMI is below the healthy range (18.5–24.9).
              {:else}
                Your goal weight will bring you closer to the healthy range (18.5–24.9).
              {/if}
            </p>
          </div>
        {/if}

        <!-- Weight chart -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <div class="mb-4 flex items-center justify-between">
            <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Weight Trend (30 days)</p>
            <span class="text-xs text-muted-foreground">{weightHistory.length} entries</span>
          </div>
          {#if weightHistory.length < MIN_WEIGHT_ENTRIES}
            <div class="flex h-48 flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border">
              <svg class="h-8 w-8 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
              <p class="text-sm text-foreground">Not enough data yet</p>
              <p class="text-xs text-muted-foreground">Log at least {MIN_WEIGHT_ENTRIES} weigh-ins to see your trend · You have {weightHistory.length}</p>
            </div>
          {:else}
            <div class="h-56">
              <canvas bind:this={weightCanvas}></canvas>
            </div>
          {/if}
        </div>

        <!-- Calorie chart -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <div class="mb-4 flex items-center justify-between">
            <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Daily Calories (last 7 days)</p>
            {#if goals?.daily_calorie_target}
              <div class="flex items-center gap-3 text-xs">
                <span class="flex items-center gap-1"><span class="inline-block h-2.5 w-2.5 rounded-sm bg-primary/70"></span>On target</span>
                <span class="flex items-center gap-1"><span class="inline-block h-2.5 w-2.5 rounded-sm bg-red-500/60"></span>Over target</span>
                <span class="flex items-center gap-1"><span class="inline-block h-2.5 w-2.5 rounded-sm bg-primary/45 opacity-60"></span>Cheat day</span>
              </div>
            {/if}
          </div>
          {#if weekSummaries.length < MIN_CALORIE_ENTRIES}
            <div class="flex h-48 flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border">
              <svg class="h-8 w-8 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" />
              </svg>
              <p class="text-sm text-foreground">Not enough data yet</p>
              <p class="text-xs text-muted-foreground">Log meals for at least {MIN_CALORIE_ENTRIES} days to see the chart · <a href="/log" class="text-primary hover:underline">Log a meal</a></p>
            </div>
          {:else}
            <div class="h-56">
              <canvas bind:this={calorieCanvas}></canvas>
            </div>
          {/if}
        </div>

        <!-- Goal Progress -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Goal Progress</p>
          <div class="space-y-4">

            <!-- Weight goal -->
            {#if profile?.target_weight_kg && latestWeight !== null}
              <div>
                <div class="flex items-center justify-between mb-1.5">
                  <span class="text-xs text-foreground">Weight goal</span>
                  <span class="text-xs font-medium text-foreground/80">
                    {latestWeight} kg → {profile.target_weight_kg} kg
                    {#if weightGoalPct !== null}
                      <span class="text-muted-foreground ml-1">({weightGoalPct}%)</span>
                    {/if}
                  </span>
                </div>
                <div class="h-2 rounded-full bg-accent overflow-hidden">
                  <div class="h-full rounded-full bg-primary transition-all duration-500" style="width: {weightGoalPct ?? 0}%"></div>
                </div>
              </div>
            {/if}

            <!-- Calorie adherence (last 7 days) -->
            {#if daysOnTarget !== null && weekSummaries.length > 0}
              <div>
                <div class="flex items-center justify-between mb-2">
                  <span class="text-xs text-foreground">Calorie target</span>
                  <span class="text-xs font-medium text-foreground/80">{daysOnTarget}/{weekSummaries.length} days on target</span>
                </div>
                <div class="flex gap-1.5">
                  {#each weekSummaries as day}
                    {@const onTarget = goals?.daily_calorie_target && day.total_calories > 0 && day.total_calories <= goals.daily_calorie_target}
                    {@const noData = day.total_calories === 0}
                    <div
                      class="flex-1 h-6 rounded-md flex items-center justify-center text-[9px] font-bold {noData ? 'bg-accent text-muted-foreground' : onTarget ? 'bg-primary/80 text-primary-foreground' : day.is_cheat_day ? 'bg-primary/30 text-primary' : 'bg-red-500/70 text-foreground'}"
                      title="{new Date(day.date).toLocaleDateString('en-US', {weekday:'short'})}: {day.total_calories} kcal"
                    >
                      {new Date(day.date).toLocaleDateString('en-US', { weekday: 'narrow' })}
                    </div>
                  {/each}
                </div>
              </div>
            {/if}

            <!-- Macro averages vs targets -->
            {#if goals && (avgProtein !== null || avgCarbs !== null || avgFat !== null)}
              <div class="space-y-2.5 pt-1 border-t border-border">
                <p class="text-xs text-muted-foreground pt-1">7-day macro averages</p>
                {#if avgProtein !== null && goals.daily_protein_g > 0}
                  {@const pct = Math.min(100, Math.round((avgProtein / goals.daily_protein_g) * 100))}
                  <div>
                    <div class="flex justify-between mb-1 text-[11px]">
                      <span class="text-foreground">Protein</span>
                      <span class="text-foreground">{avgProtein}g <span class="text-muted-foreground">/ {goals.daily_protein_g}g ({pct}%)</span></span>
                    </div>
                    <div class="h-2 rounded-full bg-accent/50 overflow-hidden">
                      <div class="h-full rounded-full bg-blue-500 transition-all" style="width:{pct}%"></div>
                    </div>
                  </div>
                {/if}
                {#if avgCarbs !== null && goals.daily_carbs_g > 0}
                  {@const pct = Math.min(100, Math.round((avgCarbs / goals.daily_carbs_g) * 100))}
                  <div>
                    <div class="flex justify-between mb-1 text-[11px]">
                      <span class="text-foreground">Carbs</span>
                      <span class="text-foreground">{avgCarbs}g <span class="text-muted-foreground">/ {goals.daily_carbs_g}g ({pct}%)</span></span>
                    </div>
                    <div class="h-2 rounded-full bg-accent/50 overflow-hidden">
                      <div class="h-full rounded-full bg-purple-500 transition-all" style="width:{pct}%"></div>
                    </div>
                  </div>
                {/if}
                {#if avgFat !== null && goals.daily_fat_g > 0}
                  {@const pct = Math.min(100, Math.round((avgFat / goals.daily_fat_g) * 100))}
                  <div>
                    <div class="flex justify-between mb-1 text-[11px]">
                      <span class="text-foreground">Fat</span>
                      <span class="text-foreground">{avgFat}g <span class="text-muted-foreground">/ {goals.daily_fat_g}g ({pct}%)</span></span>
                    </div>
                    <div class="h-2 rounded-full bg-accent/50 overflow-hidden">
                      <div class="h-full rounded-full bg-emerald-500 transition-all" style="width:{pct}%"></div>
                    </div>
                  </div>
                {/if}
              </div>
            {/if}

          </div>
        </div>

        <!-- Log weight widget -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Log Today's Weight</p>
          <div class="flex items-center gap-2.5">
            <input
              type="number"
              bind:value={logWeight}
              placeholder="e.g. 72.5"
              min="1"
              step="0.1"
              class="flex-1 rounded-xl border border-border bg-secondary px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
            />
            <span class="text-sm text-muted-foreground flex-shrink-0">kg</span>
            <button
              onclick={handleLogWeight}
              disabled={loggingWeight || !logWeight}
              class="rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50 flex-shrink-0"
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
