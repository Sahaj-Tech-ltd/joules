<script lang="ts">
  import { api } from '$lib/api';

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
      protein_g: number;
      carbs_g: number;
      fat_g: number;
      fiber_g: number;
      serving_size?: string;
      source: string;
    }>;
  }>, onload }: { meals: Array<{ id: string; timestamp: string; meal_type: string; note?: string; photo_path?: string; foods: Array<{ id: string; name: string; calories: number; protein_g: number; carbs_g: number; fat_g: number; fiber_g: number; serving_size?: string; source: string }> }>, onload?: () => void } = $props();

  let editingFoodId = $state<string | null>(null);
  let editForm = $state({ name: '', calories: '', protein_g: '', carbs_g: '', fat_g: '', fiber_g: '', serving_size: '' });
  let deletingMealId = $state<string | null>(null);

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

  function startEdit(mealId: string, food: typeof meals[0]['foods'][0]) {
    editingFoodId = food.id;
    editForm = {
      name: food.name,
      calories: String(food.calories),
      protein_g: String(food.protein_g),
      carbs_g: String(food.carbs_g),
      fat_g: String(food.fat_g),
      fiber_g: String(food.fiber_g),
      serving_size: food.serving_size ?? ''
    };
  }

  function cancelEdit() {
    editingFoodId = null;
  }

  async function saveEdit(mealId: string, foodId: string) {
    try {
      await api.put(`/meals/${mealId}/foods/${foodId}`, {
        name: editForm.name,
        calories: Number(editForm.calories) || 0,
        protein_g: Number(editForm.protein_g) || 0,
        carbs_g: Number(editForm.carbs_g) || 0,
        fat_g: Number(editForm.fat_g) || 0,
        fiber_g: Number(editForm.fiber_g) || 0,
        serving_size: editForm.serving_size
      });
      editingFoodId = null;
      onload?.();
    } catch {}
  }

  async function deleteFood(mealId: string, foodId: string) {
    try {
      await api.del(`/meals/${mealId}/foods/${foodId}`);
      onload?.();
    } catch {}
  }

  async function deleteMeal(mealId: string) {
    try {
      await api.del(`/meals/${mealId}`);
      deletingMealId = null;
      onload?.();
    } catch {}
  }
</script>

<div class="rounded-2xl border border-border bg-card p-5">
  <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Today's Meals</p>
  {#if meals.length === 0}
    <div class="flex flex-col items-center justify-center py-6 gap-2">
      <p class="text-sm text-muted-foreground">No meals logged yet</p>
    </div>
  {:else}
    <div class="flex flex-col gap-2.5">
      {#each meals as meal}
        <div class="rounded-xl border border-border bg-card/60 p-3.5">
          <div class="flex items-start justify-between gap-3">
            <div class="flex items-center gap-3 min-w-0">
              {#if meal.photo_path}
                <img src={meal.photo_path} alt="" class="w-10 h-10 rounded-xl object-cover flex-shrink-0" />
              {:else}
                <div class="w-10 h-10 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center flex-shrink-0">
                  <svg class="w-4.5 h-4.5 text-primary" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" d={mealTypeIcons[meal.meal_type] || mealTypeIcons.snack} />
                  </svg>
                </div>
              {/if}
              <div class="min-w-0">
                <p class="text-sm font-semibold text-foreground capitalize">{meal.meal_type}</p>
                <p class="text-[11px] text-muted-foreground mt-0.5">{formatTime(meal.timestamp)}</p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span class="font-display text-sm font-bold text-primary flex-shrink-0">{mealTotalCalories(meal)} <span class="text-[10px] font-normal text-muted-foreground">kcal</span></span>
              <div class="relative">
                <button
                  onclick={() => deletingMealId = deletingMealId === meal.id ? null : meal.id}
                  class="p-1 rounded-lg text-muted-foreground hover:text-red-400 hover:bg-red-500/10 transition"
                  aria-label="Meal options"
                >
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 6.75a.75.75 0 110-1.5.75.75 0 010 1.5zM12 12.75a.75.75 0 110-1.5.75.75 0 010 1.5zM12 18.75a.75.75 0 110-1.5.75.75 0 010 1.5z" />
                  </svg>
                </button>
                {#if deletingMealId === meal.id}
                  <div class="absolute right-0 top-8 z-30 rounded-xl border border-border bg-secondary shadow-xl py-1 min-w-[120px]">
                    <button
                      onclick={() => deleteMeal(meal.id)}
                      class="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-400 hover:bg-red-500/10 transition"
                    >
                      <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
                      </svg>
                      Delete Meal
                    </button>
                  </div>
                {/if}
              </div>
            </div>
          </div>

          {#if meal.foods.length > 0}
            <div class="mt-2.5 ml-13 flex flex-col gap-1.5 pl-[52px]">
              {#each meal.foods as food}
                {#if editingFoodId === food.id}
                  <div class="rounded-lg border border-primary/30 bg-primary/5 p-2.5 space-y-2">
                    <input type="text" bind:value={editForm.name} placeholder="Food name"
                      class="w-full rounded-lg border border-border bg-card px-2.5 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary/30" />
                    <div class="grid grid-cols-4 gap-1.5">
                      <input type="number" bind:value={editForm.calories} placeholder="Cal" min="0"
                        class="rounded-lg border border-border bg-card px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary/30" />
                      <input type="number" bind:value={editForm.protein_g} placeholder="P(g)" min="0"
                        class="rounded-lg border border-border bg-card px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary/30" />
                      <input type="number" bind:value={editForm.carbs_g} placeholder="C(g)" min="0"
                        class="rounded-lg border border-border bg-card px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary/30" />
                      <input type="number" bind:value={editForm.fat_g} placeholder="F(g)" min="0"
                        class="rounded-lg border border-border bg-card px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary/30" />
                    </div>
                    <div class="flex justify-end gap-1.5">
                      <button onclick={cancelEdit} class="px-2.5 py-1 text-[11px] text-muted-foreground hover:text-foreground transition">Cancel</button>
                      <button onclick={() => saveEdit(meal.id, food.id)} class="px-2.5 py-1 text-[11px] font-medium text-primary bg-primary/10 rounded-md hover:bg-primary/20 transition">Save</button>
                    </div>
                  </div>
                {:else}
                  <div class="flex items-center justify-between gap-2 group">
                    <div class="flex items-center gap-1.5 min-w-0">
                      <span class="text-xs text-foreground truncate">{food.name}</span>
                      <span class="text-[9px] px-1.5 py-0.5 rounded-full font-medium shrink-0 {food.source === 'ai'
                        ? 'bg-primary/15 text-primary'
                        : 'bg-accent/50 text-muted-foreground'
                      }">{food.source === 'ai' ? 'AI' : food.source === 'db' ? 'DB' : food.source === 'barcode' ? 'Scan' : 'Manual'}</span>
                    </div>
                    <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                      <button onclick={() => startEdit(meal.id, food)} class="p-0.5 text-muted-foreground hover:text-primary transition" aria-label="Edit food">
                        <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931z" />
                        </svg>
                      </button>
                      <button onclick={() => deleteFood(meal.id, food.id)} class="p-0.5 text-muted-foreground hover:text-red-400 transition" aria-label="Delete food">
                        <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                      </button>
                    </div>
                    <span class="text-[11px] text-muted-foreground flex-shrink-0">{food.calories} kcal</span>
                  </div>
                {/if}
              {/each}
            </div>
          {/if}

          {#if meal.note}
            <p class="mt-2 pl-[52px] text-[11px] text-muted-foreground italic">{meal.note}</p>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
