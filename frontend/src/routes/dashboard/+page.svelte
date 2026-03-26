<script lang="ts">
  import Logo from '$components/Logo.svelte';
  import MacroRing from '$components/MacroRing.svelte';
  import WeightChart from '$components/WeightChart.svelte';
  import MealList from '$components/MealList.svelte';
  import WaterWidget from '$components/WaterWidget.svelte';
  import ExerciseWidget from '$components/ExerciseWidget.svelte';
  import TipsWidget from '$components/TipsWidget.svelte';
  import ThemeToggle from '$components/ThemeToggle.svelte';
  import { authToken, userProfile, userGoals } from '$lib/stores';
  import { api } from '$lib/api';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import type { UserProfile, UserGoals } from '$lib/stores';

  interface SummaryResponse {
    date: string;
    total_calories: number;
    total_protein: number;
    total_carbs: number;
    total_fat: number;
    total_fiber: number;
    total_burned: number;
    total_water_ml: number;
    meals: MealResponse[];
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
  let recentMeals = $state<MealResponse[]>([]);
  let weightHistory = $state<WeightResponse[]>([]);
  let exercises = $state<ExerciseResponse[]>([]);
  let tips = $state<string | null>(null);
  let weightInput = $state('');
  let weightLoading = $state(false);
  let loading = $state(true);
  let weightSuccess = $state(false);

  onMount(() => {
    const unsub = authToken.subscribe((token) => {
      if (!token) goto('/login');
    });

    const today = new Date().toISOString().split('T')[0];
    const thirtyDaysAgo = new Date();
    thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
    const from = thirtyDaysAgo.toISOString().split('T')[0];
    const to = today;

    async function refreshDashboard() {
      try {
        const [p, g, s, m, w, ex] = await Promise.all([
          api.get<UserProfile>('/user/profile'),
          api.get<UserGoals>('/user/goals'),
          api.get<SummaryResponse>(`/dashboard/summary?date=${today}`),
          api.get<MealResponse[]>('/meals/recent'),
          api.get<WeightResponse[]>(`/weight?from=${from}&to=${to}`),
          api.get<ExerciseResponse[]>(`/exercises?date=${today}`)
        ]);
        profile = p;
        goals = g;
        summary = s;
        recentMeals = m;
        weightHistory = w;
        exercises = ex;
        userProfile.set(p);
        userGoals.set(g);
        api.get<{ tips: string }>('/coach/tips').then(t => { tips = t.tips; }).catch(() => {});
      } catch {}
      finally { loading = false; }
    }

    refreshDashboard();

    const onWaterLogged = () => refreshDashboard();
    const onExerciseLogged = () => refreshDashboard();
    window.addEventListener('water-logged', onWaterLogged);
    window.addEventListener('exercise-logged', onExerciseLogged);

    return () => {
      unsub();
      window.removeEventListener('water-logged', onWaterLogged);
      window.removeEventListener('exercise-logged', onExerciseLogged);
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
</script>

<div class="flex min-h-screen">
  <aside class="hidden w-64 border-r border-slate-800 bg-surface p-6 lg:block">
    <div class="flex items-center gap-3">
      <Logo size={32} />
      <span class="text-lg font-bold text-white">Joule</span>
    </div>
    <nav class="mt-8 space-y-1">
      <a
        href="/dashboard"
        class="flex items-center gap-3 rounded-lg bg-slate-800 px-3 py-2 text-sm font-medium text-white"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" /></svg>
        Dashboard
      </a>
      <a
        href="/log"
        class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
        Log Meal
      </a>
      <a
        href="/coach"
        class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" /></svg>
        Health Coach
      </a>
      <a
        href="/progress"
        class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" /></svg>
        Progress
      </a>
      <a
        href="/achievements"
        class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" /></svg>
        Achievements
      </a>
    </nav>
  </aside>

  <main class="flex-1 overflow-y-auto p-6 lg:p-10">
    {#if loading}
      <div class="flex h-64 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-slate-700 border-t-joule-500"></div>
      </div>
    {:else if profile && profile.onboarding_complete && goals}
      <div class="space-y-8">
        <div class="flex items-center justify-between">
          <div>
            <h1 class="text-2xl font-bold text-white">{greeting}, {profile.name}</h1>
            <p class="mt-1 text-sm text-slate-400">{todayFormatted}</p>
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

        <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <MacroRing consumed={summary?.total_calories ?? 0} target={goals.daily_calorie_target} label="Calories" unit="kcal" color="#f59e0b" />
          <MacroRing consumed={summary?.total_protein ?? 0} target={goals.daily_protein_g} label="Protein" unit="g" color="#3b82f6" />
          <MacroRing consumed={summary?.total_carbs ?? 0} target={goals.daily_carbs_g} label="Carbs" unit="g" color="#8b5cf6" />
          <MacroRing consumed={summary?.total_fat ?? 0} target={goals.daily_fat_g} label="Fat" unit="g" color="#10b981" />
        </div>

        <div class="flex gap-4">
          <a
            href="/log"
            class="rounded-lg bg-joule-500 px-6 py-3 text-sm font-semibold text-slate-900 hover:bg-joule-400 transition"
          >
            Log a Meal
          </a>
          <a
            href="/coach"
            class="rounded-lg border border-slate-700 bg-surface-light px-6 py-3 text-sm font-semibold text-white hover:bg-surface-lighter transition"
          >
            Chat with Coach
          </a>
        </div>

        <div class="grid gap-6 lg:grid-cols-2">
          <div class="space-y-6">
            {#if summary && summary.meals.length > 0}
              <MealList meals={summary.meals} />
            {:else}
              <div class="rounded-xl border border-slate-800 bg-surface-light p-6">
                <h3 class="text-sm font-semibold text-white">Today's Meals</h3>
                <p class="mt-4 text-sm text-slate-500">No meals logged today.</p>
              </div>
            {/if}

            <div class="rounded-xl border border-slate-800 bg-surface-light p-6">
              <h3 class="text-sm font-semibold text-white">Weight Log</h3>
              <div class="mt-4 flex items-baseline gap-4">
                <div>
                  <p class="text-xs text-slate-500">Current</p>
                  <p class="mt-1 text-2xl font-bold text-white">{profile.weight_kg ?? '—'} kg</p>
                </div>
                <div>
                  <p class="text-xs text-slate-500">Target</p>
                  <p class="mt-1 text-2xl font-bold text-white">{profile.target_weight_kg ?? '—'} kg</p>
                </div>
                {#if summary && summary.total_burned > 0}
                  <div>
                    <p class="text-xs text-slate-500">Burned</p>
                    <p class="mt-1 text-2xl font-bold text-orange-400">{summary.total_burned} kcal</p>
                  </div>
                {/if}
              </div>
              <form onsubmit={(e) => { e.preventDefault(); logWeight(); }} class="mt-4 flex items-center gap-3">
                <input
                  type="number"
                  step="0.1"
                  min="0"
                  bind:value={weightInput}
                  placeholder="kg"
                  disabled={weightLoading}
                  class="w-24 rounded-lg border border-slate-700 bg-surface px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500 disabled:opacity-50"
                />
                <button
                  type="submit"
                  disabled={weightLoading || !weightInput}
                  class="rounded-lg bg-joule-500 px-4 py-2 text-sm font-semibold text-slate-900 hover:bg-joule-400 transition disabled:opacity-50"
                >
                  {weightLoading ? '...' : 'Log'}
                </button>
                {#if weightSuccess}
                  <span class="text-xs text-green-400">Saved!</span>
                {/if}
              </form>
            </div>

            <WaterWidget totalMl={summary?.total_water_ml ?? 0} />
          </div>

          <div class="space-y-6">
            <WeightChart data={weightHistory} />

            <ExerciseWidget exercises={exercises} />

            <TipsWidget {tips} />

            <div class="rounded-xl border border-slate-800 bg-surface-light p-6">
              <h3 class="text-sm font-semibold text-white">Profile Summary</h3>
              <div class="mt-4 space-y-3">
                <div>
                  <p class="text-xs text-slate-500">Diet Plan</p>
                  <p class="mt-1 text-sm font-medium text-white">{goals.diet_plan || 'Not set'}</p>
                </div>
                <div>
                  <p class="text-xs text-slate-500">Objective</p>
                  <p class="mt-1 text-sm font-medium text-white">{goals.objective || 'Not set'}</p>
                </div>
                <div>
                  <p class="text-xs text-slate-500">Activity Level</p>
                  <p class="mt-1 text-sm font-medium text-white">{profile.activity_level || 'Not set'}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    {:else}
      <div class="flex min-h-[60vh] items-center justify-center">
        <div class="rounded-xl border border-slate-800 bg-surface-light p-8 text-center">
          <Logo size={48} />
          <h2 class="mt-4 text-lg font-semibold text-white">Welcome to Joule</h2>
          <p class="mt-2 text-sm text-slate-400">
            Your dashboard is being built. Complete onboarding to see your personalized nutrition tracking.
          </p>
          <div class="mt-6 flex justify-center gap-3">
            <a
              href="/onboarding"
              class="rounded-lg bg-joule-500 px-4 py-2 text-sm font-semibold text-slate-900 hover:bg-joule-400 transition"
            >
              Complete Setup
            </a>
          </div>
        </div>
      </div>
    {/if}
  </main>
</div>
