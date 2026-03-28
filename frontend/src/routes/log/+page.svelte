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
    source: 'ai' | 'manual' | 'db' | 'barcode' | 'recipe' | 'leftover';
  }

  interface FoodResult {
    id?: number;
    barcode?: string;
    name: string;
    brand?: string;
    calories: number;
    protein_g: number;
    carbs_g: number;
    fat_g: number;
    fiber_g: number;
    serving_size: string;
    ingredients?: string;
    source: 'local' | 'openfoodfacts';
  }

  interface RecipeFood {
    name: string;
    calories: number;
    protein_g: number;
    carbs_g: number;
    fat_g: number;
    fiber_g: number;
    serving_size: string;
  }

  interface Recipe {
    id: string;
    name: string;
    description: string;
    foods: RecipeFood[];
    created_at: string;
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

  // Food search
  let searchQuery = $state('');
  let searchResults = $state<FoodResult[]>([]);
  let searchLoading = $state(false);
  let showSearchDropdown = $state(false);
  let searchDebounceTimer = $state<ReturnType<typeof setTimeout> | null>(null);

  // Barcode scan modal
  let showBarcodeModal = $state(false);
  let barcodeSupported = $state(false);
  let barcodeLoading = $state(false);
  let barcodeError = $state('');
  let barcodeSuccess = $state('');
  let videoElement = $state<HTMLVideoElement | null>(null);
  let cameraStream = $state<MediaStream | null>(null);
  let scanInterval = $state<ReturnType<typeof setInterval> | null>(null);

  // Recipes
  let recipes = $state<Recipe[]>([]);
  let recipesLoading = $state(false);
  let showSaveRecipeModal = $state(false);
  let recipeName = $state('');
  let recipeDescription = $state('');
  let recipeSaving = $state(false);

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

    // Check BarcodeDetector support
    barcodeSupported = 'BarcodeDetector' in window;

    // Load recipes
    loadRecipes();

    return unsub;
  });

  async function loadRecipes() {
    recipesLoading = true;
    try {
      recipes = await api.get<Recipe[]>('/recipes');
    } catch {
      recipes = [];
    } finally {
      recipesLoading = false;
    }
  }

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

  function addFoodFromResult(result: FoodResult) {
    foods = [...foods, {
      name: result.brand ? `${result.name} (${result.brand})` : result.name,
      calories: result.calories,
      protein_g: result.protein_g,
      carbs_g: result.carbs_g,
      fat_g: result.fat_g,
      fiber_g: result.fiber_g,
      serving_size: result.serving_size,
      source: 'db'
    }];
    searchQuery = '';
    searchResults = [];
    showSearchDropdown = false;
  }

  function removeFood(index: number) {
    foods = foods.filter((_, i) => i !== index);
  }

  function handleSearchInput() {
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
    if (!searchQuery.trim()) {
      searchResults = [];
      showSearchDropdown = false;
      return;
    }
    searchDebounceTimer = setTimeout(async () => {
      searchLoading = true;
      showSearchDropdown = true;
      try {
        const results = await api.get<FoodResult[]>(`/foods/search?q=${encodeURIComponent(searchQuery)}&limit=10`);
        searchResults = results ?? [];
      } catch {
        searchResults = [];
      } finally {
        searchLoading = false;
      }
    }, 300);
  }

  function closeSearchDropdown() {
    showSearchDropdown = false;
  }

  // Barcode scanning
  async function openBarcodeModal() {
    showBarcodeModal = true;
    barcodeError = '';
    barcodeSuccess = '';
    if (barcodeSupported) {
      // Start camera after DOM settles
      setTimeout(() => startCamera(), 100);
    }
  }

  function closeBarcodeModal() {
    stopCamera();
    showBarcodeModal = false;
    barcodeError = '';
    barcodeSuccess = '';
    barcodeLoading = false;
  }

  async function startCamera() {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' } });
      cameraStream = stream;
      if (videoElement) {
        videoElement.srcObject = stream;
        await videoElement.play();
        startScanLoop();
      }
    } catch {
      barcodeError = 'Could not access camera. Please allow camera permission or use file upload.';
    }
  }

  function stopCamera() {
    if (scanInterval) {
      clearInterval(scanInterval);
      scanInterval = null;
    }
    if (cameraStream) {
      cameraStream.getTracks().forEach(t => t.stop());
      cameraStream = null;
    }
  }

  function startScanLoop() {
    scanInterval = setInterval(async () => {
      if (!videoElement || !barcodeSupported || barcodeLoading) return;
      try {
        // @ts-ignore BarcodeDetector is not in TS lib yet
        const detector = new BarcodeDetector({ formats: ['ean_13', 'ean_8', 'upc_a', 'upc_e', 'code_128'] });
        const barcodes = await detector.detect(videoElement);
        if (barcodes.length > 0) {
          const upc = barcodes[0].rawValue;
          await lookupBarcode(upc);
        }
      } catch {}
    }, 500);
  }

  async function lookupBarcode(upc: string) {
    if (barcodeLoading) return;
    stopCamera();
    barcodeLoading = true;
    barcodeError = '';
    try {
      const result = await api.get<FoodResult>(`/foods/barcode/${upc}`);
      foods = [...foods, {
        name: result.brand ? `${result.name} (${result.brand})` : result.name,
        calories: result.calories,
        protein_g: result.protein_g,
        carbs_g: result.carbs_g,
        fat_g: result.fat_g,
        fiber_g: result.fiber_g,
        serving_size: result.serving_size,
        source: 'barcode'
      }];
      barcodeSuccess = `Added: ${result.name}`;
      setTimeout(() => closeBarcodeModal(), 1500);
    } catch (err) {
      if (err instanceof Error && err.message.includes('404')) {
        barcodeError = 'Product not found in database.';
      } else {
        barcodeError = 'Failed to look up barcode.';
      }
      barcodeLoading = false;
      // Restart camera for retry
      if (barcodeSupported) setTimeout(() => startCamera(), 500);
    }
  }

  async function handleBarcodeFileUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    barcodeLoading = true;
    barcodeError = '';
    try {
      const bitmap = await createImageBitmap(file);
      // @ts-ignore
      const detector = new BarcodeDetector({ formats: ['ean_13', 'ean_8', 'upc_a', 'upc_e', 'code_128'] });
      const barcodes = await detector.detect(bitmap);
      if (barcodes.length === 0) {
        barcodeError = 'No barcode detected in image.';
        barcodeLoading = false;
        return;
      }
      await lookupBarcode(barcodes[0].rawValue);
    } catch {
      barcodeError = 'Failed to read barcode from image.';
      barcodeLoading = false;
    }
  }

  // Recipes
  async function logFromRecipe(recipe: Recipe) {
    try {
      await api.post(`/meals/from-recipe/${recipe.id}`, { meal_type: mealType });
      goto('/dashboard');
    } catch {}
  }

  function recipeCalories(recipe: Recipe): number {
    return recipe.foods.reduce((sum, f) => sum + f.calories, 0);
  }

  async function saveAsRecipe() {
    if (!recipeName.trim()) return;
    recipeSaving = true;
    try {
      await api.post('/recipes', {
        name: recipeName.trim(),
        description: recipeDescription.trim(),
        foods: foods
      });
      showSaveRecipeModal = false;
      recipeName = '';
      recipeDescription = '';
      await loadRecipes();
    } catch {}
    finally { recipeSaving = false; }
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
          <!-- Photo Upload -->
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

          <!-- Food Items -->
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
                  class="rounded-lg border border-slate-700 px-3 py-2 text-sm font-medium text-slate-400 hover:text-white hover:bg-slate-800 transition"
                >
                  + Manual
                </button>
              </div>
            </div>

            <!-- Food Search -->
            <div class="mb-4 relative">
              <div class="flex gap-2">
                <div class="relative flex-1">
                  <div class="pointer-events-none absolute inset-y-0 left-3 flex items-center">
                    {#if searchLoading}
                      <div class="h-4 w-4 animate-spin rounded-full border-2 border-joule-500 border-t-transparent"></div>
                    {:else}
                      <svg class="h-4 w-4 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                      </svg>
                    {/if}
                  </div>
                  <input
                    type="text"
                    bind:value={searchQuery}
                    oninput={handleSearchInput}
                    onblur={() => setTimeout(closeSearchDropdown, 150)}
                    placeholder="Search foods database..."
                    class="w-full rounded-lg border border-slate-700 bg-surface pl-9 pr-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                  />
                </div>
                <!-- Barcode scan button -->
                <button
                  type="button"
                  onclick={openBarcodeModal}
                  title="Scan barcode"
                  class="flex items-center justify-center rounded-lg border border-slate-700 px-3 py-2.5 text-slate-400 hover:text-white hover:border-slate-600 hover:bg-slate-800 transition"
                >
                  <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M3 5h2M7 5h2M3 7v2M3 15v2M3 19h2M7 19h2M17 5h2M19 5v2M19 15v2M17 19h2M19 19v0M11 5v14M11 5h2M11 19h2M15 7v10" />
                  </svg>
                </button>
              </div>

              <!-- Search dropdown -->
              {#if showSearchDropdown}
                <div class="absolute left-0 right-10 top-full z-40 mt-1 rounded-xl border border-slate-700 bg-surface shadow-xl overflow-hidden">
                  {#if searchLoading}
                    <div class="flex items-center justify-center py-6">
                      <div class="h-5 w-5 animate-spin rounded-full border-2 border-joule-500 border-t-transparent"></div>
                    </div>
                  {:else if searchResults.length === 0}
                    <p class="px-4 py-3 text-sm text-slate-500">No results found.</p>
                  {:else}
                    <div class="max-h-72 overflow-y-auto">
                      {#each searchResults as result}
                        <button
                          type="button"
                          onmousedown={() => addFoodFromResult(result)}
                          class="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-slate-800 transition border-b border-slate-800 last:border-0"
                        >
                          <div class="min-w-0 flex-1">
                            <div class="flex items-center gap-2">
                              <span class="truncate text-sm font-medium text-white">{result.name}</span>
                              {#if result.brand}
                                <span class="shrink-0 text-xs text-slate-500">{result.brand}</span>
                              {/if}
                              {#if result.source === 'openfoodfacts'}
                                <span class="shrink-0 rounded bg-orange-500/10 px-1 py-0.5 text-xs text-orange-400">OFN</span>
                              {:else}
                                <span class="shrink-0 rounded bg-blue-500/10 px-1 py-0.5 text-xs text-blue-400">DB</span>
                              {/if}
                            </div>
                            <p class="mt-0.5 text-xs text-slate-500">
                              {result.protein_g}g P · {result.carbs_g}g C · {result.fat_g}g F
                              {#if result.serving_size} · {result.serving_size}{/if}
                            </p>
                          </div>
                          <span class="shrink-0 text-sm font-semibold text-white">{result.calories} kcal</span>
                        </button>
                      {/each}
                    </div>
                  {/if}
                </div>
              {/if}
            </div>

            <!-- My Recipes section -->
            {#if recipes.length > 0 || recipesLoading}
              <div class="mb-4">
                <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-500">My Recipes</p>
                {#if recipesLoading}
                  <div class="flex items-center gap-2 py-2">
                    <div class="h-4 w-4 animate-spin rounded-full border-2 border-joule-500 border-t-transparent"></div>
                    <span class="text-xs text-slate-500">Loading recipes...</span>
                  </div>
                {:else}
                  <div class="flex gap-3 overflow-x-auto pb-2">
                    {#each recipes as recipe}
                      <button
                        type="button"
                        onclick={() => logFromRecipe(recipe)}
                        class="flex-none rounded-xl border border-slate-700 bg-surface px-4 py-3 text-left hover:border-joule-500/40 hover:bg-joule-500/5 transition min-w-[140px]"
                      >
                        <p class="truncate text-sm font-medium text-white max-w-[130px]">{recipe.name}</p>
                        <p class="mt-1 text-xs font-semibold text-joule-400">{recipeCalories(recipe)} kcal</p>
                        <p class="text-xs text-slate-500">{recipe.foods.length} item{recipe.foods.length !== 1 ? 's' : ''}</p>
                      </button>
                    {/each}
                  </div>
                {/if}
              </div>
            {/if}

            <!-- Food list -->
            {#if foods.length > 0}
              <div class="space-y-2">
                {#each foods as food, i}
                  <div class="flex items-center justify-between rounded-lg border border-slate-800 bg-surface px-4 py-3">
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-2">
                        <span class="truncate text-sm font-medium text-white">{food.name}</span>
                        {#if food.source === 'ai'}
                          <span class="rounded bg-joule-500/10 px-1.5 py-0.5 text-xs text-joule-400">AI</span>
                        {:else if food.source === 'db'}
                          <span class="rounded bg-blue-500/10 px-1.5 py-0.5 text-xs text-blue-400">DB</span>
                        {:else if food.source === 'barcode'}
                          <span class="rounded bg-green-500/10 px-1.5 py-0.5 text-xs text-green-400">Scan</span>
                        {:else if food.source === 'recipe'}
                          <span class="rounded bg-purple-500/10 px-1.5 py-0.5 text-xs text-purple-400">Recipe</span>
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

              <!-- Save as Recipe button -->
              <div class="mt-3 flex justify-end">
                <button
                  type="button"
                  onclick={() => { showSaveRecipeModal = true; }}
                  class="flex items-center gap-1.5 rounded-lg border border-slate-700 px-3 py-1.5 text-xs font-medium text-slate-400 hover:text-purple-400 hover:border-purple-500/40 hover:bg-purple-500/5 transition"
                >
                  <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
                  </svg>
                  Save as Recipe
                </button>
              </div>
            {:else}
              <p class="py-4 text-center text-sm text-slate-500">No foods added yet. Search above, scan a barcode, or upload a photo.</p>
            {/if}

            <!-- Manual add form -->
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

          <!-- Meal Details -->
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

        <!-- Sidebar summary -->
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

  <!-- Barcode Scan Modal -->
  {#if showBarcodeModal}
    <div role="dialog" aria-modal="true" aria-label="Scan Barcode" class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 bg-black/60 backdrop-blur-sm" onclick={(e) => { if (e.target === e.currentTarget) closeBarcodeModal(); }}>
      <div class="w-full max-w-md rounded-2xl border border-slate-700 bg-surface shadow-2xl">
        <div class="flex items-center justify-between border-b border-slate-800 px-5 py-4">
          <div>
            <h3 class="font-semibold text-white">Scan Barcode</h3>
            <p class="mt-0.5 text-xs text-slate-400">
              {barcodeSupported ? 'Point camera at a barcode' : 'Upload a photo of the barcode'}
            </p>
          </div>
          <button aria-label="Close barcode scanner" onclick={closeBarcodeModal} class="text-slate-500 hover:text-white transition">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div class="p-5">
          {#if barcodeSuccess}
            <div class="flex items-center gap-3 rounded-lg border border-green-500/20 bg-green-500/10 px-4 py-3">
              <svg class="h-5 w-5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
              <p class="text-sm text-green-400">{barcodeSuccess}</p>
            </div>
          {:else if barcodeLoading}
            <div class="flex flex-col items-center gap-3 py-8">
              <div class="h-8 w-8 animate-spin rounded-full border-2 border-joule-500 border-t-transparent"></div>
              <p class="text-sm text-slate-400">Looking up product...</p>
            </div>
          {:else if barcodeSupported}
            <!-- Live camera feed -->
            <div class="relative overflow-hidden rounded-xl bg-black aspect-video">
              <!-- svelte-ignore a11y_media_has_caption -->
              <video
                bind:this={videoElement}
                autoplay
                playsinline
                muted
                class="h-full w-full object-cover"
              ></video>
              <!-- Scan frame overlay -->
              <div class="absolute inset-0 flex items-center justify-center pointer-events-none">
                <div class="h-32 w-64 rounded-lg border-2 border-joule-500/70 shadow-[0_0_0_9999px_rgba(0,0,0,0.3)]"></div>
              </div>
            </div>
            <p class="mt-3 text-center text-xs text-slate-500">Scanning automatically...</p>
            {#if barcodeError}
              <p class="mt-2 text-center text-sm text-red-400">{barcodeError}</p>
            {/if}
          {:else}
            <!-- File upload fallback -->
            <div class="space-y-4">
              <p class="text-sm text-slate-400">BarcodeDetector is not supported in this browser. Upload a photo of the barcode instead.</p>
              <label class="flex cursor-pointer flex-col items-center gap-3 rounded-lg border-2 border-dashed border-slate-700 px-6 py-10 transition hover:border-slate-600">
                <svg class="h-10 w-10 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                <span class="text-sm text-slate-400">Upload barcode photo</span>
                <input
                  type="file"
                  accept="image/*"
                  class="hidden"
                  onchange={handleBarcodeFileUpload}
                />
              </label>
              {#if barcodeError}
                <p class="text-center text-sm text-red-400">{barcodeError}</p>
              {/if}
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}

  <!-- Save as Recipe Modal -->
  {#if showSaveRecipeModal}
    <div role="dialog" aria-modal="true" aria-label="Save as Recipe" class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 bg-black/60 backdrop-blur-sm" onclick={(e) => { if (e.target === e.currentTarget) showSaveRecipeModal = false; }}>
      <div class="w-full max-w-sm rounded-2xl border border-slate-700 bg-surface shadow-2xl">
        <div class="flex items-center justify-between border-b border-slate-800 px-5 py-4">
          <div>
            <h3 class="font-semibold text-white">Save as Recipe</h3>
            <p class="mt-0.5 text-xs text-slate-400">{foods.length} food item{foods.length !== 1 ? 's' : ''} will be saved</p>
          </div>
          <button aria-label="Close save recipe" onclick={() => (showSaveRecipeModal = false)} class="text-slate-500 hover:text-white transition">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="space-y-4 p-5">
          <div>
            <label for="recipe-name" class="mb-1.5 block text-xs font-medium text-slate-400">Recipe Name</label>
            <input
              id="recipe-name"
              type="text"
              bind:value={recipeName}
              placeholder="e.g. Morning oats bowl"
              class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
            />
          </div>
          <div>
            <label for="recipe-description" class="mb-1.5 block text-xs font-medium text-slate-400">Description (Optional)</label>
            <input
              id="recipe-description"
              type="text"
              bind:value={recipeDescription}
              placeholder="Brief description..."
              class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
            />
          </div>
          <div class="flex justify-end gap-2 pt-1">
            <button
              type="button"
              onclick={() => (showSaveRecipeModal = false)}
              class="rounded-lg border border-slate-700 px-4 py-2 text-sm font-medium text-slate-400 hover:text-white hover:bg-slate-800 transition"
            >
              Cancel
            </button>
            <button
              type="button"
              onclick={saveAsRecipe}
              disabled={!recipeName.trim() || recipeSaving}
              class="rounded-lg bg-joule-500 px-4 py-2 text-sm font-semibold text-slate-900 hover:bg-joule-400 disabled:opacity-50 disabled:cursor-not-allowed transition"
            >
              {recipeSaving ? 'Saving...' : 'Save Recipe'}
            </button>
          </div>
        </div>
      </div>
    </div>
  {/if}
{/if}
