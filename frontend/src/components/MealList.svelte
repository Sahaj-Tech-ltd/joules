<script lang="ts">
  let { meals = [] as Array<{
    id: string;
    timestamp: string;
    meal_type: string;
    note?: string;
    photo_path?: string;
    foods: Array<{
      id: string;
      name: string;
      calories: number;
      source: string;
    }>;
  }> }: { meals: Array<{ id: string; timestamp: string; meal_type: string; note?: string; photo_path?: string; foods: Array<{ id: string; name: string; calories: number; source: string }> }> } = $props();

  let mealTypeIcons: Record<string, string> = {
    breakfast: 'M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707',
    lunch: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z',
    dinner: 'M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z',
    snack: 'M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z'
  };

  function formatTime(ts: string): string {
    return new Date(ts).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
  }

  function mealTotalCalories(meal: typeof meals[0]): number {
    return meal.foods.reduce((sum, f) => sum + f.calories, 0);
  }
</script>

<div class="rounded-xl border border-slate-800 bg-surface-light p-6">
  <h3 class="text-sm font-semibold text-slate-100 mb-4">Today's Meals</h3>
  {#if meals.length === 0}
    <div class="flex flex-col items-center justify-center py-8 gap-3">
      <p class="text-sm text-slate-400">No meals logged today</p>
      <button class="text-xs text-joule-500 hover:text-joule-400 font-medium transition-colors">
        Log a meal
      </button>
    </div>
  {:else}
    <div class="flex flex-col gap-3">
      {#each meals as meal}
        <div class="rounded-lg border border-slate-800 bg-surface p-4">
          <div class="flex items-start justify-between gap-3">
            <div class="flex items-center gap-2 min-w-0">
              {#if meal.photo_path}
                <img
                  src={meal.photo_path}
                  alt=""
                  class="w-10 h-10 rounded-lg object-cover flex-shrink-0"
                />
              {:else}
                <div class="w-10 h-10 rounded-lg bg-surface-lighter flex items-center justify-center flex-shrink-0">
                  <svg class="w-5 h-5 text-joule-500" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" d={mealTypeIcons[meal.meal_type] || mealTypeIcons.snack} />
                  </svg>
                </div>
              {/if}
              <div class="min-w-0">
                <p class="text-sm font-medium text-slate-100 capitalize">{meal.meal_type}</p>
                <p class="text-xs text-slate-400">{formatTime(meal.timestamp)}</p>
              </div>
            </div>
            <span class="text-sm font-semibold text-joule-500 flex-shrink-0">{mealTotalCalories(meal)} kcal</span>
          </div>

          {#if meal.foods.length > 0}
            <div class="mt-3 ml-12 flex flex-col gap-2">
              {#each meal.foods as food}
                <div class="flex items-center justify-between gap-2">
                  <div class="flex items-center gap-2 min-w-0">
                    <span class="text-sm text-slate-200 truncate">{food.name}</span>
                    <span class="text-[10px] px-1.5 py-0.5 rounded-full font-medium {food.source === 'ai'
                      ? 'bg-joule-500/20 text-joule-400'
                      : 'bg-slate-600/50 text-slate-400'
                    }">{food.source === 'ai' ? 'AI' : 'Manual'}</span>
                  </div>
                  <span class="text-xs text-slate-400 flex-shrink-0">{food.calories} kcal</span>
                </div>
              {/each}
            </div>
          {/if}

          {#if meal.note}
            <p class="mt-2 ml-12 text-xs text-slate-400 italic">{meal.note}</p>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
