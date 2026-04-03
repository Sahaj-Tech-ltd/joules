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
  let eatingWindowStart = $state('12:00');
  let dietType = $state('omnivore');
  let allergies = $state<string[]>([]);
  let foodNotes = $state('');
  let eatingContext = $state('');

  let currentStep = $state(1);
  let error = $state('');
  let loading = $state(false);
  let authenticated = $state(false);

  const steps = [
    { number: 1, label: 'Personal Info' },
    { number: 2, label: 'Goals' },
    { number: 3, label: 'Diet Plan' },
    { number: 4, label: 'Summary' },
    { number: 5, label: 'Preferences' }
  ];

  const dietTypeOptions = [
    { value: 'omnivore', label: 'Omnivore' },
    { value: 'vegetarian', label: 'Vegetarian' },
    { value: 'vegan', label: 'Vegan' },
    { value: 'pescatarian', label: 'Pescatarian' },
    { value: 'halal', label: 'Halal' },
    { value: 'kosher', label: 'Kosher' },
    { value: 'gluten_free', label: 'Gluten Free' }
  ];

  const allergyOptions = [
    { value: 'nuts', label: 'Nuts' },
    { value: 'dairy', label: 'Dairy' },
    { value: 'gluten', label: 'Gluten' },
    { value: 'shellfish', label: 'Shellfish' },
    { value: 'soy', label: 'Soy' },
    { value: 'eggs', label: 'Eggs' }
  ];

  const activityOptions = [
    { value: 'sedentary', label: 'Sedentary', description: 'Little or no exercise' },
    { value: 'light', label: 'Lightly Active', description: '1-3 days/week' },
    { value: 'moderate', label: 'Moderately Active', description: '3-5 days/week' },
    { value: 'active', label: 'Very Active', description: '6-7 days/week' },
    { value: 'very_active', label: 'Extremely Active', description: 'Physical job or 2x/day training' }
  ];

  const objectiveOptions = [
    { value: 'cut_fat', label: 'Lose Fat', description: 'Calorie deficit to reduce body fat' },
    { value: 'feel_better', label: 'Feel Better', description: 'Small deficit for overall wellness' },
    { value: 'maintain', label: 'Maintain Weight', description: 'Keep your current physique' },
    { value: 'build_muscle', label: 'Build Muscle', description: 'Calorie surplus to gain muscle mass' }
  ];

  const dietPlanOptions = [
    {
      value: 'balanced',
      label: 'Just Track & Stay Healthy',
      description: '50% carbs · 25% protein · 25% fat',
      bestFor: 'Best for: anyone who just wants to track calories and eat well',
      tip: 'No special rules — eat what you like, track what you eat. Your AI coach will guide you based on your habits.'
    },
    {
      value: 'calorie_deficit',
      label: 'Calorie Deficit',
      description: '40% carbs · 30% protein · 30% fat',
      bestFor: 'Best for: general weight loss',
      tip: 'Eat less than you burn — no foods are banned. Great starting point for most people.'
    },
    {
      value: 'keto',
      label: 'Keto',
      description: '5% carbs · 25% protein · 70% fat',
      bestFor: 'Best for: fast fat loss, low-carb lovers',
      tip: 'Drastically cuts carbs (bread, pasta, sugar, rice). You eat mostly meat, eggs, cheese, nuts, and avocado.'
    },
    {
      value: 'intermittent_fasting',
      label: 'Intermittent Fasting',
      description: '40% carbs · 30% protein · 30% fat',
      bestFor: 'Best for: people who prefer skipping breakfast',
      tip: 'You only eat within a certain window each day (e.g. 12pm–8pm). No food restriction — just time restriction.'
    },
    {
      value: 'paleo',
      label: 'Paleo',
      description: '25% carbs · 35% protein · 40% fat',
      bestFor: 'Best for: whole-food eaters, grain-free',
      tip: 'Eat like our ancestors — meat, fish, eggs, vegetables, fruits, nuts. Avoid grains, dairy, and processed food.'
    },
    {
      value: 'mediterranean',
      label: 'Mediterranean',
      description: '45% carbs · 25% protein · 30% fat',
      bestFor: 'Best for: heart health, sustainable long-term eating',
      tip: 'Olive oil, fish, whole grains, legumes, vegetables. Moderate wine. Minimal red meat.'
    }
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

  function toggleAllergy(value: string) {
    if (allergies.includes(value)) {
      allergies = allergies.filter(a => a !== value);
    } else {
      allergies = [...allergies, value];
    }
  }

  let activityMultiplier = $derived(
    activityLevel === 'sedentary' ? 1.2 :
    activityLevel === 'light' ? 1.375 :
    activityLevel === 'moderate' ? 1.55 :
    activityLevel === 'active' ? 1.725 :
    activityLevel === 'very_active' ? 1.9 :
    1.2
  );

  let objectiveMultiplier = $derived(
    objective === 'cut_fat' ? 0.8 :
    objective === 'feel_better' ? 0.9 :
    objective === 'maintain' ? 1.0 :
    objective === 'build_muscle' ? 1.1 :
    1.0
  );

  let bmr = $derived.by(() => {
    const w = Number(weight);
    const h = Number(height);
    const a = Number(age);
    if (!w || !h || !a) return 0;
    if (sex === 'male') return 10 * w + 6.25 * h - 5 * a + 5;
    return 10 * w + 6.25 * h - 5 * a - 161;
  });

  let tdee = $derived(Math.round(bmr * activityMultiplier * objectiveMultiplier));

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
    if (currentStep < 5) {
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
        height_cm: Number(height),
        weight_kg: Number(weight),
        target_weight_kg: Number(targetWeight),
        activity_level: activityLevel,
        objective,
        diet_plan: dietPlan,
        fasting_window: dietPlan === 'intermittent_fasting' ? fastingWindow : null,
        eating_window_start: dietPlan === 'intermittent_fasting' ? eatingWindowStart : null
      });
      // Save dietary preferences (best-effort)
      try {
        await api.put('/user/preferences', {
          diet_type: dietType,
          allergies,
          food_notes: foodNotes,
          eating_context: eatingContext
        });
      } catch {}
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
  <div class="flex h-screen items-center justify-center bg-background">
    <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
  </div>
{:else}
  <div class="flex min-h-screen flex-col bg-background">
    <div class="flex flex-col items-center px-6 pt-8 pb-4">
      <div class="flex items-center gap-3">
        <Logo size={36} />
        <span class="text-xl font-bold text-foreground">Joules</span>
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
                class="flex h-8 w-8 items-center justify-center rounded-full text-xs font-semibold transition-all {isActive ? 'bg-primary text-primary-foreground' : isCompleted ? 'bg-primary/20 text-primary' : 'bg-accent text-muted-foreground'}"
              >
                {isCompleted ? '✓' : step.number}
              </div>
              <span class="text-xs {isActive ? 'text-primary' : 'text-muted-foreground'}">{step.label}</span>
            </div>
            {#if i < steps.length - 1}
              <div class="mb-5 h-px w-8 {isCompleted ? 'bg-primary/40' : 'bg-accent'}"></div>
            {/if}
          </div>
        {/each}
      </div>

      {#if currentStep === 1}
        <div>
          <h2 class="text-2xl font-bold text-foreground">Personal Info</h2>
          <p class="mt-1 text-sm text-foreground">Tell us about yourself to personalize your plan</p>

          <div class="mt-8 space-y-5">
            <div>
              <label for="name" class="mb-1.5 block text-sm font-medium text-foreground">Name</label>
              <input
                id="name"
                type="text"
                bind:value={name}
                required
                placeholder="Your name"
                class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
              />
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="age" class="mb-1.5 block text-sm font-medium text-foreground">Age</label>
                <input
                  id="age"
                  type="number"
                  bind:value={age}
                  min="1"
                  max="150"
                  required
                  placeholder="25"
                  class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
                />
              </div>
              <div>
                <span class="mb-1.5 block text-sm font-medium text-foreground">Sex</span>
                <div class="flex gap-2">
                  <button
                    type="button"
                    onclick={() => (sex = 'male')}
                    class="flex-1 rounded-lg border px-3.5 py-2.5 text-sm font-medium transition {sex === 'male' ? 'border-primary bg-primary/10 text-primary' : 'border-border text-foreground hover:border-border'}"
                  >
                    Male
                  </button>
                  <button
                    type="button"
                    onclick={() => (sex = 'female')}
                    class="flex-1 rounded-lg border px-3.5 py-2.5 text-sm font-medium transition {sex === 'female' ? 'border-primary bg-primary/10 text-primary' : 'border-border text-foreground hover:border-border'}"
                  >
                    Female
                  </button>
                </div>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="height" class="mb-1.5 block text-sm font-medium text-foreground">Height (cm)</label>
                <input
                  id="height"
                  type="number"
                  bind:value={height}
                  min="0"
                  step="0.1"
                  required
                  placeholder="175"
                  class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
                />
              </div>
              <div>
                <label for="weight" class="mb-1.5 block text-sm font-medium text-foreground">Weight (kg)</label>
                <input
                  id="weight"
                  type="number"
                  bind:value={weight}
                  min="0"
                  step="0.1"
                  required
                  placeholder="70"
                  class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
                />
              </div>
            </div>
          </div>
        </div>
      {/if}

      {#if currentStep === 2}
        <div>
          <h2 class="text-2xl font-bold text-foreground">Goals</h2>
          <p class="mt-1 text-sm text-foreground">Set your targets and activity level</p>

          <div class="mt-8 space-y-6">
            <div>
              <label for="target-weight" class="mb-1.5 block text-sm font-medium text-foreground">Target Weight (kg)</label>
              <input
                id="target-weight"
                type="number"
                bind:value={targetWeight}
                min="0"
                step="0.1"
                required
                placeholder="65"
                class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
              />
            </div>

            <div>
              <span class="mb-2 block text-sm font-medium text-foreground">Activity Level</span>
              <div class="grid grid-cols-1 gap-3">
                {#each activityOptions as option}
                  <button
                    type="button"
                    onclick={() => (activityLevel = option.value)}
                    class="rounded-xl border p-4 text-left transition {activityLevel === option.value ? 'border-primary bg-primary/10' : 'border-border hover:border-border'}"
                  >
                    <div class="text-sm font-medium text-foreground">{option.label}</div>
                    <div class="mt-1 text-xs text-foreground">{option.description}</div>
                  </button>
                {/each}
              </div>
            </div>

            <div>
              <span class="mb-2 block text-sm font-medium text-foreground">Objective</span>
              <div class="grid grid-cols-2 gap-3">
                {#each objectiveOptions as option}
                  <button
                    type="button"
                    onclick={() => (objective = option.value)}
                    class="rounded-xl border p-4 text-left transition {objective === option.value ? 'border-primary bg-primary/10' : 'border-border hover:border-border'}"
                  >
                    <div class="text-sm font-medium text-foreground">{option.label}</div>
                    <div class="mt-1 text-xs text-foreground">{option.description}</div>
                  </button>
                {/each}
              </div>
            </div>
          </div>
        </div>
      {/if}

      {#if currentStep === 3}
        <div>
          <h2 class="text-2xl font-bold text-foreground">Diet Plan</h2>
          <p class="mt-1 text-sm text-foreground">Choose a diet plan that works for you</p>

          <div class="mt-8 space-y-6">
            <div>
              <span class="mb-2 block text-sm font-medium text-foreground">Diet Plan</span>
              <div class="grid grid-cols-1 gap-3">
                {#each dietPlanOptions as option}
                  <button
                    type="button"
                    onclick={() => (dietPlan = option.value)}
                    class="rounded-xl border p-4 text-left transition {dietPlan === option.value ? 'border-primary bg-primary/10' : 'border-border hover:border-border'}"
                  >
                    <div class="flex items-center justify-between">
                      <div class="text-sm font-medium text-foreground">{option.label}</div>
                      <div class="text-xs text-muted-foreground">{option.description}</div>
                    </div>
                    <div class="mt-1 text-xs font-medium text-primary">{option.bestFor}</div>
                    <div class="mt-1 text-xs text-foreground">{option.tip}</div>
                  </button>
                {/each}
              </div>
            </div>

            {#if dietPlan === 'intermittent_fasting'}
              <div>
                <span class="mb-2 block text-sm font-medium text-foreground">Fasting Window</span>
                <div class="flex gap-2">
                  {#each fastingOptions as option}
                    <button
                      type="button"
                      onclick={() => (fastingWindow = option.value)}
                      class="flex-1 rounded-lg border px-3.5 py-2.5 text-center text-sm font-medium transition {fastingWindow === option.value ? 'border-primary bg-primary/10 text-primary' : 'border-border text-foreground hover:border-border'}"
                    >
                      {option.label}
                    </button>
                  {/each}
                </div>
              </div>
              <div>
                <label for="eating-window-start" class="mb-1.5 block text-sm font-medium text-foreground">Eating window starts at</label>
                <input
                  id="eating-window-start"
                  type="time"
                  bind:value={eatingWindowStart}
                  class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
                />
                {#if fastingWindow}
                  {@const hours = fastingWindow === 'omad' ? 1 : fastingWindow === '20:4' ? 4 : fastingWindow === '18:6' ? 6 : 8}
                  {@const [startH, startM] = eatingWindowStart.split(':').map(Number)}
                  {@const endH = (startH + hours) % 24}
                  <p class="mt-1.5 text-xs text-muted-foreground">
                    Eating {eatingWindowStart} – {String(endH).padStart(2, '0')}:{String(startM).padStart(2, '0')} · Fasting {24 - hours}h
                  </p>
                {/if}
              </div>
            {/if}
          </div>
        </div>
      {/if}

      {#if currentStep === 4}
        <div>
          <h2 class="text-2xl font-bold text-foreground">Summary</h2>
          <p class="mt-1 text-sm text-foreground">Review your profile before we start</p>

          <div class="mt-8 space-y-6">
            <div class="rounded-xl border border-border bg-card p-5">
              <h3 class="mb-3 text-sm font-semibold text-primary">Personal Info</h3>
              <div class="grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
                <div>
                  <span class="text-muted-foreground">Name</span>
                  <p class="text-foreground">{name}</p>
                </div>
                <div>
                  <span class="text-muted-foreground">Age</span>
                  <p class="text-foreground">{age}</p>
                </div>
                <div>
                  <span class="text-muted-foreground">Sex</span>
                  <p class="text-foreground capitalize">{sex}</p>
                </div>
                <div>
                  <span class="text-muted-foreground">Height</span>
                  <p class="text-foreground">{height} cm</p>
                </div>
                <div>
                  <span class="text-muted-foreground">Weight</span>
                  <p class="text-foreground">{weight} kg</p>
                </div>
              </div>
            </div>

            <div class="rounded-xl border border-border bg-card p-5">
              <h3 class="mb-3 text-sm font-semibold text-primary">Goals</h3>
              <div class="grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
                <div>
                  <span class="text-muted-foreground">Target Weight</span>
                  <p class="text-foreground">{targetWeight} kg</p>
                </div>
                <div>
                  <span class="text-muted-foreground">Activity</span>
                  <p class="text-foreground">{selectedActivityLabel}</p>
                </div>
                <div>
                  <span class="text-muted-foreground">Objective</span>
                  <p class="text-foreground">{selectedObjectiveLabel}</p>
                </div>
              </div>
            </div>

            <div class="rounded-xl border border-border bg-card p-5">
              <h3 class="mb-3 text-sm font-semibold text-primary">Diet Plan</h3>
              <div class="grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
                <div>
                  <span class="text-muted-foreground">Plan</span>
                  <p class="text-foreground">{selectedDietPlanLabel}</p>
                </div>
                {#if dietPlan === 'intermittent_fasting'}
                  <div>
                    <span class="text-muted-foreground">Fasting Window</span>
                    <p class="text-foreground">{fastingWindow}</p>
                  </div>
                  <div>
                    <span class="text-muted-foreground">Eating Starts</span>
                    <p class="text-foreground">{eatingWindowStart}</p>
                  </div>
                {/if}
              </div>
            </div>

            <div class="rounded-xl border border-primary/20 bg-primary/5 p-5">
              <h3 class="mb-2 text-sm font-semibold text-primary">Estimated Daily Target</h3>
              <div class="text-3xl font-bold text-foreground">{tdee} <span class="text-lg font-normal text-foreground">kcal</span></div>
              <p class="mt-1 text-xs text-muted-foreground">Based on Mifflin-St Jeor equation. Final values calculated server-side.</p>
            </div>
          </div>
        </div>
      {/if}

      {#if currentStep === 5}
        <div>
          <h2 class="text-2xl font-bold text-foreground">Your Food Preferences</h2>
          <p class="mt-1 text-sm text-foreground">Optional — helps your AI coach give better advice</p>

          <div class="mt-8 space-y-6">
            <div>
              <span class="mb-2 block text-sm font-medium text-foreground">Diet Type</span>
              <div class="flex flex-wrap gap-2">
                {#each dietTypeOptions as option}
                  <button
                    type="button"
                    onclick={() => (dietType = option.value)}
                    class="rounded-lg border px-3.5 py-2 text-sm font-medium transition {dietType === option.value ? 'border-primary bg-primary/10 text-primary' : 'border-border text-foreground hover:border-border'}"
                  >
                    {option.label}
                  </button>
                {/each}
              </div>
            </div>

            <div>
              <span class="mb-2 block text-sm font-medium text-foreground">Allergies / Intolerances</span>
              <div class="flex flex-wrap gap-2">
                {#each allergyOptions as option}
                  <button
                    type="button"
                    onclick={() => toggleAllergy(option.value)}
                    class="rounded-lg border px-3.5 py-2 text-sm font-medium transition {allergies.includes(option.value) ? 'border-primary bg-primary/10 text-primary' : 'border-border text-foreground hover:border-border'}"
                  >
                    {option.label}
                  </button>
                {/each}
              </div>
            </div>

            <div>
              <label for="food-notes" class="mb-1.5 block text-sm font-medium text-foreground">Tell us about your food preferences</label>
              <textarea
                id="food-notes"
                bind:value={foodNotes}
                rows="3"
                placeholder="e.g. I love pizza, usually eat veggie burgers on weekdays"
                class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring resize-none"
              ></textarea>
            </div>

            <div>
              <label for="eating-context" class="mb-1.5 block text-sm font-medium text-foreground">Where / how do you usually eat?</label>
              <textarea
                id="eating-context"
                bind:value={eatingContext}
                rows="3"
                placeholder="e.g. mostly home-cooked, sometimes fast food at work"
                class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring resize-none"
              ></textarea>
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
            class="rounded-lg border border-border px-5 py-2.5 text-sm font-medium text-foreground transition hover:border-border hover:text-foreground"
          >
            Back
          </button>
        {/if}
        {#if currentStep < 5}
          <button
            type="button"
            onclick={nextStep}
            disabled={!canProceed}
            class="flex-1 rounded-lg bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground transition hover:bg-primary/80 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50"
          >
            Next
          </button>
        {:else}
          <button
            type="button"
            onclick={handleSubmit}
            disabled={loading}
            class="flex-1 rounded-lg bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground transition hover:bg-primary/80 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? 'Setting up...' : 'Start Tracking'}
          </button>
        {/if}
      </div>
    </div>
  </div>
{/if}
