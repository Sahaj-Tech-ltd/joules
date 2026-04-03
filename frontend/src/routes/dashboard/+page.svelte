<script lang="ts">
  import Logo from '$components/Logo.svelte';
  import Sidebar from '$components/Sidebar.svelte';
  import MacroRing from '$components/MacroRing.svelte';
  import PetWidget from '$components/PetWidget.svelte';
  import BannerStrip from '$components/BannerStrip.svelte';
  import WeightChart from '$components/WeightChart.svelte';
  import MealList from '$components/MealList.svelte';
  import WaterWidget from '$components/WaterWidget.svelte';
  import ExerciseWidget from '$components/ExerciseWidget.svelte';
  import StepsWidget from '$components/StepsWidget.svelte';
  import TipsWidget from '$components/TipsWidget.svelte';
  import ThemeToggle from '$components/ThemeToggle.svelte';
  import Walkthrough from '$components/Walkthrough.svelte';
  import { authToken, userProfile, userGoals } from '$lib/stores';
  import { api } from '$lib/api';
  import { showAchievement } from '$lib/achievements';
  import { defaultUnits, displayEnergy, type UnitPrefs } from '$lib/units';
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { goto } from '$app/navigation';
  import type { UserProfile, UserGoals, FastingStatus } from '$lib/stores';

  interface HabitSummary {
    total_points: number;
    level: number;
    level_name: string;
    level_progress_pct: number;
    next_level_at: number;
    streak_days: number;
    pet_mood: string;
    today_points: number;
    today_checked_in: boolean;
  }

  interface SummaryResponse {
    date: string;
    total_calories: number;
    total_protein: number;
    total_carbs: number;
    total_fat: number;
    total_fiber: number;
    total_burned: number;
    total_water_ml: number;
    total_steps: number;
    meals: MealResponse[];
    is_cheat_day: boolean;
  }

  interface MealResponse {
    id: string;
    timestamp: string;
    meal_type: string;
    photo_path?: string;
    note?: string;
    foods: FoodItemResponse[];
  }

  interface FoodItemResponse {
    id: string;
    name: string;
    calories: number;
    protein_g: number;
    carbs_g: number;
    fat_g: number;
    fiber_g: number;
    serving_size?: string;
    source: string;
  }

  interface WeightResponse {
    date: string;
    weight_kg: number;
  }

  interface ExerciseResponse {
    id: string;
    name: string;
    duration_min: number;
    calories_burned: number;
    timestamp: string;
  }

  let profile = $state<UserProfile | null>(null);
  let goals = $state<UserGoals | null>(null);
  let summary = $state<SummaryResponse | null>(null);
  let unitPrefs = $state<UnitPrefs>({ ...defaultUnits });
  let recentMeals = $state<MealResponse[]>([]);
  let weightHistory = $state<WeightResponse[]>([]);
  let exercises = $state<ExerciseResponse[]>([]);
  let tips = $state<string | null>(null);
  let fastingStatus = $state<FastingStatus | null>(null);
  let weightInput = $state('');
  let weightLoading = $state(false);
  let loading = $state(true);
  let weightSuccess = $state(false);
  let cheatDayLoading = $state(false);
  let habitSummary = $state<HabitSummary | null>(null);
  let quickAddFavorites = $state<Array<{id: string; name: string; calories: number}>>([]);
  let exportFrom = $state('');
  let exportTo = $state('');
  let exportType = $state('meals');
  let exportFormat = $state('csv');

  onMount(() => {
    const unsub = authToken.subscribe((token) => {
      if (!token) goto('/login');
    });

    const today = new Date().toISOString().split('T')[0];
    const thirtyDaysAgo = new Date();
    thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
    const from = thirtyDaysAgo.toISOString().split('T')[0];
    const to = today;

    async function checkAchievements() {
      try {
        const unlocked = await api.post<{ id: string; title: string; description: string; unlocked_at: string }[]>('/achievements/check', {});
        if (Array.isArray(unlocked)) {
          const twoMinAgo = Date.now() - 2 * 60 * 1000;
          for (const a of unlocked) {
            if (new Date(a.unlocked_at).getTime() >= twoMinAgo) {
              showAchievement({ id: a.id, title: a.title, description: a.description });
            }
          }
        }
      } catch {}
    }

    async function refreshDashboard() {
      // Load unit prefs from localStorage (fast, no extra API call)
      try {
        const stored = localStorage.getItem('unit_prefs');
        if (stored) unitPrefs = { ...defaultUnits, ...JSON.parse(stored) };
      } catch {}
      try {
        const [p, g, s, m, w, ex, hs] = await Promise.all([
          api.get<UserProfile>('/user/profile'),
          api.get<UserGoals>('/user/goals'),
          api.get<SummaryResponse>(`/dashboard/summary?date=${today}`),
          api.get<MealResponse[]>('/meals/recent'),
          api.get<WeightResponse[]>(`/weight?from=${from}&to=${to}`),
          api.get<ExerciseResponse[]>(`/exercises?date=${today}`),
          api.get<HabitSummary>('/habits/summary').catch(() => null)
        ]);
        profile = p;
        goals = g;
        summary = s;
        recentMeals = m;
        weightHistory = w;
        exercises = ex;
        habitSummary = hs;
        userProfile.set(p);
        userGoals.set(g);
        api.get<{ tips: string }>('/coach/tips').then(t => { tips = t.tips; }).catch(() => { tips = ''; });
        api.get<Array<{id: string; name: string; calories: number}>>('/favorites/top').then(f => { quickAddFavorites = f; }).catch(() => {});
        api.get<FastingStatus>('/fasting/status').then(s => { fastingStatus = s; }).catch(() => {});
        api.post<HabitSummary>('/habits/checkin', {}).then(r => { if (r) habitSummary = r; }).catch(() => {});
        checkAchievements();
      } catch {}
      finally { loading = false; }
    }

    refreshDashboard();

    const onWaterLogged = () => refreshDashboard();
    const onExerciseLogged = () => refreshDashboard();
    const onStepsUpdated = () => refreshDashboard();
    window.addEventListener('water-logged', onWaterLogged);
    window.addEventListener('exercise-logged', onExerciseLogged);
    window.addEventListener('steps-updated', onStepsUpdated);

    return () => {
      unsub();
      window.removeEventListener('water-logged', onWaterLogged);
      window.removeEventListener('exercise-logged', onExerciseLogged);
      window.removeEventListener('steps-updated', onStepsUpdated);
    };
  });

  let greeting = $derived(
    (() => {
      const hour = new Date().getHours();
      if (hour < 12) return 'Good morning';
      if (hour < 17) return 'Good afternoon';
      return 'Good evening';
    })()
  );

  let todayFormatted = $derived(
    new Date().toLocaleDateString('en-US', {
      weekday: 'long',
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    })
  );

  let isOverCalories = $derived(
    goals !== null && summary !== null && summary.total_calories > goals.daily_calorie_target
  );

  let todayWeightLogged = $derived(
    weightHistory.some(w => w.date === new Date().toISOString().split('T')[0])
  );

  async function toggleCheatDay() {
    if (!summary) return;
    cheatDayLoading = true;
    try {
      const today = new Date().toISOString().split('T')[0];
      if (summary.is_cheat_day) {
        await api.del(`/dashboard/cheat-day?date=${today}`);
        summary = { ...summary, is_cheat_day: false };
      } else {
        await api.post('/dashboard/cheat-day', { date: today });
        summary = { ...summary, is_cheat_day: true };
      }
    } catch {}
    finally { cheatDayLoading = false; }
  }

  async function logWeight() {
    if (!weightInput) return;
    weightLoading = true;
    try {
      const w = parseFloat(weightInput);
      if (w <= 0) return;
      await api.post('/weight', { weight_kg: w });
      weightInput = '';
      weightSuccess = true;
      setTimeout(() => { weightSuccess = false; }, 2000);
      const thirtyDaysAgo = new Date();
      thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
      const from = thirtyDaysAgo.toISOString().split('T')[0];
      const to = new Date().toISOString().split('T')[0];
      weightHistory = await api.get(`/weight?from=${from}&to=${to}`);
      if (profile) {
        profile = { ...profile, weight_kg: w };
        userProfile.set(profile);
      }
    } catch {}
    finally { weightLoading = false; }
  }

  async function downloadExport() {
    const params = new URLSearchParams();
    if (exportType) params.set('type', exportType);
    if (exportFrom) params.set('from', exportFrom);
    if (exportTo) params.set('to', exportTo);
    const url = `/api/export/${exportFormat}?${params.toString()}`;
    const token = get(authToken);
    const res = await fetch(url, { headers: { 'Authorization': `Bearer ${token}` } });
    if (!res.ok) return;
    const blob = await res.blob();
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = `joule-export.${exportFormat}`;
    a.click();
    URL.revokeObjectURL(a.href);
  }
</script>

<div class="flex min-h-screen overflow-x-hidden">
  <Sidebar activePage="dashboard" isAdmin={profile?.is_admin ?? false} />

  <main class="flex-1 min-w-0 overflow-y-auto overflow-x-hidden p-4 lg:p-8" style="padding-bottom: calc(5rem + env(safe-area-inset-bottom, 0px));">
    {#if loading}
      <div class="flex h-64 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-border border-t-primary"></div>
      </div>
    {:else if profile && profile.onboarding_complete && goals}
      <div class="space-y-5 md:space-y-6">
        <Walkthrough />
        <BannerStrip />

        <!-- Header -->
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3.5 min-w-0 overflow-hidden">
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br from-primary/80 to-primary font-display text-sm font-bold text-primary-foreground shadow-lg shadow-primary/20 ring-2 ring-ring/20 select-none">
              {profile.name.split(' ').map((n: string) => n[0]).filter(Boolean).join('').slice(0, 2).toUpperCase()}
            </div>
            <div class="min-w-0">
              <h1 class="font-display text-lg font-bold text-primary-foreground leading-tight truncate lg:text-[22px]">{greeting}, {profile.name.split(' ')[0]}</h1>
              <p class="text-xs text-muted-foreground mt-0.5 truncate">{todayFormatted}</p>
            </div>
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

        <!-- Macro rings -->
        <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <MacroRing
            consumed={summary?.total_calories ?? 0}
            target={goals.daily_calorie_target}
            displayConsumed={displayEnergy(summary?.total_calories ?? 0, unitPrefs.energy_unit)}
            displayTarget={displayEnergy(goals.daily_calorie_target, unitPrefs.energy_unit)}
            label="Calories"
            unit={unitPrefs.energy_unit}
            color="#f59e0b"
          />
          <MacroRing consumed={summary?.total_protein ?? 0} target={goals.daily_protein_g} label="Protein" unit="g" color="#3b82f6" />
          <MacroRing consumed={summary?.total_carbs ?? 0} target={goals.daily_carbs_g} label="Carbs" unit="g" color="#8b5cf6" />
          <MacroRing consumed={summary?.total_fat ?? 0} target={goals.daily_fat_g} label="Fat" unit="g" color="#10b981" />
        </div>

        <!-- Quick actions -->
        <div class="flex flex-wrap gap-2.5">
          <a
            href="/log"
            class="inline-flex items-center gap-2 rounded-2xl bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground shadow-lg shadow-primary/25 hover:bg-primary/80 active:scale-95 transition-all"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" /></svg>
            Log a Meal
          </a>
          <a
            href="/coach"
            class="inline-flex items-center gap-2 rounded-2xl bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground shadow-lg shadow-primary/25 hover:bg-primary/80 active:scale-95 transition-all"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" /></svg>
            Chat with Coach
          </a>
          {#if isOverCalories || summary?.is_cheat_day}
            <button
              onclick={toggleCheatDay}
              disabled={cheatDayLoading}
              class="inline-flex items-center gap-2 rounded-2xl border px-5 py-2.5 text-sm font-semibold transition-all active:scale-95 disabled:opacity-50 {summary?.is_cheat_day ? 'border-orange-500/40 bg-orange-500/10 text-orange-400 hover:bg-orange-500/20' : 'border-border bg-accent/50 text-foreground hover:border-orange-500/40 hover:text-orange-400'}"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5 3l14 9-14 9V3z" /></svg>
              {summary?.is_cheat_day ? 'Cheat Day On' : 'Mark Cheat Day'}
            </button>
          {/if}
        </div>

        <!-- Main content grid -->
        <div class="grid gap-4 lg:grid-cols-2 lg:gap-5">
          <!-- Left column -->
          <div class="space-y-4">
            {#if summary && summary.meals.length > 0}
              <MealList meals={summary.meals} onload={() => refreshDashboard()} />
            {:else}
              <div class="rounded-2xl border border-border bg-card p-6">
                <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Today's Meals</p>
                <p class="mt-3 text-sm text-muted-foreground">No meals logged yet.</p>
                <a href="/log" class="mt-3 inline-flex items-center gap-1.5 text-sm text-primary hover:text-primary/80 transition">
                  <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" /></svg>
                  Log your first meal
                </a>
              </div>
            {/if}

            <div class="rounded-2xl border {!todayWeightLogged ? 'border-primary/25 bg-primary/5' : 'border-border bg-card'} p-5">
              <div class="flex items-center justify-between mb-4">
                <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Weight</p>
                {#if !todayWeightLogged}
                  <span class="text-[11px] text-primary font-medium">Log today →</span>
                {/if}
              </div>
              <div class="flex items-baseline gap-3 mb-4 flex-wrap">
                <div>
                  <p class="text-[10px] text-muted-foreground mb-0.5">Current</p>
                  <p class="font-display text-2xl font-bold text-foreground">{profile.weight_kg ?? '—'} <span class="text-sm font-normal text-muted-foreground">kg</span></p>
                </div>
                <div>
                  <p class="text-[10px] text-muted-foreground mb-0.5">Target</p>
                  <p class="font-display text-2xl font-bold text-foreground">{profile.target_weight_kg ?? '—'} <span class="text-sm font-normal text-muted-foreground">kg</span></p>
                </div>
                {#if summary && summary.total_burned > 0}
                  <div>
                    <p class="text-[10px] text-muted-foreground mb-0.5">Burned</p>
                    <p class="font-display text-2xl font-bold text-orange-400">{summary.total_burned} <span class="text-sm font-normal text-muted-foreground">kcal</span></p>
                  </div>
                {/if}
              </div>
              <form onsubmit={(e) => { e.preventDefault(); logWeight(); }} class="flex items-center gap-2">
                <input
                  type="number"
                  step="0.1"
                  min="0"
                  bind:value={weightInput}
                  placeholder="e.g. 72.5"
                  disabled={weightLoading}
                  class="flex-1 rounded-xl border border-border bg-secondary px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 disabled:opacity-50"
                />
                <span class="text-sm text-muted-foreground">kg</span>
                <button
                  type="submit"
                  disabled={weightLoading || !weightInput}
                  class="rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50"
                >
                  {weightLoading ? '...' : 'Log'}
                </button>
                {#if weightSuccess}
                  <span class="text-xs text-emerald-400">Saved!</span>
                {/if}
              </form>
            </div>

             <WaterWidget totalMl={summary?.total_water_ml ?? 0} />

            {#if quickAddFavorites.length > 0}
              <div class="rounded-2xl border border-border bg-card p-5">
                <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-3">Quick Add</p>
                <div class="flex gap-2 overflow-x-auto pb-1">
                  {#each quickAddFavorites as fav}
                    <button
                      type="button"
                      onclick={async () => {
                        await api.post('/meals', {
                          meal_type: 'snack',
                          foods: [{ name: fav.name, calories: fav.calories, protein_g: 0, carbs_g: 0, fat_g: 0, fiber_g: 0, serving_size: '' }]
                        });
                        refreshDashboard();
                      }}
                      class="flex-none rounded-xl border border-border bg-card/60 px-3 py-2 text-left hover:border-primary/30 hover:bg-primary/5 transition min-w-[100px]"
                    >
                      <p class="truncate text-xs font-medium text-foreground max-w-[90px]">{fav.name}</p>
                      <p class="text-[11px] text-muted-foreground">{fav.calories} kcal</p>
                    </button>
                  {/each}
                </div>
              </div>
            {/if}
           </div>

           <!-- Right column -->
           <div class="space-y-4">
             <WeightChart data={weightHistory} />

             <ExerciseWidget exercises={exercises} />

             <StepsWidget />

             {#if habitSummary}
               <div class="rounded-2xl border border-border bg-card p-5 flex flex-col items-center">
                 <PetWidget
                   mood={habitSummary.pet_mood}
                   streak_days={habitSummary.streak_days}
                   level={habitSummary.level}
                   level_name={habitSummary.level_name}
                 />
                 <div class="w-full mt-4">
                   <div class="flex items-center justify-between mb-1.5">
                     <span class="text-xs text-foreground">{habitSummary.total_points} XP</span>
                     <span class="text-xs text-muted-foreground">next lv. at {habitSummary.next_level_at}</span>
                   </div>
                   <div class="h-2 w-full rounded-full bg-accent/50 overflow-hidden">
                     <div class="h-full rounded-full bg-gradient-to-r from-primary to-primary/80 transition-all" style="width:{habitSummary.level_progress_pct}%"></div>
                   </div>
                 </div>
                 {#if habitSummary.today_points > 0}
                   <p class="mt-2 text-xs text-primary font-medium">+{habitSummary.today_points} XP earned today</p>
                 {/if}
               </div>
             {/if}

             <TipsWidget {tips} />

             <div class="rounded-2xl border border-border bg-card p-5">
               <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Profile</p>
               <div class="space-y-3">
                 <div>
                   <p class="text-[10px] text-muted-foreground">Diet Plan</p>
                   <p class="mt-0.5 text-sm font-medium text-foreground">{goals.diet_plan || 'Not set'}</p>
                 </div>
                 <div>
                   <p class="text-[10px] text-muted-foreground">Objective</p>
                   <p class="mt-0.5 text-sm font-medium text-foreground">{goals.objective || 'Not set'}</p>
                 </div>
                 <div>
                   <p class="text-[10px] text-muted-foreground">Activity Level</p>
                   <p class="mt-0.5 text-sm font-medium text-foreground">{profile.activity_level || 'Not set'}</p>
                 </div>
                </div>
              </div>

            <div class="rounded-2xl border border-border bg-card p-5">
              <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-3">Export Data</p>
              <div class="space-y-3">
                <div class="grid grid-cols-2 gap-2">
                  <div>
                    <label class="text-[10px] text-muted-foreground">From</label>
                    <input type="date" bind:value={exportFrom} class="w-full rounded-lg border border-border bg-secondary px-2.5 py-1.5 text-xs text-foreground" />
                  </div>
                  <div>
                    <label class="text-[10px] text-muted-foreground">To</label>
                    <input type="date" bind:value={exportTo} class="w-full rounded-lg border border-border bg-secondary px-2.5 py-1.5 text-xs text-foreground" />
                  </div>
                </div>
                <div class="flex gap-2">
                  <select bind:value={exportType} class="flex-1 rounded-lg border border-border bg-secondary px-2.5 py-1.5 text-xs text-foreground">
                    <option value="meals">Meals</option>
                    <option value="weight">Weight</option>
                    <option value="water">Water</option>
                    <option value="exercise">Exercise</option>
                  </select>
                  <select bind:value={exportFormat} class="rounded-lg border border-border bg-secondary px-2.5 py-1.5 text-xs text-foreground">
                    <option value="csv">CSV</option>
                    <option value="json">JSON</option>
                  </select>
                </div>
                <button
                  type="button"
                  onclick={downloadExport}
                  class="block w-full text-center rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground hover:bg-primary/80 transition"
                >
                  Download
                </button>
              </div>
            </div>

              {#if goals.diet_plan === 'intermittent_fasting'}
               <div class="rounded-2xl border border-border bg-card p-5">
                 <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Fasting</p>
                 {#if fastingStatus}
                   <div class="flex items-center justify-between">
                     <div>
                       <div class="font-display text-lg font-semibold {fastingStatus.is_fasting ? 'text-emerald-400' : 'text-foreground'}">
                         {fastingStatus.is_fasting ? 'Fasting' : 'Eating Window'}
                       </div>
                       <div class="text-xs text-muted-foreground mt-0.5">
                         {fastingStatus.fasting_window} · starts {fastingStatus.eating_window_start}
                       </div>
                     </div>
                     <div class="text-right">
                       <div class="font-display text-2xl font-bold text-orange-400">{fastingStatus.fasting_streak}</div>
                       <div class="text-xs text-muted-foreground">day streak</div>
                     </div>
                   </div>
                   {#if fastingStatus.is_fasting}
                     {@const hoursLeft = Math.max(0, Math.floor(fastingStatus.seconds_remaining / 3600))}
                     {@const minsLeft = Math.max(0, Math.floor((fastingStatus.seconds_remaining % 3600) / 60))}
                     <div class="mt-4 rounded-xl bg-accent/50 px-4 py-3 text-center">
                       <div class="font-display text-lg font-semibold text-foreground/80">
                         {Math.floor(fastingStatus.seconds_elapsed / 3600)}h {Math.floor((fastingStatus.seconds_elapsed % 3600) / 60)}m elapsed
                       </div>
                       <div class="text-xs text-muted-foreground mt-0.5">{hoursLeft}h {minsLeft}m remaining</div>
                     </div>
                   {/if}
                 {:else}
                   <div class="text-sm text-muted-foreground">Loading...</div>
                 {/if}
               </div>
             {/if}
           </div>
         </div>
       </div>
     {:else}
       <div class="flex min-h-[60vh] items-center justify-center">
         <div class="rounded-2xl border border-border bg-card p-8 text-center max-w-sm">
           <Logo size={48} />
           <h2 class="mt-4 font-display text-xl font-bold text-foreground">Welcome to Joules</h2>
           <p class="mt-2 text-sm text-foreground leading-relaxed">
             Complete your setup to start tracking nutrition and hitting your goals.
           </p>
           <a
             href="/onboarding"
             class="mt-6 inline-flex items-center gap-2 rounded-2xl bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition shadow-lg shadow-primary/20"
           >
             Complete Setup
           </a>
         </div>
       </div>
     {/if}
   </main>
  </div>
