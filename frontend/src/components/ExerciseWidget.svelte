<script lang="ts">
  import { api } from '$lib/api';

  interface Exercise {
    id: string;
    name: string;
    duration_min: number;
    calories_burned: number;
    timestamp: string;
  }

  let { exercises = [] as Exercise[] }: { exercises: Exercise[] } = $props();

  let name = $state('');
  let duration = $state('');
  let loading = $state(false);
  let showPresets = $state(false);
  let durationInput: HTMLInputElement | undefined = $state(undefined);
  let successMsg = $state('');
  let successTimer = $state<ReturnType<typeof setTimeout> | null>(null);

  const exerciseCategories = [
    {
      icon: '<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M15.362 5.214A8.252 8.252 0 0112 21 8.25 8.25 0 016.038 7.047 8.287 8.287 0 009 9.601a8.983 8.983 0 013.361-6.867 8.21 8.21 0 003 2.48z" /></svg>',
      label: 'Cardio',
      exercises: ['Walking', 'Running', 'Cycling', 'Swimming', 'Jump Rope', 'Elliptical', 'Rowing', 'Stair Climbing']
    },
    {
      icon: '<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" /></svg>',
      label: 'Strength',
      exercises: ['Weight Training', 'Bodyweight', 'CrossFit', 'HIIT', 'Resistance Training']
    },
    {
      icon: '<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" /></svg>',
      label: 'Mind & Body',
      exercises: ['Yoga', 'Pilates', 'Stretching']
    },
    {
      icon: '<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25" /></svg>',
      label: 'Sports',
      exercises: ['Basketball', 'Soccer', 'Tennis', 'Dancing', 'Zumba']
    },
  ];

  function dispatchLogged() {
    const event = new CustomEvent('exercise-logged');
    window.dispatchEvent(event);
  }

  function formatTime(ts: string): string {
    return new Date(ts).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
  }

  function selectPreset(exerciseName: string) {
    name = exerciseName;
    showPresets = false;
    durationInput?.focus();
  }

  function showSuccess(msg: string) {
    if (successTimer) clearTimeout(successTimer);
    successMsg = msg;
    successTimer = setTimeout(() => { successMsg = ''; }, 2500);
  }

  async function logExercise() {
    const dur = parseInt(duration, 10);
    if (loading || !name.trim() || isNaN(dur) || dur <= 0) return;
    loading = true;
    try {
      await api.post('/exercises', { name: name.trim(), duration_min: dur });
      const label = name.trim();
      name = '';
      duration = '';
      showPresets = false;
      dispatchLogged();
      showSuccess(`${label} logged!`);
      api.post('/habits/checkin', {}).catch(() => {});
    } catch {
    } finally {
      loading = false;
    }
  }

  let totalBurned = $derived(exercises.reduce((sum, e) => sum + e.calories_burned, 0));
  let totalMinutes = $derived(exercises.reduce((sum, e) => sum + e.duration_min, 0));
</script>

<div class="rounded-2xl border border-border bg-card p-5">
  <div class="flex items-center justify-between mb-4">
    <div class="flex items-center gap-2">
      <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-orange-500/15 dark:bg-orange-500/10">
        <svg class="h-4 w-4 text-orange-600 dark:text-orange-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" d="M15.362 5.214A8.252 8.252 0 0112 21 8.25 8.25 0 016.038 7.047 8.287 8.287 0 009 9.601a8.983 8.983 0 013.361-6.867 8.21 8.21 0 003 2.48z" />
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 18a3.75 3.75 0 00.495-7.468 5.99 5.99 0 00-1.925 3.547 5.975 5.975 0 01-2.133-1.001A3.75 3.75 0 0012 18z" />
        </svg>
      </div>
      <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Exercise</p>
    </div>
    {#if exercises.length > 0}
      <div class="flex items-center gap-3 text-right">
        <div>
          <p class="text-xs font-bold text-orange-600 dark:text-orange-400">{totalBurned} kcal</p>
          <p class="text-[10px] text-muted-foreground">{totalMinutes} min</p>
        </div>
      </div>
    {/if}
  </div>

  {#if exercises.length > 0}
    <div class="flex flex-col mb-4 rounded-xl border border-border overflow-hidden">
      {#each exercises as exercise, i}
        <div class="flex items-center justify-between gap-3 px-3.5 py-3 {i < exercises.length - 1 ? 'border-b border-border' : ''}">
          <div class="min-w-0">
            <p class="text-sm font-medium text-foreground truncate">{exercise.name}</p>
            <p class="text-[11px] text-muted-foreground mt-0.5">{formatTime(exercise.timestamp)}</p>
          </div>
          <div class="flex items-center gap-3 flex-shrink-0">
            <span class="text-xs text-muted-foreground">{exercise.duration_min} min</span>
            <div class="flex items-center gap-1 rounded-full bg-orange-500/15 dark:bg-orange-500/10 px-2.5 py-1">
              <svg class="h-3 w-3 text-orange-600 dark:text-orange-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.362 5.214A8.252 8.252 0 0112 21 8.25 8.25 0 016.038 7.047 8.287 8.287 0 009 9.601a8.983 8.983 0 013.361-6.867 8.21 8.21 0 003 2.48z" /></svg>
              <span class="text-xs font-semibold text-orange-600 dark:text-orange-400">{exercise.calories_burned}</span>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  {#if successMsg}
    <div class="mb-3 flex items-center gap-2 rounded-xl border border-green-500/30 dark:border-green-500/20 bg-green-500/15 dark:bg-green-500/10 px-3.5 py-2.5">
      <svg class="h-4 w-4 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
      <span class="text-xs font-medium text-green-600 dark:text-green-400">{successMsg}</span>
    </div>
  {/if}

  <div class="flex items-center gap-2 mb-3">
    <div class="relative flex-1">
      <input
        type="text"
        bind:value={name}
        placeholder="Exercise name"
        onfocus={() => showPresets = true}
        class="w-full bg-secondary border border-border rounded-xl pl-3 pr-8 py-2.5 text-xs text-foreground/80 placeholder:text-muted-foreground focus:outline-none focus:border-orange-500/50 focus:ring-2 focus:ring-orange-500/20 transition-colors"
      />
      <button
        class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition"
        onclick={() => showPresets = !showPresets}
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.25 15L12 18.75 15.75 15m-7.5-6L12 5.25 15.75 9" /></svg>
      </button>
    </div>
    <input
      type="number"
      bind:value={duration}
      placeholder="min"
      min="1"
      bind:this={durationInput}
      class="w-16 bg-secondary border border-border rounded-xl px-3 py-2.5 text-xs text-foreground/80 placeholder:text-muted-foreground focus:outline-none focus:border-orange-500/50 transition-colors"
    />
    <button
      class="bg-orange-500 hover:bg-orange-400 disabled:opacity-40 text-foreground rounded-xl px-4 py-2.5 text-xs font-semibold transition-colors flex-shrink-0"
      onclick={logExercise}
      disabled={loading || !name.trim() || !duration}
    >
      {loading ? '...' : 'Log'}
    </button>
  </div>

  {#if showPresets}
    <div class="space-y-2.5">
      {#each exerciseCategories as category}
        <div>
          <div class="flex items-center gap-1.5 mb-1.5">
            {@html category.icon}
            <span class="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">{category.label}</span>
          </div>
          <div class="flex flex-wrap gap-1.5">
            {#each category.exercises as preset}
              <button
                class="bg-secondary border border-border hover:border-orange-500/40 dark:hover:border-orange-500/30 hover:bg-orange-500/10 dark:hover:bg-orange-500/5 rounded-lg px-2.5 py-1 text-[11px] text-foreground/80 hover:text-orange-600 dark:hover:text-orange-400 whitespace-nowrap transition-all"
                onclick={() => selectPreset(preset)}
              >
                {preset}
              </button>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
