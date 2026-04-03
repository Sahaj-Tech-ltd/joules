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
  let showMediaPicker = $state(false);
  let cameraInputRef = $state<HTMLInputElement | undefined>(undefined);
  let galleryInputRef = $state<HTMLInputElement | undefined>(undefined);
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
  let barcodeLoading = $state(false);
  let barcodeError = $state('');
  let barcodeSuccess = $state('');
  let barcodeScanPhoto = $state<string | null>(null);
  let barcodeScanPreview = $state<string | null>(null);

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

  let favorites = $state<Array<{id: string; name: string; calories: number; protein_g: number; carbs_g: number; fat_g: number; fiber_g: number; serving_size: string; source: string}>>([]);
  let showFavorites = $state(false);

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

    loadRecipes();
    loadFavorites();
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

  async function loadFavorites() {
    try {
      favorites = await api.get<typeof favorites>('/favorites');
    } catch { favorites = []; }
  }

  async function toggleFavorite(food: FoodItem) {
    const existing = favorites.find(f => f.name.toLowerCase() === food.name.toLowerCase());
    if (existing) {
      await api.del(`/favorites/${existing.id}`);
      favorites = favorites.filter(f => f.id !== existing.id);
    } else {
      const fav = await api.post<typeof favorites[0]>('/favorites', {
        name: food.name,
        calories: food.calories,
        protein_g: food.protein_g,
        carbs_g: food.carbs_g,
        fat_g: food.fat_g,
        fiber_g: food.fiber_g,
        serving_size: food.serving_size,
        source: food.source
      });
      favorites = [...favorites, fav];
    }
  }

  function isFavorited(foodName: string): boolean {
    return favorites.some(f => f.name.toLowerCase() === foodName.toLowerCase());
  }

  function addFoodFromFavorite(fav: typeof favorites[0]) {
    foods = [...foods, {
      name: fav.name,
      calories: fav.calories,
      protein_g: fav.protein_g,
      carbs_g: fav.carbs_g,
      fat_g: fav.fat_g,
      fiber_g: fav.fiber_g,
      serving_size: fav.serving_size,
      source: 'manual'
    }];
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
    barcodeScanPhoto = null;
    barcodeScanPreview = null;
  }

  function closeBarcodeModal() {
    showBarcodeModal = false;
    barcodeError = '';
    barcodeSuccess = '';
    barcodeLoading = false;
    barcodeScanPhoto = null;
    barcodeScanPreview = null;
  }

  async function submitBarcodeScan() {
    if (!barcodeScanPhoto) return;
    barcodeLoading = true;
    barcodeError = '';
    try {
      const result = await api.post<FoodResult>('/foods/barcode-scan', {
        photo: barcodeScanPhoto
      });
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
      barcodeError = err instanceof Error ? err.message : 'Failed to identify product.';
      barcodeLoading = false;
    }
  }

  function handleBarcodePhotoSelect(file: File) {
    if (!file.type.startsWith('image/')) return;
    const reader = new FileReader();
    reader.onload = (e) => {
      barcodeScanPhoto = e.target?.result as string;
      barcodeScanPreview = e.target?.result as string;
    };
    reader.readAsDataURL(file);
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
      const now = new Date();
      const local = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1);
      const dateStr = `${local.getFullYear()}-${String(local.getMonth() + 1).padStart(2, '0')}-${String(local.getDate()).padStart(2, '0')}`;
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
  <div class="flex h-screen items-center justify-center bg-background">
    <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
  </div>
{:else}
  <div class="flex min-h-screen overflow-x-hidden">
    <Sidebar activePage="log" />

    <main class="flex-1 min-w-0 overflow-x-hidden p-4 lg:p-8" style="padding-bottom: calc(5rem + env(safe-area-inset-bottom, 0px));">
      <div class="mb-6 flex items-center justify-between">
        <div>
          <h1 class="font-display text-2xl font-bold text-foreground">Log a Meal</h1>
          <p class="mt-0.5 text-xs text-muted-foreground">Add your meal to track nutrition</p>
        </div>
        <button
          onclick={() => { authToken.set(null); goto('/login'); }}
          class="rounded-xl border border-border bg-accent/50 px-3 py-1.5 text-xs font-medium text-foreground hover:text-foreground hover:bg-accent/50 transition"
        >
          Sign out
        </button>
      </div>

      <div class="grid grid-cols-1 gap-6 xl:grid-cols-3">
        <div class="space-y-6 xl:col-span-2">
          <!-- Photo Upload -->
          <div class="rounded-2xl border border-border bg-card p-5">
            <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Photo (Optional)</p>
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
                  class="absolute top-2 right-2 rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground hover:text-foreground transition"
                >
                  Remove photo
                </button>
              </div>
              <div class="mt-4">
                <label for="portion-hint" class="mb-1.5 block text-xs font-medium text-foreground">
                  Help the AI estimate portions
                  <span class="ml-1 text-muted-foreground">(optional but helps accuracy)</span>
                </label>
                <input
                  id="portion-hint"
                  type="text"
                  bind:value={portionHint}
                  placeholder='e.g. "double patty burger", "half a pizza", "large bowl"'
                  class="w-full rounded-xl border border-border bg-secondary px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
                />
              </div>
              <p class="mt-2 text-xs text-muted-foreground">AI will analyze your photo — you can review and edit items after logging.</p>
            {:else}
              <!-- Hidden inputs: one for camera, one for gallery -->
              <input
                bind:this={cameraInputRef}
                type="file"
                accept="image/*"
                capture="environment"
                class="hidden"
                onchange={(e) => {
                  const file = (e.target as HTMLInputElement).files?.[0];
                  if (file) handleFileSelect(file);
                }}
              />
              <input
                bind:this={galleryInputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                class="hidden"
                onchange={(e) => {
                  const file = (e.target as HTMLInputElement).files?.[0];
                  if (file) handleFileSelect(file);
                }}
              />

              <!-- Tap area -->
              <div
                role="button"
                tabindex="0"
                class="flex cursor-pointer flex-col items-center gap-3 rounded-3xl border-2 border-dashed border-border px-6 py-12 transition hover:border-border hover:bg-white/3"
                ondrop={handleDrop}
                ondragover={handleDragOver}
                onclick={() => { showMediaPicker = true; }}
                onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') showMediaPicker = true; }}
              >
                <svg class="h-10 w-10 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                <span class="text-sm text-foreground">Take a photo or upload an image</span>
                <span class="text-xs text-muted-foreground">JPG, PNG, or WebP</span>
              </div>

              <!-- Media picker bottom sheet -->
              {#if showMediaPicker}
                <div
                  class="fixed inset-0 z-50 flex items-end bg-black/60 backdrop-blur-sm"
                  onclick={() => { showMediaPicker = false; }}
                  role="presentation"
                >
                  <div
                    class="w-full rounded-t-2xl border-t border-border/80 bg-secondary p-4 pb-8 shadow-2xl"
                    onclick={(e) => e.stopPropagation()}
                    role="dialog"
                    aria-modal="true"
                    aria-label="Add photo"
                  >
                    <div class="mx-auto mb-5 h-1 w-10 rounded-full bg-accent"></div>
                    <p class="mb-4 text-center text-sm font-semibold text-foreground">Add Photo</p>
                    <div class="flex flex-col gap-2">
                      <button
                        class="flex items-center gap-3 rounded-xl bg-accent px-4 py-3.5 text-sm font-medium text-foreground transition hover:bg-accent active:scale-[0.98]"
                        onclick={() => { showMediaPicker = false; cameraInputRef?.click(); }}
                      >
                        <svg class="h-5 w-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M6.827 6.175A2.31 2.31 0 015.186 7.23c-.38.054-.757.112-1.134.175C2.999 7.58 2.25 8.507 2.25 9.574V18a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9.574c0-1.067-.75-1.994-1.802-2.169a47.865 47.865 0 00-1.134-.175 2.31 2.31 0 01-1.64-1.055l-.822-1.316a2.192 2.192 0 00-1.736-1.039 48.774 48.774 0 00-5.232 0 2.192 2.192 0 00-1.736 1.039l-.821 1.316z" />
                          <path stroke-linecap="round" stroke-linejoin="round" d="M16.5 12.75a4.5 4.5 0 11-9 0 4.5 4.5 0 019 0zM18.75 10.5h.008v.008h-.008V10.5z" />
                        </svg>
                        Take Photo
                      </button>
                      <button
                        class="flex items-center gap-3 rounded-xl bg-accent px-4 py-3.5 text-sm font-medium text-foreground transition hover:bg-accent active:scale-[0.98]"
                        onclick={() => { showMediaPicker = false; galleryInputRef?.click(); }}
                      >
                        <svg class="h-5 w-5 text-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 001.5-1.5V6a1.5 1.5 0 00-1.5-1.5H3.75A1.5 1.5 0 002.25 6v12a1.5 1.5 0 001.5 1.5zm10.5-11.25h.008v.008h-.008V6.75zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z" />
                        </svg>
                        Choose from Library
                      </button>
                      <button
                        class="mt-1 rounded-xl border border-border px-4 py-3 text-sm font-medium text-foreground transition hover:text-foreground"
                        onclick={() => { showMediaPicker = false; }}
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                </div>
              {/if}
            {/if}
          </div>

          <!-- Food Items -->
          <div class="rounded-2xl border border-border bg-card p-5">
            <div class="mb-4 flex items-center justify-between">
              <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Food Items</p>
              <div class="flex gap-2">
                <button
                  type="button"
                  onclick={openLeftovers}
                  class="flex items-center gap-1.5 rounded-xl border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:text-primary hover:border-primary/30 hover:bg-primary/5 transition"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                  </svg>
                  Leftovers?
                </button>
                <button
                  type="button"
                  onclick={() => (showAddFood = !showAddFood)}
                  class="rounded-xl border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:text-foreground hover:bg-accent/50 transition"
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
                      <div class="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
                    {:else}
                      <svg class="h-4 w-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
                    class="w-full rounded-2xl border border-border bg-secondary pl-10 pr-4 py-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
                  />
                </div>
                <!-- Barcode scan button -->
                <button
                  type="button"
                  onclick={openBarcodeModal}
                  title="Scan barcode"
                  class="flex items-center justify-center rounded-2xl border border-border px-3 py-3 text-foreground hover:text-foreground hover:border-border hover:bg-accent/50 transition"
                >
                  <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M3 5h2M7 5h2M3 7v2M3 15v2M3 19h2M7 19h2M17 5h2M19 5v2M19 15v2M17 19h2M19 19v0M11 5v14M11 5h2M11 19h2M15 7v10" />
                  </svg>
                </button>
              </div>

              <!-- Search dropdown -->
              {#if showSearchDropdown}
                <div class="absolute left-0 right-10 top-full z-40 mt-1 rounded-2xl border border-border bg-secondary shadow-xl overflow-hidden">
                  {#if searchLoading}
                    <div class="flex items-center justify-center py-6">
                      <div class="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
                    </div>
                  {:else if searchResults.length === 0}
                    <p class="px-4 py-3 text-sm text-muted-foreground">No results found.</p>
                  {:else}
                    <div class="max-h-72 overflow-y-auto">
                      {#each searchResults as result}
                        <button
                          type="button"
                          onmousedown={() => addFoodFromResult(result)}
                          class="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-accent transition border-b border-border last:border-0"
                        >
                          <div class="min-w-0 flex-1">
                            <div class="flex items-center gap-2">
                              <span class="truncate text-sm font-medium text-foreground">{result.name}</span>
                              {#if result.brand}
                                <span class="shrink-0 text-xs text-muted-foreground">{result.brand}</span>
                              {/if}
                              {#if result.source === 'openfoodfacts'}
                                <span class="shrink-0 rounded bg-orange-500/10 px-1 py-0.5 text-xs text-orange-400">OFN</span>
                              {:else}
                                <span class="shrink-0 rounded bg-blue-500/10 px-1 py-0.5 text-xs text-blue-400">DB</span>
                              {/if}
                            </div>
                            <p class="mt-0.5 text-xs text-muted-foreground">
                              {result.protein_g}g P · {result.carbs_g}g C · {result.fat_g}g F
                              {#if result.serving_size} · {result.serving_size}{/if}
                            </p>
                          </div>
                          <span class="shrink-0 text-sm font-semibold text-foreground">{result.calories} kcal</span>
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
                <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">My Recipes</p>
                {#if recipesLoading}
                  <div class="flex items-center gap-2 py-2">
                    <div class="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
                    <span class="text-xs text-muted-foreground">Loading recipes...</span>
                  </div>
                {:else}
                  <div class="flex gap-3 overflow-x-auto pb-2">
                    {#each recipes as recipe}
                      <button
                        type="button"
                        onclick={() => logFromRecipe(recipe)}
                        class="flex-none rounded-xl border border-border bg-card/60 px-4 py-3 text-left hover:border-primary/30 hover:bg-primary/5 transition min-w-[140px]"
                      >
                        <p class="truncate text-sm font-medium text-foreground max-w-[130px]">{recipe.name}</p>
                        <p class="mt-1 text-xs font-semibold text-primary">{recipeCalories(recipe)} kcal</p>
                        <p class="text-xs text-muted-foreground">{recipe.foods.length} item{recipe.foods.length !== 1 ? 's' : ''}</p>
                      </button>
                    {/each}
                  </div>
                {/if}
              </div>
            {/if}

            {#if favorites.length > 0}
              <div class="mb-4">
                <div class="flex items-center justify-between mb-2">
                  <p class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Favorites</p>
                  <button
                    type="button"
                    onclick={() => showFavorites = !showFavorites}
                    class="text-xs text-primary hover:underline"
                  >{showFavorites ? 'Hide' : 'Show All'}</button>
                </div>
                <div class="flex gap-2 overflow-x-auto pb-2">
                  {#each (showFavorites ? favorites : favorites.slice(0, 5)) as fav}
                    <button
                      type="button"
                      onclick={() => addFoodFromFavorite(fav)}
                      class="flex-none rounded-xl border border-border bg-card/60 px-3 py-2 text-left hover:border-primary/30 hover:bg-primary/5 transition min-w-[110px]"
                    >
                      <p class="truncate text-xs font-medium text-foreground max-w-[100px]">{fav.name}</p>
                      <p class="mt-0.5 text-xs font-semibold text-primary">{fav.calories} kcal</p>
                    </button>
                  {/each}
                </div>
              </div>
            {/if}

            <!-- Food list -->
            {#if foods.length > 0}
              <div class="space-y-2">
                {#each foods as food, i}
                  <div class="flex items-center justify-between rounded-xl border border-border bg-card/60 px-4 py-3">
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-2">
                        <span class="truncate text-sm font-medium text-foreground">{food.name}</span>
                        {#if food.source === 'ai'}
                          <span class="rounded bg-primary/10 px-1.5 py-0.5 text-xs text-primary">AI</span>
                        {:else if food.source === 'db'}
                          <span class="rounded bg-blue-500/10 px-1.5 py-0.5 text-xs text-blue-400">DB</span>
                        {:else if food.source === 'barcode'}
                          <span class="rounded bg-green-500/10 px-1.5 py-0.5 text-xs text-green-400">Scan</span>
                        {:else if food.source === 'recipe'}
                          <span class="rounded bg-purple-500/10 px-1.5 py-0.5 text-xs text-purple-400">Recipe</span>
                        {/if}
                      </div>
                      <div class="mt-0.5 text-xs text-muted-foreground">
                        {food.protein_g}g P · {food.carbs_g}g C · {food.fat_g}g F{#if food.fiber_g} · {food.fiber_g}g fiber{/if}
                        {#if food.serving_size}
                          · {food.serving_size}
                        {/if}
                      </div>
                    </div>
                    <div class="flex items-center gap-3">
                      <span class="text-sm font-medium text-foreground">{food.calories} kcal</span>
                      <button
                        type="button"
                        onclick={() => toggleFavorite(food)}
                        aria-label={isFavorited(food.name) ? 'Unfavorite' : 'Favorite'}
                        class="{isFavorited(food.name) ? 'text-yellow-400' : 'text-muted-foreground hover:text-yellow-400'} transition"
                      >
                        <svg class="h-4 w-4" fill={isFavorited(food.name) ? 'currentColor' : 'none'} viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M11.48 3.499a.562.562 0 011.04 0l2.125 5.111a.563.563 0 00.475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 00-.182.557l1.285 5.385a.562.562 0 01-.84.61l-4.725-2.885a.563.563 0 00-.586 0L6.982 20.54a.562.562 0 01-.84-.61l1.285-5.386a.562.562 0 00-.182-.557l-4.204-3.602a.563.563 0 01.321-.988l5.518-.442a.563.563 0 00.475-.345L11.48 3.5z" />
                        </svg>
                      </button>
                      <button
                        type="button"
                        onclick={() => removeFood(i)}
                        aria-label="Remove food"
                        class="text-muted-foreground hover:text-red-400 transition"
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
                  class="flex items-center gap-1.5 rounded-xl border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:text-purple-400 hover:border-purple-500/30 hover:bg-purple-500/5 transition"
                >
                  <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
                  </svg>
                  Save as Recipe
                </button>
              </div>
            {:else}
              <p class="py-4 text-center text-sm text-muted-foreground">No foods added yet. Search above, scan a barcode, or upload a photo.</p>
            {/if}

            <!-- Manual add form -->
            {#if showAddFood}
              <div class="mt-4 space-y-3 rounded-2xl border border-border bg-secondary p-4">
                <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
                  <div class="col-span-2 sm:col-span-4">
                    <input
                      type="text"
                      bind:value={newFood.name}
                      placeholder="Food name"
                      class="w-full rounded-xl border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
                    />
                  </div>
                  <input
                    type="number"
                    bind:value={newFood.calories}
                    placeholder="Calories"
                    min="0"
                    class="w-full rounded-xl border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
                  />
                  <input
                    type="number"
                    bind:value={newFood.protein_g}
                    placeholder="Protein (g)"
                    min="0"
                    class="w-full rounded-xl border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
                  />
                  <input
                    type="number"
                    bind:value={newFood.carbs_g}
                    placeholder="Carbs (g)"
                    min="0"
                    class="w-full rounded-xl border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
                  />
                  <input
                    type="number"
                    bind:value={newFood.fat_g}
                    placeholder="Fat (g)"
                    min="0"
                    class="w-full rounded-xl border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
                  />
                  <input
                    type="number"
                    bind:value={newFood.fiber_g}
                    placeholder="Fiber (g)"
                    min="0"
                    class="w-full rounded-xl border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
                  />
                  <input
                    type="text"
                    bind:value={newFood.serving_size}
                    placeholder="Serving (e.g. 1 cup)"
                    class="w-full rounded-xl border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
                  />
                </div>
                <div class="flex justify-end gap-2">
                  <button
                    type="button"
                    onclick={() => (showAddFood = false)}
                    class="rounded-xl border border-border px-4 py-2 text-sm font-medium text-foreground hover:text-foreground hover:bg-accent/50 transition"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    onclick={addFood}
                    disabled={!newFood.name.trim()}
                    class="rounded-xl bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 disabled:opacity-50 disabled:cursor-not-allowed transition"
                  >
                    Add
                  </button>
                </div>
              </div>
            {/if}
          </div>

          <!-- Meal Details -->
          <div class="rounded-2xl border border-border bg-card p-5">
            <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Meal Details</p>
            <div class="space-y-4">
              <div>
                <span class="mb-2.5 block text-xs text-foreground">Meal Type</span>
                <div class="flex flex-wrap gap-2">
                  {#each mealTypes as type}
                    <button
                      type="button"
                      onclick={() => (mealType = type)}
                      class="rounded-xl border px-4 py-2 text-xs font-semibold capitalize transition {mealType === type ? 'border-primary/50 bg-primary/15 text-primary' : 'border-border text-foreground hover:border-border hover:text-foreground/80'}"
                    >
                      {type}
                    </button>
                  {/each}
                </div>
              </div>
              <div>
                <label for="note" class="mb-1.5 block text-xs text-foreground">Note (Optional)</label>
                <input
                  id="note"
                  type="text"
                  bind:value={note}
                  placeholder="Add a note about this meal..."
                  class="w-full rounded-xl border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- Sidebar summary -->
        <div class="space-y-4">
          <div class="rounded-2xl border border-border bg-card p-5">
            <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Summary</p>
            {#if foods.length > 0}
              <div class="mb-4 rounded-2xl border border-primary/20 bg-primary/5 p-4 text-center">
                <p class="font-display text-3xl font-bold text-foreground">{totalCalories}</p>
                <p class="text-xs text-foreground">total calories</p>
              </div>
              <div class="space-y-3">
                {#each foods as food}
                  <div class="flex items-center justify-between text-sm">
                    <span class="truncate text-foreground">{food.name}</span>
                    <span class="ml-2 shrink-0 text-foreground">{food.calories}</span>
                  </div>
                {/each}
              </div>
            {:else}
              <p class="py-4 text-center text-sm text-muted-foreground">Add foods to see the summary</p>
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
            class="w-full rounded-2xl bg-primary px-4 py-3.5 text-base font-semibold text-primary-foreground hover:bg-primary/80 disabled:opacity-50 disabled:cursor-not-allowed transition active:scale-[0.98]"
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
      <div class="w-full max-w-lg rounded-2xl border border-border bg-secondary shadow-2xl max-h-[80vh] flex flex-col">
        <div class="flex items-center justify-between border-b border-border px-5 py-4">
          <div>
            <h3 class="font-semibold text-foreground">Yesterday's Leftovers</h3>
            <p class="mt-0.5 text-xs text-foreground">Select foods to carry forward to today</p>
          </div>
          <button onclick={() => (showLeftovers = false)} class="text-muted-foreground hover:text-foreground transition">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div class="flex-1 overflow-y-auto px-5 py-3 space-y-4">
          {#if leftoverLoading}
            <div class="flex items-center justify-center py-10">
              <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
            </div>
          {:else if leftoverMeals.length === 0}
            <div class="py-10 text-center">
              <p class="text-sm text-foreground">No meals logged yesterday.</p>
            </div>
          {:else}
            {#each leftoverMeals as meal}
              <div>
                <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground capitalize">{meal.meal_type}</p>
                <div class="space-y-1.5">
                  {#each meal.foods as food}
                    {@const selected = selectedFoodIds.has(food.id)}
                    <button
                      type="button"
                      onclick={() => toggleFood(food.id)}
                      class="w-full flex items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition {selected ? 'border-primary/40 bg-primary/10' : 'border-border hover:border-border bg-secondary'}"
                    >
                      <div class="flex h-4 w-4 shrink-0 items-center justify-center rounded border {selected ? 'border-primary bg-primary' : 'border-border'}">
                        {#if selected}
                          <svg class="h-3 w-3 text-primary-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
                          </svg>
                        {/if}
                      </div>
                      <div class="min-w-0 flex-1">
                        <p class="truncate text-sm font-medium text-foreground">{food.name}</p>
                        <p class="text-xs text-muted-foreground">{food.protein_g}g P · {food.carbs_g}g C · {food.fat_g}g F{#if food.serving_size} · {food.serving_size}{/if}</p>
                      </div>
                      <span class="shrink-0 text-sm font-medium {selected ? 'text-primary' : 'text-foreground'}">{food.calories} kcal</span>
                    </button>
                  {/each}
                </div>
              </div>
            {/each}
          {/if}
        </div>

        {#if !leftoverLoading && leftoverMeals.length > 0}
          <div class="border-t border-border px-5 py-4 space-y-3">
            <label class="flex items-center gap-3 cursor-pointer">
              <div
                class="relative h-5 w-9 rounded-full transition {removeFromYesterday ? 'bg-primary' : 'bg-accent'}"
                onclick={() => (removeFromYesterday = !removeFromYesterday)}
              >
                <div class="absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white shadow transition-transform {removeFromYesterday ? 'translate-x-4' : ''}"></div>
              </div>
              <span class="text-sm text-foreground">Remove from yesterday's log</span>
            </label>
            <div class="flex items-center justify-between gap-3">
              <p class="text-xs text-muted-foreground">
                {selectedFoodIds.size} item{selectedFoodIds.size !== 1 ? 's' : ''} selected
                {#if selectedFoodIds.size > 0}
                  · {[...leftoverMeals.flatMap(m => m.foods)].filter(f => selectedFoodIds.has(f.id)).reduce((s, f) => s + f.calories, 0)} kcal
                {/if}
              </p>
              <button
                type="button"
                onclick={carryForward}
                disabled={selectedFoodIds.size === 0 || carryingOver}
                class="rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 disabled:opacity-50 disabled:cursor-not-allowed transition"
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
      <div class="w-full max-w-md rounded-2xl border border-border bg-secondary shadow-2xl">
        <div class="flex items-center justify-between border-b border-border px-5 py-4">
          <div>
            <h3 class="font-semibold text-foreground">Scan Product</h3>
            <p class="mt-0.5 text-xs text-foreground">Take a photo of the barcode or product packaging</p>
          </div>
          <button aria-label="Close barcode scanner" onclick={closeBarcodeModal} class="text-muted-foreground hover:text-foreground transition">
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
              <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
              <p class="text-sm text-foreground">Identifying product...</p>
            </div>
          {:else if barcodeScanPreview}
            <div class="space-y-4">
              <div class="relative">
                <img src={barcodeScanPreview} alt="Scanned product" class="mx-auto max-h-48 rounded-lg object-contain" />
                <button
                  type="button"
                  onclick={() => { barcodeScanPhoto = null; barcodeScanPreview = null; }}
                  class="absolute top-2 right-2 rounded-lg border border-border bg-card px-2 py-1 text-xs text-foreground hover:text-foreground transition"
                >
                  Retake
                </button>
              </div>
              <button
                type="button"
                onclick={submitBarcodeScan}
                class="w-full rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition"
              >
                Identify Product
              </button>
            </div>
          {:else}
            <div class="flex gap-3">
              <label class="flex flex-1 cursor-pointer flex-col items-center gap-2 rounded-lg border-2 border-dashed border-border px-4 py-6 transition hover:border-primary/50">
                <svg class="h-8 w-8 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                <span class="text-xs text-foreground">Camera</span>
                <input type="file" accept="image/*" capture="environment" class="hidden" onchange={(e) => { const file = (e.target as HTMLInputElement).files?.[0]; if (file) handleBarcodePhotoSelect(file); }} />
              </label>
              <label class="flex flex-1 cursor-pointer flex-col items-center gap-2 rounded-lg border-2 border-dashed border-border px-4 py-6 transition hover:border-primary/50">
                <svg class="h-8 w-8 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <span class="text-xs text-foreground">Gallery</span>
                <input type="file" accept="image/*" class="hidden" onchange={(e) => { const file = (e.target as HTMLInputElement).files?.[0]; if (file) handleBarcodePhotoSelect(file); }} />
              </label>
            </div>
            <p class="mt-3 text-center text-xs text-muted-foreground">AI will identify the product from the photo</p>
          {/if}
          {#if barcodeError}
            <p class="mt-3 text-center text-sm text-red-400">{barcodeError}</p>
          {/if}
        </div>
      </div>
    </div>
  {/if}

  <!-- Save as Recipe Modal -->
  {#if showSaveRecipeModal}
    <div role="dialog" aria-modal="true" aria-label="Save as Recipe" class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 bg-black/60 backdrop-blur-sm" onclick={(e) => { if (e.target === e.currentTarget) showSaveRecipeModal = false; }}>
      <div class="w-full max-w-sm rounded-2xl border border-border bg-secondary shadow-2xl">
        <div class="flex items-center justify-between border-b border-border px-5 py-4">
          <div>
            <h3 class="font-semibold text-foreground">Save as Recipe</h3>
            <p class="mt-0.5 text-xs text-foreground">{foods.length} food item{foods.length !== 1 ? 's' : ''} will be saved</p>
          </div>
          <button aria-label="Close save recipe" onclick={() => (showSaveRecipeModal = false)} class="text-muted-foreground hover:text-foreground transition">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="space-y-4 p-5">
          <div>
            <label for="recipe-name" class="mb-1.5 block text-xs font-medium text-foreground">Recipe Name</label>
            <input
              id="recipe-name"
              type="text"
              bind:value={recipeName}
              placeholder="e.g. Morning oats bowl"
              class="w-full rounded-xl border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
            />
          </div>
          <div>
            <label for="recipe-description" class="mb-1.5 block text-xs font-medium text-foreground">Description (Optional)</label>
            <input
              id="recipe-description"
              type="text"
              bind:value={recipeDescription}
              placeholder="Brief description..."
              class="w-full rounded-xl border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
            />
          </div>
          <div class="flex justify-end gap-2 pt-1">
            <button
              type="button"
              onclick={() => (showSaveRecipeModal = false)}
              class="rounded-lg border border-border px-4 py-2 text-sm font-medium text-foreground hover:text-foreground hover:bg-accent transition"
            >
              Cancel
            </button>
            <button
              type="button"
              onclick={saveAsRecipe}
              disabled={!recipeName.trim() || recipeSaving}
              class="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 disabled:opacity-50 disabled:cursor-not-allowed transition"
            >
              {recipeSaving ? 'Saving...' : 'Save Recipe'}
            </button>
          </div>
        </div>
      </div>
    </div>
  {/if}
{/if}
