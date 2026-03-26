<script lang="ts">
  import Logo from '$components/Logo.svelte';
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

  let mealType = $state<'breakfast' | 'lunch' | 'dinner' | 'snack'>('lunch');
  let note = $state('');
  let photoBase64 = $state<string | null>(null);
  let photoPreview = $state<string | null>(null);
  let foods = $state<FoodItem[]>([]);
  let loading = $state(false);
  let error = $state('');
  let showAddFood = $state(false);
  let authenticated = $state(false);

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
    foods.length > 0 && !loading
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

  async function handleSubmit() {
    loading = true;
    error = '';
    try {
      await api.post('/meals', {
        meal_type: mealType,
        photo: photoBase64 || null,
        note: note || null,
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
    <aside class="hidden w-64 border-r border-slate-800 bg-surface p-6 lg:block">
      <div class="flex items-center gap-3">
        <Logo size={32} />
        <span class="text-lg font-bold text-white">Joule</span>
      </div>
      <nav class="mt-8 space-y-1">
        <a
          href="/dashboard"
          class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" /></svg>
          Dashboard
        </a>
        <a
          href="/log"
          class="flex items-center gap-3 rounded-lg bg-slate-800 px-3 py-2 text-sm font-medium text-white"
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
      </nav>
    </aside>

    <main class="flex-1 p-6 lg:p-10">
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
              <button
                type="button"
                onclick={() => (showAddFood = !showAddFood)}
                class="rounded-lg border border-slate-700 px-4 py-2 text-sm font-medium text-slate-400 hover:text-white hover:bg-slate-800 transition"
              >
                + Add Food
              </button>
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
{/if}
