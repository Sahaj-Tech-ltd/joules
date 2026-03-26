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
  let durationInput: HTMLInputElement | undefined = $state(undefined);

  const presets = ['Walking', 'Running', 'Cycling', 'Swimming', 'Weight Training', 'Yoga', 'HIIT', 'Jump Rope'];

  function dispatchLogged() {
    const event = new CustomEvent('exercise-logged');
    window.dispatchEvent(event);
  }

  function formatTime(ts: string): string {
    return new Date(ts).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
  }

  function selectPreset(exerciseName: string) {
    name = exerciseName;
    durationInput?.focus();
  }

  async function logExercise() {
    const dur = parseInt(duration, 10);
    if (loading || !name.trim() || isNaN(dur) || dur <= 0) return;
    loading = true;
    try {
      await api.post('/api/exercises', { name: name.trim(), duration_min: dur });
      name = '';
      duration = '';
      dispatchLogged();
    } catch {
    } finally {
      loading = false;
    }
  }
</script>

<div class="rounded-xl border border-slate-800 bg-surface-light p-6">
  <div class="flex items-center gap-2 mb-4">
    <svg class="w-4 h-4 text-orange-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" d="M15.362 5.214A8.252 8.252 0 0112 21 8.25 8.25 0 016.038 7.047 8.287 8.287 0 009 9.601a8.983 8.983 0 013.361-6.867 8.21 8.21 0 003 2.48z" />
      <path stroke-linecap="round" stroke-linejoin="round" d="M12 18a3.75 3.75 0 00.495-7.468 5.99 5.99 0 00-1.925 3.547 5.975 5.975 0 01-2.133-1.001A3.75 3.75 0 0012 18z" />
    </svg>
    <h3 class="text-sm font-semibold text-slate-100">Exercise</h3>
  </div>

  {#if exercises.length === 0}
    <p class="text-sm text-slate-500 mb-4">No exercises logged today</p>
  {:else}
    <div class="flex flex-col mb-4">
      {#each exercises as exercise, i}
        <div class="flex items-center justify-between gap-3 py-2.5 {i < exercises.length - 1 ? 'border-b border-slate-800' : ''}">
          <div class="min-w-0">
            <p class="text-sm font-medium text-slate-100 truncate">{exercise.name}</p>
            <p class="text-xs text-slate-400">{formatTime(exercise.timestamp)}</p>
          </div>
          <span class="text-xs text-slate-300 flex-shrink-0">{exercise.duration_min} min &middot; {exercise.calories_burned} kcal</span>
        </div>
      {/each}
    </div>
  {/if}

  <div class="flex items-center gap-1.5 overflow-x-auto pb-1 mb-3 scrollbar-hide">
    {#each presets as preset}
      <button
        class="bg-surface border border-slate-700 hover:bg-surface-lighter rounded-lg px-2.5 py-1 text-[11px] text-slate-300 whitespace-nowrap transition-colors flex-shrink-0"
        onclick={() => selectPreset(preset)}
      >
        {preset}
      </button>
    {/each}
  </div>

  <div class="flex items-center gap-2">
    <input
      type="text"
      bind:value={name}
      placeholder="Exercise"
      class="flex-1 min-w-0 bg-surface border border-slate-700 rounded-lg px-3 py-1.5 text-xs text-slate-200 placeholder:text-slate-500 focus:outline-none focus:border-orange-500 transition-colors"
    />
    <input
      type="number"
      bind:value={duration}
      placeholder="min"
      min="1"
      bind:this={durationInput}
      class="w-16 bg-surface border border-slate-700 rounded-lg px-3 py-1.5 text-xs text-slate-200 placeholder:text-slate-500 focus:outline-none focus:border-orange-500 transition-colors"
    />
    <button
      class="bg-orange-500 hover:bg-orange-600 disabled:bg-slate-700 disabled:text-slate-500 text-white rounded-lg px-3 py-1.5 text-xs font-medium transition-colors flex-shrink-0"
      onclick={logExercise}
      disabled={loading || !name.trim() || !duration}
    >
      Log
    </button>
  </div>
</div>
