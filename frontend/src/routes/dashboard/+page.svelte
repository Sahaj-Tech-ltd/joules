<script lang="ts">
  import Logo from '$components/Logo.svelte';
  import Sidebar from '$components/Sidebar.svelte';
  import MacroRing from '$components/MacroRing.svelte';
  import BannerStrip from '$components/BannerStrip.svelte';
  import WeightChart from '$components/WeightChart.svelte';
  import MealList from '$components/MealList.svelte';
  import WaterWidget from '$components/WaterWidget.svelte';
  import ExerciseWidget from '$components/ExerciseWidget.svelte';
  import TipsWidget from '$components/TipsWidget.svelte';
  import ThemeToggle from '$components/ThemeToggle.svelte';
  import { authToken, userProfile, userGoals } from '$lib/stores';
  import { api } from '$lib/api';
  import { showAchievement } from '$lib/achievements';
  import { defaultUnits, displayEnergy, type UnitPrefs } from '$lib/units';
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
  let unitPrefs = $state<UnitPrefs>({ ...defaultUnits });
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

    async function checkAchievements() {
      try {
        const seenRaw = localStorage.getItem('seen_achievement_ids');
        const seen: string[] = seenRaw ? JSON.parse(seenRaw) : [];
        const seenSet = new Set<string>(seen);

        const unlocked = await api.post<{ id: string; title: string; description: string }[]>('/achievements/check', {});
        if (Array.isArray(unlocked)) {
          const newIds: string[] = [];
          for (const a of unlocked) {
            if (!seenSet.has(a.id)) {
              showAchievement({ id: a.id, title: a.title, description: a.description });
              newIds.push(a.id);
            }
          }
          if (newIds.length > 0) {
            localStorage.setItem('seen_achievement_ids', JSON.stringify([...seen, ...newIds]));
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
        checkAchievements();
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
  <Sidebar activePage="dashboard" isAdmin={profile?.is_admin ?? false} />

  <main class="flex-1 overflow-y-auto p-4 pb-20 lg:p-10 lg:pb-10">
    {#if loading}
      <div class="flex h-64 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-slate-700 border-t-joule-500"></div>
      </div>
    {:else if profile && profile.onboarding_complete && goals}
      <div class="space-y-8">
        <BannerStrip />
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-4">
            <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-joule-400 to-joule-600 text-sm font-bold text-slate-900 shadow-lg shadow-joule-500/25 select-none">
              {profile.name.split(' ').map((n: string) => n[0]).filter(Boolean).join('').slice(0, 2).toUpperCase()}
            </div>
            <div>
              <h1 class="text-xl font-bold text-white leading-tight">{greeting}, {profile.name.split(' ')[0]}</h1>
              <p class="text-sm text-slate-500">{todayFormatted}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <ThemeToggle />
            <button
              onclick={() => { authToken.set(null); goto('/login'); }}
              class="rounded-lg border border-slate-700/80 px-3 py-1.5 text-sm text-slate-400 hover:text-white hover:border-slate-600 transition"
            >
              Sign out
            </button>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
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

        <div class="flex flex-wrap gap-3">
          <a
            href="/log"
            class="inline-flex items-center gap-2 rounded-xl bg-joule-500 px-5 py-2.5 text-sm font-semibold text-slate-900 shadow-lg shadow-joule-500/20 hover:bg-joule-400 transition-all"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" /></svg>
            Log a Meal
          </a>
          <a
            href="/coach"
            class="inline-flex items-center gap-2 rounded-xl border border-slate-700/80 bg-surface-light px-5 py-2.5 text-sm font-semibold text-slate-300 hover:bg-slate-800 hover:text-white hover:border-slate-600 transition-all"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" /></svg>
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
          <h2 class="mt-4 text-lg font-semibold text-white">Welcome to Joules</h2>
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
