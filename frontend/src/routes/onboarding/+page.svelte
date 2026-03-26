<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { authToken } from '$lib/stores';
  import Logo from '$components/Logo.svelte';

  let name = $state('');
  let age = $state('');
  let sex = $state<'male' | 'female' | ''>('');
  let height = $state('');
  let weight = $state('');
  let targetWeight = $state('');
  let activityLevel = $state('');
  let objective = $state('');
  let dietPlan = $state('');
  let fastingWindow = $state('');

  let currentStep = $state(1);
  let error = $state('');
  let loading = $state(false);
  let authenticated = $state(false);

  const steps = [
    { number: 1, label: 'Personal Info' },
    { number: 2, label: 'Goals' },
    { number: 3, label: 'Diet Plan' },
    { number: 4, label: 'Summary' }
  ];

  const activityOptions = [
    { value: 'sedentary', label: 'Sedentary', description: 'Little or no exercise' },
    { value: 'lightly_active', label: 'Lightly Active', description: '1-3 days/week' },
    { value: 'moderately_active', label: 'Moderately Active', description: '3-5 days/week' },
    { value: 'very_active', label: 'Very Active', description: '6-7 days/week' },
    { value: 'extremely_active', label: 'Extremely Active', description: 'Physical job or 2x/day training' }
  ];

  const objectiveOptions = [
    { value: 'lose_fat', label: 'Lose Fat', description: 'Calorie deficit to reduce body fat' },
    { value: 'feel_better', label: 'Feel Better', description: 'Small deficit for overall wellness' },
    { value: 'maintain', label: 'Maintain Weight', description: 'Keep your current physique' },
    { value: 'build_muscle', label: 'Build Muscle', description: 'Calorie surplus to gain muscle mass' }
  ];

  const dietPlanOptions = [
    { value: 'calorie_deficit', label: 'Calorie Deficit', description: '40C / 30P / 30F' },
    { value: 'keto', label: 'Keto', description: '5C / 25P / 70F' },
    { value: 'intermittent_fasting', label: 'Intermittent Fasting', description: '40C / 30P / 30F' },
    { value: 'paleo', label: 'Paleo', description: '25C / 35P / 40F' },
    { value: 'mediterranean', label: 'Mediterranean', description: '45C / 25P / 30F' }
  ];

  const fastingOptions = [
    { value: '16:8', label: '16:8' },
    { value: '18:6', label: '18:6' },
    { value: '20:4', label: '20:4' },
    { value: 'omad', label: 'OMAD' }
  ];

  let step1Valid = $derived(
    name.trim() !== '' &&
    Number(age) >= 1 && Number(age) <= 150 &&
    sex !== '' &&
    Number(height) > 0 &&
    Number(weight) > 0
  );

  let step2Valid = $derived(
    Number(targetWeight) > 0 &&
    activityLevel !== '' &&
    objective !== ''
  );

  let step3Valid = $derived(
    dietPlan !== '' &&
    (dietPlan !== 'intermittent_fasting' || fastingWindow !== '')
  );

  let canProceed = $derived(
    currentStep === 1 ? step1Valid :
    currentStep === 2 ? step2Valid :
    currentStep === 3 ? step3Valid :
    true
  );

  let activityMultiplier = $derived(
    activityLevel === 'sedentary' ? 1.2 :
    activityLevel === 'lightly_active' ? 1.375 :
    activityLevel === 'moderately_active' ? 1.55 :
    activityLevel === 'very_active' ? 1.725 :
    activityLevel === 'extremely_active' ? 1.9 :
    1.2
  );

  let objectiveMultiplier = $derived(
    objective === 'lose_fat' ? 0.8 :
    objective === 'feel_better' ? 0.9 :
    objective === 'maintain' ? 1.0 :
    objective === 'build_muscle' ? 1.1 :
    1.0
  );

  let bmr = $derived(() => {
    const w = Number(weight);
    const h = Number(height);
    const a = Number(age);
    if (!w || !h || !a) return 0;
    if (sex === 'male') return 10 * w + 6.25 * h - 5 * a + 5;
    return 10 * w + 6.25 * h - 5 * a - 161;
  });

  let tdee = $derived(() => Math.round(bmr() * activityMultiplier * objectiveMultiplier));

  let selectedActivityLabel = $derived(
    activityOptions.find(o => o.value === activityLevel)?.label ?? ''
  );

  let selectedObjectiveLabel = $derived(
    objectiveOptions.find(o => o.value === objective)?.label ?? ''
  );

  let selectedDietPlanLabel = $derived(
    dietPlanOptions.find(o => o.value === dietPlan)?.label ?? ''
  );

  function nextStep() {
    if (currentStep < 4) {
      currentStep++;
      error = '';
    }
  }

  function prevStep() {
    if (currentStep > 1) {
      currentStep--;
      error = '';
    }
  }

  async function handleSubmit() {
    error = '';
    loading = true;
    try {
      await api.post('/user/onboarding', {
        name: name.trim(),
        age: Number(age),
        sex,
        height: Number(height),
        weight: Number(weight),
        target_weight: Number(targetWeight),
        activity_level: activityLevel,
        objective,
        diet_plan: dietPlan,
        fasting_window: dietPlan === 'intermittent_fasting' ? fastingWindow : null
      });
      goto('/dashboard');
    } catch (err) {
      error = err instanceof Error ? err.message : 'Onboarding failed';
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    const unsub = authToken.subscribe((token) => {
      if (!token) {
        goto('/login');
        return;
      }
      authenticated = true;
    });
    return unsub;
  });
</script>

{#if !authenticated}
  <div class="flex h-screen items-center justify-center bg-slate-950">
    <div class="h-8 w-8 animate-spin rounded-full border-2 border-joule-500 border-t-transparent"></div>
  </div>
{:else}
  <div class="flex min-h-screen flex-col bg-slate-950">
    <div class="flex flex-col items-center px-6 pt-8 pb-4">
      <div class="flex items-center gap-3">
        <Logo size={36} />
        <span class="text-xl font-bold text-white">Joule</span>
      </div>
    </div>

    <div class="mx-auto w-full max-w-lg flex-1 px-6 pb-8">
      <div class="mb-8 flex items-center justify-center gap-2">
        {#each steps as step, i}
          {@const isActive = currentStep === step.number}
          {@const isCompleted = currentStep > step.number}
          <div class="flex items-center gap-2">
            <div class="flex flex-col items-center gap-1.5">
              <div
                class="flex h-8 w-8 items-center justify-center rounded-full text-xs font-semibold transition-all {isActive ? 'bg-joule-500 text-slate-900' : isCompleted ? 'bg-joule-500/20 text-joule-400' : 'bg-slate-800 text-slate-500'}"
              >
                {isCompleted ? '✓' : step.number}
              </div>
              <span class="text-xs {isActive ? 'text-joule-400' : 'text-slate-500'}">{step.label}</span>
            </div>
            {#if i < steps.length - 1}
              <div class="mb-5 h-px w-8 {isCompleted ? 'bg-joule-500/40' : 'bg-slate-800'}"></div>
            {/if}
          </div>
        {/each}
      </div>

      {#if currentStep === 1}
        <div>
          <h2 class="text-2xl font-bold text-white">Personal Info</h2>
          <p class="mt-1 text-sm text-slate-400">Tell us about yourself to personalize your plan</p>

          <div class="mt-8 space-y-5">
            <div>
              <label for="name" class="mb-1.5 block text-sm font-medium text-slate-300">Name</label>
              <input
                id="name"
                type="text"
                bind:value={name}
                required
                placeholder="Your name"
                class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
              />
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="age" class="mb-1.5 block text-sm font-medium text-slate-300">Age</label>
                <input
                  id="age"
                  type="number"
                  bind:value={age}
                  min="1"
                  max="150"
                  required
                  placeholder="25"
                  class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                />
              </div>
              <div>
                <span class="mb-1.5 block text-sm font-medium text-slate-300">Sex</span>
                <div class="flex gap-2">
                  <button
                    type="button"
                    onclick={() => (sex = 'male')}
                    class="flex-1 rounded-lg border px-3.5 py-2.5 text-sm font-medium transition {sex === 'male' ? 'border-joule-500 bg-joule-500/10 text-joule-400' : 'border-slate-700 text-slate-400 hover:border-slate-600'}"
                  >
                    Male
                  </button>
                  <button
                    type="button"
                    onclick={() => (sex = 'female')}
                    class="flex-1 rounded-lg border px-3.5 py-2.5 text-sm font-medium transition {sex === 'female' ? 'border-joule-500 bg-joule-500/10 text-joule-400' : 'border-slate-700 text-slate-400 hover:border-slate-600'}"
                  >
                    Female
                  </button>
                </div>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="height" class="mb-1.5 block text-sm font-medium text-slate-300">Height (cm)</label>
                <input
                  id="height"
                  type="number"
                  bind:value={height}
                  min="0"
                  step="0.1"
                  required
                  placeholder="175"
                  class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                />
              </div>
              <div>
                <label for="weight" class="mb-1.5 block text-sm font-medium text-slate-300">Weight (kg)</label>
                <input
                  id="weight"
                  type="number"
                  bind:value={weight}
                  min="0"
                  step="0.1"
                  required
                  placeholder="70"
                  class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                />
              </div>
            </div>
          </div>
        </div>
      {/if}

      {#if currentStep === 2}
        <div>
          <h2 class="text-2xl font-bold text-white">Goals</h2>
          <p class="mt-1 text-sm text-slate-400">Set your targets and activity level</p>

          <div class="mt-8 space-y-6">
            <div>
              <label for="target-weight" class="mb-1.5 block text-sm font-medium text-slate-300">Target Weight (kg)</label>
              <input
                id="target-weight"
                type="number"
                bind:value={targetWeight}
                min="0"
                step="0.1"
                required
                placeholder="65"
                class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
              />
            </div>

            <div>
              <span class="mb-2 block text-sm font-medium text-slate-300">Activity Level</span>
              <div class="grid grid-cols-1 gap-3">
                {#each activityOptions as option}
                  <button
                    type="button"
                    onclick={() => (activityLevel = option.value)}
                    class="rounded-xl border p-4 text-left transition {activityLevel === option.value ? 'border-joule-500 bg-joule-500/10' : 'border-slate-700 hover:border-slate-600'}"
                  >
                    <div class="text-sm font-medium text-white">{option.label}</div>
                    <div class="mt-1 text-xs text-slate-400">{option.description}</div>
                  </button>
                {/each}
              </div>
            </div>

            <div>
              <span class="mb-2 block text-sm font-medium text-slate-300">Objective</span>
              <div class="grid grid-cols-2 gap-3">
                {#each objectiveOptions as option}
                  <button
                    type="button"
                    onclick={() => (objective = option.value)}
                    class="rounded-xl border p-4 text-left transition {objective === option.value ? 'border-joule-500 bg-joule-500/10' : 'border-slate-700 hover:border-slate-600'}"
                  >
                    <div class="text-sm font-medium text-white">{option.label}</div>
                    <div class="mt-1 text-xs text-slate-400">{option.description}</div>
                  </button>
                {/each}
              </div>
            </div>
          </div>
        </div>
      {/if}

      {#if currentStep === 3}
        <div>
          <h2 class="text-2xl font-bold text-white">Diet Plan</h2>
          <p class="mt-1 text-sm text-slate-400">Choose a diet plan that works for you</p>

          <div class="mt-8 space-y-6">
            <div>
              <span class="mb-2 block text-sm font-medium text-slate-300">Diet Plan</span>
              <div class="grid grid-cols-1 gap-3">
                {#each dietPlanOptions as option}
                  <button
                    type="button"
                    onclick={() => (dietPlan = option.value)}
                    class="rounded-xl border p-4 text-left transition {dietPlan === option.value ? 'border-joule-500 bg-joule-500/10' : 'border-slate-700 hover:border-slate-600'}"
                  >
                    <div class="text-sm font-medium text-white">{option.label}</div>
                    <div class="mt-1 text-xs text-slate-400">{option.description}</div>
                  </button>
                {/each}
              </div>
            </div>

            {#if dietPlan === 'intermittent_fasting'}
              <div>
                <span class="mb-2 block text-sm font-medium text-slate-300">Fasting Window</span>
                <div class="flex gap-2">
                  {#each fastingOptions as option}
                    <button
                      type="button"
                      onclick={() => (fastingWindow = option.value)}
                      class="flex-1 rounded-lg border px-3.5 py-2.5 text-center text-sm font-medium transition {fastingWindow === option.value ? 'border-joule-500 bg-joule-500/10 text-joule-400' : 'border-slate-700 text-slate-400 hover:border-slate-600'}"
                    >
                      {option.label}
                    </button>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/if}

      {#if currentStep === 4}
        <div>
          <h2 class="text-2xl font-bold text-white">Summary</h2>
          <p class="mt-1 text-sm text-slate-400">Review your profile before we start</p>

          <div class="mt-8 space-y-6">
            <div class="rounded-xl border border-slate-700 bg-surface-light p-5">
              <h3 class="mb-3 text-sm font-semibold text-joule-400">Personal Info</h3>
              <div class="grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
                <div>
                  <span class="text-slate-500">Name</span>
                  <p class="text-white">{name}</p>
                </div>
                <div>
                  <span class="text-slate-500">Age</span>
                  <p class="text-white">{age}</p>
                </div>
                <div>
                  <span class="text-slate-500">Sex</span>
                  <p class="text-white capitalize">{sex}</p>
                </div>
                <div>
                  <span class="text-slate-500">Height</span>
                  <p class="text-white">{height} cm</p>
                </div>
                <div>
                  <span class="text-slate-500">Weight</span>
                  <p class="text-white">{weight} kg</p>
                </div>
              </div>
            </div>

            <div class="rounded-xl border border-slate-700 bg-surface-light p-5">
              <h3 class="mb-3 text-sm font-semibold text-joule-400">Goals</h3>
              <div class="grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
                <div>
                  <span class="text-slate-500">Target Weight</span>
                  <p class="text-white">{targetWeight} kg</p>
                </div>
                <div>
                  <span class="text-slate-500">Activity</span>
                  <p class="text-white">{selectedActivityLabel}</p>
                </div>
                <div>
                  <span class="text-slate-500">Objective</span>
                  <p class="text-white">{selectedObjectiveLabel}</p>
                </div>
              </div>
            </div>

            <div class="rounded-xl border border-slate-700 bg-surface-light p-5">
              <h3 class="mb-3 text-sm font-semibold text-joule-400">Diet Plan</h3>
              <div class="grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
                <div>
                  <span class="text-slate-500">Plan</span>
                  <p class="text-white">{selectedDietPlanLabel}</p>
                </div>
                {#if dietPlan === 'intermittent_fasting'}
                  <div>
                    <span class="text-slate-500">Fasting Window</span>
                    <p class="text-white">{fastingWindow}</p>
                  </div>
                {/if}
              </div>
            </div>

            <div class="rounded-xl border border-joule-500/20 bg-joule-500/5 p-5">
              <h3 class="mb-2 text-sm font-semibold text-joule-400">Estimated Daily Target</h3>
              <div class="text-3xl font-bold text-white">{tdee()} <span class="text-lg font-normal text-slate-400">kcal</span></div>
              <p class="mt-1 text-xs text-slate-500">Based on Mifflin-St Jeor equation. Final values calculated server-side.</p>
            </div>
          </div>
        </div>
      {/if}

      {#if error}
        <div class="mt-6 rounded-lg border border-red-500/20 bg-red-500/10 px-3.5 py-2.5 text-sm text-red-400">
          {error}
        </div>
      {/if}

      <div class="mt-8 flex items-center gap-3">
        {#if currentStep > 1}
          <button
            type="button"
            onclick={prevStep}
            class="rounded-lg border border-slate-700 px-5 py-2.5 text-sm font-medium text-slate-400 transition hover:border-slate-600 hover:text-white"
          >
            Back
          </button>
        {/if}
        {#if currentStep < 4}
          <button
            type="button"
            onclick={nextStep}
            disabled={!canProceed}
            class="flex-1 rounded-lg bg-joule-500 px-5 py-2.5 text-sm font-semibold text-slate-900 transition hover:bg-joule-400 focus:outline-none focus:ring-2 focus:ring-joule-500 focus:ring-offset-2 focus:ring-offset-slate-900 disabled:cursor-not-allowed disabled:opacity-50"
          >
            Next
          </button>
        {:else}
          <button
            type="button"
            onclick={handleSubmit}
            disabled={loading}
            class="flex-1 rounded-lg bg-joule-500 px-5 py-2.5 text-sm font-semibold text-slate-900 transition hover:bg-joule-400 focus:outline-none focus:ring-2 focus:ring-joule-500 focus:ring-offset-2 focus:ring-offset-slate-900 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? 'Setting up...' : 'Start Tracking'}
          </button>
        {/if}
      </div>
    </div>
  </div>
{/if}
