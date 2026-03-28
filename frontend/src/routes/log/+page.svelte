<script lang="ts">
  import Sidebar from '$components/Sidebar.svelte';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';

  interface FoodItem {
    name: string;
    calories: number;
    protein_g: number;
    carbs_g: number;
    fat_g: number;
    fiber_g: number;
    serving_size: string;
    source: 'ai' | 'manual';
  }

  interface YesterdayMeal {
    id: string;
    meal_type: string;
    timestamp: string;
    foods: { id: string; name: string; calories: number; protein_g: number; carbs_g: number; fat_g: number; fiber_g: number; serving_size?: string }[];
  }

  let mealType = $state<'breakfast' | 'lunch' | 'dinner' | 'snack'>('lunch');
  let note = $state('');
  let portionHint = $state('');
  let photoBase64 = $state<string | null>(null);
  let photoPreview = $state<string | null>(null);
  let foods = $state<FoodItem[]>([]);
  let loading = $state(false);
  let error = $state('');
  let showAddFood = $state(false);
  let authenticated = $state(false);

  // Leftovers modal
  let showLeftovers = $state(false);
  let leftoverMeals = $state<YesterdayMeal[]>([]);
  let leftoverLoading = $state(false);
  let selectedFoodIds = $state<Set<string>>(new Set());
  let removeFromYesterday = $state(true);
  let carryingOver = $state(false);

  let newFood = $state({
    name: '',
    calories: '',
    protein_g: '',
    carbs_g: '',
    fat_g: '',
    fiber_g: '',
    serving_size: ''
  });

  const mealTypes = ['breakfast', 'lunch', 'dinner', 'snack'] as const;

  let totalCalories = $derived(foods.reduce((sum, f) => sum + f.calories, 0));

  let canSubmit = $derived(
    (foods.length > 0 || photoBase64 !== null) && !loading
  );

  onMount(() => {
    const unsub = authToken.subscribe((token) => {
      if (!token) goto('/login');
      authenticated = !!token;
    });
    return unsub;
  });

  function handleFileSelect(file: File) {
    if (!file.type.startsWith('image/')) return;
    const reader = new FileReader();
    reader.onload = (e) => {
      const result = e.target?.result as string;
      photoBase64 = result;
      photoPreview = result;
    };
    reader.readAsDataURL(file);
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    const file = e.dataTransfer?.files[0];
    if (file) handleFileSelect(file);
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
  }

  function removePhoto() {
    photoBase64 = null;
    photoPreview = null;
  }

  function addFood() {
    if (!newFood.name.trim()) return;
    foods = [...foods, {
      name: newFood.name.trim(),
      calories: Number(newFood.calories) || 0,
      protein_g: Number(newFood.protein_g) || 0,
      carbs_g: Number(newFood.carbs_g) || 0,
      fat_g: Number(newFood.fat_g) || 0,
      fiber_g: Number(newFood.fiber_g) || 0,
      serving_size: newFood.serving_size.trim(),
      source: 'manual'
    }];
    newFood = { name: '', calories: '', protein_g: '', carbs_g: '', fat_g: '', fiber_g: '', serving_size: '' };
    showAddFood = false;
  }

  function removeFood(index: number) {
    foods = foods.filter((_, i) => i !== index);
  }

  async function openLeftovers() {
    showLeftovers = true;
    if (leftoverMeals.length > 0) return;
    leftoverLoading = true;
    try {
      const yesterday = new Date();
      yesterday.setDate(yesterday.getDate() - 1);
      const dateStr = yesterday.toISOString().split('T')[0];
      const summary = await api.get<{ meals: YesterdayMeal[] }>(`/dashboard/summary?date=${dateStr}`);
      leftoverMeals = (summary.meals ?? []).filter(m => m.foods.length > 0);
    } catch {}
    finally { leftoverLoading = false; }
  }

  function toggleFood(id: string) {
    const next = new Set(selectedFoodIds);
    if (next.has(id)) next.delete(id); else next.add(id);
    selectedFoodIds = next;
  }

  async function carryForward() {
    const allFoods = leftoverMeals.flatMap(m => m.foods);
    const chosen = allFoods.filter(f => selectedFoodIds.has(f.id));
    if (chosen.length === 0) return;
    carryingOver = true;
    try {
      await api.post('/meals/carry-forward', {
        meal_type: mealType,
        remove_from_yesterday: removeFromYesterday,
        foods: chosen.map(f => ({
          name: f.name,
          calories: f.calories,
          protein_g: f.protein_g,
          carbs_g: f.carbs_g,
          fat_g: f.fat_g,
          fiber_g: f.fiber_g,
          serving_size: f.serving_size ?? '',
          original_food_id: f.id,
        }))
      });
      showLeftovers = false;
      selectedFoodIds = new Set();
      goto('/dashboard');
    } catch {}
    finally { carryingOver = false; }
  }

  async function handleSubmit() {
    loading = true;
    error = '';
    try {
      await api.post('/meals', {
        meal_type: mealType,
        photo: photoBase64 || null,
        note: note || null,
        portion_hint: portionHint || null,
        foods: foods.map(f => ({
          name: f.name,
          calories: f.calories,
          protein_g: f.protein_g,
          carbs_g: f.carbs_g,
          fat_g: f.fat_g,
          fiber_g: f.fiber_g,
          serving_size: f.serving_size
        }))
      });
      goto('/dashboard');
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to log meal';
    } finally {
      loading = false;
    }
  }
</script>

{#if !authenticated}
  <div class="flex h-screen items-center justify-center bg-slate-950">
    <div class="h-8 w-8 animate-spin rounded-full border-2 border-joule-500 border-t-transparent"></div>
  </div>
{:else}
  <div class="flex min-h-screen">
    <Sidebar activePage="log" />

    <main class="flex-1 p-4 pb-20 lg:p-10 lg:pb-10">
      <div class="mb-8 flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-white">Log a Meal</h1>
          <p class="mt-1 text-sm text-slate-400">Add your meal to track nutrition</p>
        </div>
        <button
          onclick={() => { authToken.set(null); goto('/login'); }}
          class="rounded-lg border border-slate-700 px-3 py-1.5 text-sm text-slate-400 hover:text-white transition"
        >
          Sign out
        </button>
      </div>

      <div class="grid grid-cols-1 gap-6 xl:grid-cols-3">
        <div class="space-y-6 xl:col-span-2">
          <div class="rounded-xl border border-slate-800 bg-surface-light p-6">
            <h2 class="mb-4 text-sm font-semibold text-white">Photo (Optional)</h2>
            {#if photoPreview}
              <div class="relative">
                <img
                  src={photoPreview}
                  alt=""
                  class="mx-auto max-h-64 rounded-lg object-contain"
                />
                <button
                  type="button"
                  onclick={removePhoto}
                  class="absolute top-2 right-2 rounded-lg border border-slate-700 bg-surface-light px-3 py-1.5 text-sm text-slate-400 hover:text-white transition"
                >
                  Remove photo
                </button>
              </div>
              <div class="mt-4">
                <label for="portion-hint" class="mb-1.5 block text-xs font-medium text-slate-400">
                  Help the AI estimate portions
                  <span class="ml-1 text-slate-500">(optional but helps accuracy)</span>
                </label>
                <input
                  id="portion-hint"
                  type="text"
                  bind:value={portionHint}
                  placeholder='e.g. "double patty burger", "half a pizza", "large bowl"'
                  class="w-full rounded-lg border border-slate-700 bg-surface px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                />
              </div>
              <p class="mt-2 text-xs text-slate-500">AI will analyze your photo — you can review and edit items after logging.</p>
            {:else}
              <label
                class="flex cursor-pointer flex-col items-center gap-3 rounded-lg border-2 border-dashed border-slate-700 px-6 py-12 transition hover:border-slate-600"
                ondrop={handleDrop}
                ondragover={handleDragOver}
              >
                <svg class="h-10 w-10 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                <span class="text-sm text-slate-400">Take a photo or upload an image</span>
                <span class="text-xs text-slate-500">JPG, PNG, or WebP</span>
                <input
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  class="hidden"
                  onchange={(e) => {
                    const file = (e.target as HTMLInputElement).files?.[0];
                    if (file) handleFileSelect(file);
                  }}
                />
              </label>
            {/if}
          </div>

          <div class="rounded-xl border border-slate-800 bg-surface-light p-6">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-sm font-semibold text-white">Food Items</h2>
              <div class="flex gap-2">
                <button
                  type="button"
                  onclick={openLeftovers}
                  class="flex items-center gap-1.5 rounded-lg border border-slate-700 px-3 py-2 text-sm font-medium text-slate-400 hover:text-amber-400 hover:border-amber-500/40 hover:bg-amber-500/5 transition"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                  </svg>
                  Leftovers?
                </button>
                <button
                  type="button"
                  onclick={() => (showAddFood = !showAddFood)}
                  class="rounded-lg border border-slate-700 px-4 py-2 text-sm font-medium text-slate-400 hover:text-white hover:bg-slate-800 transition"
                >
                  + Add Food
                </button>
              </div>
            </div>

            {#if foods.length > 0}
              <div class="space-y-2">
                {#each foods as food, i}
                  <div class="flex items-center justify-between rounded-lg border border-slate-800 bg-surface px-4 py-3">
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-2">
                        <span class="truncate text-sm font-medium text-white">{food.name}</span>
                        {#if food.source === 'ai'}
                          <span class="rounded bg-joule-500/10 px-1.5 py-0.5 text-xs text-joule-400">AI</span>
                        {/if}
                      </div>
                      <div class="mt-0.5 text-xs text-slate-500">
                        {food.protein_g}g P · {food.carbs_g}g C · {food.fat_g}g F{#if food.fiber_g} · {food.fiber_g}g fiber{/if}
                        {#if food.serving_size}
                          · {food.serving_size}
                        {/if}
                      </div>
                    </div>
                    <div class="flex items-center gap-3">
                      <span class="text-sm font-medium text-white">{food.calories} kcal</span>
                      <button
                        type="button"
                        onclick={() => removeFood(i)}
                        aria-label="Remove food"
                        class="text-slate-500 hover:text-red-400 transition"
                      >
                        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                      </button>
                    </div>
                  </div>
                {/each}
              </div>
            {:else}
              <p class="py-4 text-center text-sm text-slate-500">No foods added yet. Add foods manually or upload a photo.</p>
            {/if}

            {#if showAddFood}
              <div class="mt-4 space-y-3 rounded-lg border border-slate-700 bg-surface p-4">
                <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
                  <div class="col-span-2 sm:col-span-4">
                    <input
                      type="text"
                      bind:value={newFood.name}
                      placeholder="Food name"
                      class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                    />
                  </div>
                  <input
                    type="number"
                    bind:value={newFood.calories}
                    placeholder="Calories"
                    min="0"
                    class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                  />
                  <input
                    type="number"
                    bind:value={newFood.protein_g}
                    placeholder="Protein (g)"
                    min="0"
                    class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                  />
                  <input
                    type="number"
                    bind:value={newFood.carbs_g}
                    placeholder="Carbs (g)"
                    min="0"
                    class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                  />
                  <input
                    type="number"
                    bind:value={newFood.fat_g}
                    placeholder="Fat (g)"
                    min="0"
                    class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                  />
                  <input
                    type="number"
                    bind:value={newFood.fiber_g}
                    placeholder="Fiber (g)"
                    min="0"
                    class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                  />
                  <input
                    type="text"
                    bind:value={newFood.serving_size}
                    placeholder="Serving (e.g. 1 cup)"
                    class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                  />
                </div>
                <div class="flex justify-end gap-2">
                  <button
                    type="button"
                    onclick={() => (showAddFood = false)}
                    class="rounded-lg border border-slate-700 px-4 py-2 text-sm font-medium text-slate-400 hover:text-white hover:bg-slate-800 transition"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    onclick={addFood}
                    disabled={!newFood.name.trim()}
                    class="rounded-lg bg-joule-500 px-4 py-2 text-sm font-semibold text-slate-900 hover:bg-joule-400 disabled:opacity-50 disabled:cursor-not-allowed transition"
                  >
                    Add
                  </button>
                </div>
              </div>
            {/if}
          </div>

          <div class="rounded-xl border border-slate-800 bg-surface-light p-6">
            <h2 class="mb-4 text-sm font-semibold text-white">Meal Details</h2>
            <div class="space-y-4">
              <div>
                <span class="mb-2 block text-sm font-medium text-slate-300">Meal Type</span>
                <div class="flex flex-wrap gap-2">
                  {#each mealTypes as type}
                    <button
                      type="button"
                      onclick={() => (mealType = type)}
                      class="rounded-lg border px-4 py-2 text-sm font-medium capitalize transition {mealType === type ? 'border-joule-500 bg-joule-500/10 text-joule-400' : 'border-slate-700 text-slate-400 hover:border-slate-600'}"
                    >
                      {type}
                    </button>
                  {/each}
                </div>
              </div>
              <div>
                <label for="note" class="mb-1.5 block text-sm font-medium text-slate-300">Note (Optional)</label>
                <input
                  id="note"
                  type="text"
                  bind:value={note}
                  placeholder="Add a note about this meal..."
                  class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                />
              </div>
            </div>
          </div>
        </div>

        <div class="space-y-6">
          <div class="rounded-xl border border-slate-800 bg-surface-light p-6">
            <h2 class="mb-4 text-sm font-semibold text-white">Summary</h2>
            {#if foods.length > 0}
              <div class="mb-4 rounded-lg border border-joule-500/20 bg-joule-500/5 p-4 text-center">
                <p class="text-3xl font-bold text-white">{totalCalories}</p>
                <p class="text-xs text-slate-400">total calories</p>
              </div>
              <div class="space-y-3">
                {#each foods as food}
                  <div class="flex items-center justify-between text-sm">
                    <span class="truncate text-slate-300">{food.name}</span>
                    <span class="ml-2 shrink-0 text-white">{food.calories}</span>
                  </div>
                {/each}
              </div>
            {:else}
              <p class="py-4 text-center text-sm text-slate-500">Add foods to see the summary</p>
            {/if}
          </div>

          {#if error}
            <div class="rounded-lg border border-red-500/20 bg-red-500/10 px-3.5 py-2.5 text-sm text-red-400">
              {error}
            </div>
          {/if}

          <button
            type="button"
            onclick={handleSubmit}
            disabled={!canSubmit}
            class="w-full rounded-lg bg-joule-500 px-4 py-3 text-sm font-semibold text-slate-900 hover:bg-joule-400 disabled:opacity-50 disabled:cursor-not-allowed transition"
          >
            {loading ? 'Logging...' : 'Log Meal'}
          </button>
        </div>
      </div>
    </main>
  </div>

  <!-- Leftovers Modal -->
  {#if showLeftovers}
    <div class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 bg-black/60 backdrop-blur-sm" onclick={(e) => { if (e.target === e.currentTarget) showLeftovers = false; }}>
      <div class="w-full max-w-lg rounded-2xl border border-slate-700 bg-surface shadow-2xl max-h-[80vh] flex flex-col">
        <!-- Header -->
        <div class="flex items-center justify-between border-b border-slate-800 px-5 py-4">
          <div>
            <h3 class="font-semibold text-white">Yesterday's Leftovers</h3>
            <p class="mt-0.5 text-xs text-slate-400">Select foods to carry forward to today</p>
          </div>
          <button onclick={() => (showLeftovers = false)} class="text-slate-500 hover:text-white transition">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Food list -->
        <div class="flex-1 overflow-y-auto px-5 py-3 space-y-4">
          {#if leftoverLoading}
            <div class="flex items-center justify-center py-10">
              <div class="h-6 w-6 animate-spin rounded-full border-2 border-joule-500 border-t-transparent"></div>
            </div>
          {:else if leftoverMeals.length === 0}
            <div class="py-10 text-center">
              <p class="text-sm text-slate-400">No meals logged yesterday.</p>
            </div>
          {:else}
            {#each leftoverMeals as meal}
              <div>
                <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-500 capitalize">{meal.meal_type}</p>
                <div class="space-y-1.5">
                  {#each meal.foods as food}
                    {@const selected = selectedFoodIds.has(food.id)}
                    <button
                      type="button"
                      onclick={() => toggleFood(food.id)}
                      class="w-full flex items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition {selected ? 'border-joule-500/40 bg-joule-500/10' : 'border-slate-800 hover:border-slate-700 bg-surface'}"
                    >
                      <div class="flex h-4 w-4 shrink-0 items-center justify-center rounded border {selected ? 'border-joule-500 bg-joule-500' : 'border-slate-600'}">
                        {#if selected}
                          <svg class="h-3 w-3 text-slate-900" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
                          </svg>
                        {/if}
                      </div>
                      <div class="min-w-0 flex-1">
                        <p class="truncate text-sm font-medium text-white">{food.name}</p>
                        <p class="text-xs text-slate-500">{food.protein_g}g P · {food.carbs_g}g C · {food.fat_g}g F{#if food.serving_size} · {food.serving_size}{/if}</p>
                      </div>
                      <span class="shrink-0 text-sm font-medium {selected ? 'text-joule-400' : 'text-slate-400'}">{food.calories} kcal</span>
                    </button>
                  {/each}
                </div>
              </div>
            {/each}
          {/if}
        </div>

        <!-- Footer -->
        {#if !leftoverLoading && leftoverMeals.length > 0}
          <div class="border-t border-slate-800 px-5 py-4 space-y-3">
            <label class="flex items-center gap-3 cursor-pointer">
              <div
                class="relative h-5 w-9 rounded-full transition {removeFromYesterday ? 'bg-joule-500' : 'bg-slate-700'}"
                onclick={() => (removeFromYesterday = !removeFromYesterday)}
              >
                <div class="absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white shadow transition-transform {removeFromYesterday ? 'translate-x-4' : ''}"></div>
              </div>
              <span class="text-sm text-slate-300">Remove from yesterday's log</span>
            </label>
            <div class="flex items-center justify-between gap-3">
              <p class="text-xs text-slate-500">
                {selectedFoodIds.size} item{selectedFoodIds.size !== 1 ? 's' : ''} selected
                {#if selectedFoodIds.size > 0}
                  · {[...leftoverMeals.flatMap(m => m.foods)].filter(f => selectedFoodIds.has(f.id)).reduce((s, f) => s + f.calories, 0)} kcal
                {/if}
              </p>
              <button
                type="button"
                onclick={carryForward}
                disabled={selectedFoodIds.size === 0 || carryingOver}
                class="rounded-lg bg-joule-500 px-5 py-2 text-sm font-semibold text-slate-900 hover:bg-joule-400 disabled:opacity-50 disabled:cursor-not-allowed transition"
              >
                {carryingOver ? 'Adding...' : 'Carry Forward'}
              </button>
            </div>
          </div>
        {/if}
      </div>
    </div>
  {/if}
{/if}
