<script lang="ts">
  import { page } from '$app/stores';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';

  interface FoodViewItem {
    name: string;
    calories: number;
    protein_g: number;
    carbs_g: number;
    fat_g: number;
  }

  interface MealViewItem {
    id: string;
    timestamp: string;
    meal_type: string;
    note?: string;
    foods: FoodViewItem[];
  }

  interface UserViewProfile {
    name: string;
    age?: number;
    sex?: string;
    height_cm?: number;
    weight_kg?: number;
    target_weight_kg?: number;
    activity_level?: string;
  }

  interface UserViewGoals {
    objective: string;
    diet_plan: string;
    fasting_window?: string;
    daily_calorie_target: number;
    daily_protein_g: number;
    daily_carbs_g: number;
    daily_fat_g: number;
  }

  interface UserViewSummary {
    total_calories: number;
    total_protein: number;
    total_carbs: number;
    total_fat: number;
    total_fiber: number;
    total_burned: number;
    total_water_ml: number;
  }

  interface WeightViewEntry {
    date: string;
    weight_kg: number;
  }

  interface UserView {
    email: string;
    created_at: string;
    date: string;
    profile?: UserViewProfile;
    goals?: UserViewGoals;
    summary: UserViewSummary;
    meals: MealViewItem[];
    weight_history: WeightViewEntry[];
  }

  let userID = $derived($page.params.id);
  let data = $state<UserView | null>(null);
  let loading = $state(true);
  let dateStr = $state(new Date().toISOString().split('T')[0]);

  async function loadView() {
    loading = true;
    try {
      data = await api.get<UserView>(`/admin/users/${userID}/view?date=${dateStr}`);
    } catch {
      goto('/admin');
    } finally {
      loading = false;
    }
  }

  // Reload when userID or date changes (handles SvelteKit navigation between users)
  $effect(() => {
    void userID; // track dependency
    void dateStr;
    loadView();
  });

  onMount(() => {
    const unsub = authToken.subscribe((token) => {
      if (!token) { goto('/login'); return; }
    });
    return unsub;
  });

  function pct(val: number, target: number) {
    if (!target) return 0;
    return Math.min(100, Math.round((val / target) * 100));
  }

  function fmtMealType(t: string) {
    return t.charAt(0).toUpperCase() + t.slice(1);
  }

  function fmtTime(iso: string) {
    return new Date(iso).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
  }
</script>

<div class="min-h-screen bg-slate-950 text-white">
  <!-- Header -->
  <div class="border-b border-slate-800 bg-slate-900/50 px-6 py-4">
    <div class="flex items-center gap-4">
      <a href="/admin" aria-label="Back to admin" class="text-slate-400 hover:text-white transition">
        <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </a>
      <div>
        <div class="flex items-center gap-2">
          <span class="inline-flex items-center rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-400">
            God's Eye
          </span>
          <h1 class="text-lg font-bold">{data?.email ?? '...'}</h1>
        </div>
        <p class="text-xs text-slate-500 mt-0.5">Read-only admin view</p>
      </div>
    </div>
  </div>

  <div class="mx-auto max-w-5xl px-6 py-8">
    {#if loading}
      <div class="flex h-64 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-slate-700 border-t-joule-500"></div>
      </div>
    {:else if data}
      <!-- Date Picker + Nav -->
      <div class="mb-6 flex items-center gap-3">
        <button
          aria-label="Previous day"
          onclick={() => {
            const d = new Date(dateStr + 'T12:00:00');
            d.setDate(d.getDate() - 1);
            dateStr = d.toISOString().split('T')[0];
            loadView();
          }}
          class="rounded-lg border border-slate-700 p-2 text-slate-400 hover:text-white hover:bg-slate-800 transition"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <input
          type="date"
          bind:value={dateStr}
          onchange={loadView}
          class="rounded-lg border border-slate-700 bg-surface px-3 py-2 text-sm text-white focus:border-joule-500 focus:outline-none"
        />
        <button
          aria-label="Next day"
          onclick={() => {
            const d = new Date(dateStr + 'T12:00:00');
            d.setDate(d.getDate() + 1);
            dateStr = d.toISOString().split('T')[0];
            loadView();
          }}
          class="rounded-lg border border-slate-700 p-2 text-slate-400 hover:text-white hover:bg-slate-800 transition"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </button>
        <span class="text-sm text-slate-400">{data.date}</span>
      </div>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <!-- Left column: profile + goals -->
        <div class="space-y-4">
          <!-- Profile -->
          <div class="rounded-xl border border-slate-700 bg-slate-900 p-4">
            <h3 class="mb-3 text-xs font-semibold uppercase tracking-wider text-slate-500">Profile</h3>
            {#if data.profile}
              <dl class="space-y-2 text-sm">
                {#if data.profile.name}
                  <div class="flex justify-between">
                    <dt class="text-slate-400">Name</dt>
                    <dd class="text-white">{data.profile.name}</dd>
                  </div>
                {/if}
                {#if data.profile.age}
                  <div class="flex justify-between">
                    <dt class="text-slate-400">Age</dt>
                    <dd class="text-white">{data.profile.age}</dd>
                  </div>
                {/if}
                {#if data.profile.sex}
                  <div class="flex justify-between">
                    <dt class="text-slate-400">Sex</dt>
                    <dd class="text-white capitalize">{data.profile.sex}</dd>
                  </div>
                {/if}
                {#if data.profile.height_cm}
                  <div class="flex justify-between">
                    <dt class="text-slate-400">Height</dt>
                    <dd class="text-white">{data.profile.height_cm} cm</dd>
                  </div>
                {/if}
                {#if data.profile.weight_kg}
                  <div class="flex justify-between">
                    <dt class="text-slate-400">Weight</dt>
                    <dd class="text-white">{data.profile.weight_kg} kg</dd>
                  </div>
                {/if}
                {#if data.profile.target_weight_kg}
                  <div class="flex justify-between">
                    <dt class="text-slate-400">Target</dt>
                    <dd class="text-white">{data.profile.target_weight_kg} kg</dd>
                  </div>
                {/if}
                {#if data.profile.activity_level}
                  <div class="flex justify-between">
                    <dt class="text-slate-400">Activity</dt>
                    <dd class="text-white capitalize">{data.profile.activity_level.replace('_', ' ')}</dd>
                  </div>
                {/if}
              </dl>
            {:else}
              <p class="text-sm text-slate-500">No profile yet</p>
            {/if}
          </div>

          <!-- Goals -->
          <div class="rounded-xl border border-slate-700 bg-slate-900 p-4">
            <h3 class="mb-3 text-xs font-semibold uppercase tracking-wider text-slate-500">Goals</h3>
            {#if data.goals}
              <dl class="space-y-2 text-sm">
                <div class="flex justify-between">
                  <dt class="text-slate-400">Plan</dt>
                  <dd class="text-white capitalize">{data.goals.diet_plan.replace(/_/g, ' ')}</dd>
                </div>
                <div class="flex justify-between">
                  <dt class="text-slate-400">Objective</dt>
                  <dd class="text-white capitalize">{data.goals.objective.replace(/_/g, ' ')}</dd>
                </div>
                {#if data.goals.fasting_window}
                  <div class="flex justify-between">
                    <dt class="text-slate-400">Fasting</dt>
                    <dd class="text-white">{data.goals.fasting_window}</dd>
                  </div>
                {/if}
                <div class="mt-2 pt-2 border-t border-slate-800 space-y-1">
                  <div class="flex justify-between text-xs">
                    <span class="text-slate-500">Calories</span>
                    <span class="text-white">{data.goals.daily_calorie_target} kcal</span>
                  </div>
                  <div class="flex justify-between text-xs">
                    <span class="text-slate-500">Protein</span>
                    <span class="text-white">{data.goals.daily_protein_g}g</span>
                  </div>
                  <div class="flex justify-between text-xs">
                    <span class="text-slate-500">Carbs</span>
                    <span class="text-white">{data.goals.daily_carbs_g}g</span>
                  </div>
                  <div class="flex justify-between text-xs">
                    <span class="text-slate-500">Fat</span>
                    <span class="text-white">{data.goals.daily_fat_g}g</span>
                  </div>
                </div>
              </dl>
            {:else}
              <p class="text-sm text-slate-500">No goals set</p>
            {/if}
          </div>

          <!-- Weight history -->
          {#if data.weight_history.length > 0}
            <div class="rounded-xl border border-slate-700 bg-slate-900 p-4">
              <h3 class="mb-3 text-xs font-semibold uppercase tracking-wider text-slate-500">Weight (last 30 days)</h3>
              <div class="space-y-1.5 max-h-48 overflow-y-auto">
                {#each data.weight_history as entry}
                  <div class="flex justify-between text-sm">
                    <span class="text-slate-400">{entry.date}</span>
                    <span class="text-white font-medium">{entry.weight_kg} kg</span>
                  </div>
                {/each}
              </div>
            </div>
          {/if}
        </div>

        <!-- Right column: daily summary + meals -->
        <div class="lg:col-span-2 space-y-4">
          <!-- Daily summary -->
          <div class="rounded-xl border border-slate-700 bg-slate-900 p-4">
            <h3 class="mb-4 text-xs font-semibold uppercase tracking-wider text-slate-500">Daily Summary</h3>
            <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {#each [
                { label: 'Calories', val: data.summary.total_calories, target: data.goals?.daily_calorie_target, unit: 'kcal' },
                { label: 'Protein', val: Math.round(data.summary.total_protein), target: data.goals?.daily_protein_g, unit: 'g' },
                { label: 'Carbs', val: Math.round(data.summary.total_carbs), target: data.goals?.daily_carbs_g, unit: 'g' },
                { label: 'Fat', val: Math.round(data.summary.total_fat), target: data.goals?.daily_fat_g, unit: 'g' },
              ] as m}
                <div class="rounded-lg bg-slate-800/60 p-3">
                  <p class="text-xs text-slate-500 mb-1">{m.label}</p>
                  <p class="text-lg font-bold text-white">{m.val}<span class="text-xs font-normal text-slate-400 ml-0.5">{m.unit}</span></p>
                  {#if m.target}
                    <div class="mt-1.5 h-1.5 rounded-full bg-slate-700 overflow-hidden">
                      <div
                        class="h-full rounded-full bg-joule-500 transition-all"
                        style="width:{pct(m.val, m.target)}%"
                      ></div>
                    </div>
                    <p class="mt-0.5 text-xs text-slate-600">/ {m.target}{m.unit}</p>
                  {/if}
                </div>
              {/each}
            </div>
            <div class="mt-3 flex gap-4 text-sm">
              <span class="text-slate-400">Water: <span class="text-white font-medium">{data.summary.total_water_ml} ml</span></span>
              <span class="text-slate-400">Burned: <span class="text-white font-medium">{data.summary.total_burned} kcal</span></span>
            </div>
          </div>

          <!-- Meals -->
          <div class="rounded-xl border border-slate-700 bg-slate-900 p-4">
            <h3 class="mb-3 text-xs font-semibold uppercase tracking-wider text-slate-500">
              Meals ({data.meals.length})
            </h3>
            {#if data.meals.length === 0}
              <p class="text-sm text-slate-500">No meals logged on this date.</p>
            {:else}
              <div class="space-y-3">
                {#each data.meals as meal}
                  <div class="rounded-lg border border-slate-800 bg-slate-800/40 p-3">
                    <div class="flex items-center justify-between mb-2">
                      <span class="text-sm font-medium text-white">{fmtMealType(meal.meal_type)}</span>
                      <span class="text-xs text-slate-500">{fmtTime(meal.timestamp)}</span>
                    </div>
                    {#if meal.note}
                      <p class="text-xs text-slate-400 mb-2 italic">{meal.note}</p>
                    {/if}
                    {#if meal.foods.length > 0}
                      <div class="space-y-1">
                        {#each meal.foods as food}
                          <div class="flex items-center justify-between text-xs">
                            <span class="text-slate-300">{food.name}</span>
                            <span class="text-slate-500">
                              {food.calories} kcal
                              <span class="text-slate-600 ml-1">P:{Math.round(food.protein_g)}g C:{Math.round(food.carbs_g)}g F:{Math.round(food.fat_g)}g</span>
                            </span>
                          </div>
                        {/each}
                      </div>
                    {:else}
                      <p class="text-xs text-slate-600">No food items</p>
                    {/if}
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>
