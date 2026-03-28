<script lang="ts">
  import Sidebar from '$components/Sidebar.svelte';
  import ThemeToggle from '$components/ThemeToggle.svelte';
  import { authToken, userGoals } from '$lib/stores';
  import { api } from '$lib/api';
  import { get } from 'svelte/store';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { defaultUnits, type UnitPrefs } from '$lib/units';

  interface Goals {
    objective: string;
    diet_plan: string;
    fasting_window: string | null;
    daily_calorie_target: number;
    daily_protein_g: number;
    daily_carbs_g: number;
    daily_fat_g: number;
  }

  interface Profile {
    name: string;
    is_admin: boolean;
    activity_level: string | null;
    age: number | null;
    sex: string | null;
    height_cm: number | null;
    weight_kg: number | null;
    target_weight_kg: number | null;
    avatar_url: string | null;
  }

  let loading = $state(true);
  let saving = $state(false);
  let saved = $state(false);
  let isAdmin = $state(false);
  let fullProfile = $state<Profile | null>(null);

  // Profile section state
  let profileName = $state('');
  let profileAvatarURL = $state<string | null>(null);
  let avatarPreview = $state<string | null>(null);
  let avatarFile = $state<File | null>(null);
  let profileSaving = $state(false);
  let profileSaved = $state(false);
  let avatarUploading = $state(false);

  // Unit preferences
  let unitPrefs = $state<UnitPrefs>({ ...defaultUnits });

  // Notification preferences
  interface NotifPrefs {
    water_reminders: boolean;
    water_interval_hours: number;
    meal_reminders: boolean;
    if_window_reminders: boolean;
    streak_reminders: boolean;
    quiet_start: number;
    quiet_end: number;
    ntfy_topic: string;
  }
  let notifPrefs = $state<NotifPrefs>({
    water_reminders: true,
    water_interval_hours: 2,
    meal_reminders: true,
    if_window_reminders: true,
    streak_reminders: true,
    quiet_start: 22,
    quiet_end: 8,
    ntfy_topic: '',
  });
  let notifSupported = $state(typeof window !== 'undefined' && 'Notification' in window && 'serviceWorker' in navigator);
  let notifPermission = $state<NotificationPermission>('default');
  let notifSubscribed = $state(false);
  let notifSaving = $state(false);
  let notifSaved = $state(false);
  let notifTesting = $state(false);
  let vapidPublicKey = $state('');

  let objective = $state('maintain');
  let dietPlan = $state('balanced');
  let fastingWindow = $state('');
  let activityLevel = $state('sedentary');
  let manualOverride = $state(false);
  let calories = $state(2000);
  let proteinG = $state(150);
  let carbsG = $state(200);
  let fatG = $state(67);
  let currentWeight = $state<number | null>(null);
  let targetWeight = $state<number | null>(null);

  const objectives = [
    { value: 'cut_fat', label: 'Cut Fat', description: 'Calorie deficit for fat loss' },
    { value: 'feel_better', label: 'Feel Better', description: 'Mild deficit, focus on nutrition quality' },
    { value: 'maintain', label: 'Maintain', description: 'Maintain current weight' },
    { value: 'build_muscle', label: 'Build Muscle', description: 'Calorie surplus for muscle gain' },
  ];

  const dietPlans = [
    { value: 'balanced', label: 'Balanced' },
    { value: 'calorie_deficit', label: 'Calorie Deficit' },
    { value: 'keto', label: 'Keto' },
    { value: 'paleo', label: 'Paleo' },
    { value: 'mediterranean', label: 'Mediterranean' },
    { value: 'intermittent_fasting', label: 'Intermittent Fasting' },
  ];

  const activityLevels = [
    { value: 'sedentary', label: 'Sedentary', description: 'Little or no exercise' },
    { value: 'light', label: 'Light', description: '1–3 days/week' },
    { value: 'moderate', label: 'Moderate', description: '3–5 days/week' },
    { value: 'active', label: 'Active', description: '6–7 days/week' },
    { value: 'very_active', label: 'Very Active', description: 'Hard exercise daily' },
  ];

  onMount(() => {
    const unsub = authToken.subscribe(token => {
      if (!token) goto('/login');
    });

    (async () => {
      try {
        const [profile, goals, prefs] = await Promise.all([
          api.get<Profile>('/user/profile'),
          api.get<Goals>('/user/goals'),
          api.get<UnitPrefs & Record<string, string>>('/user/preferences'),
        ]);
        unitPrefs = {
          height_unit: (prefs.height_unit as UnitPrefs['height_unit']) || 'cm',
          weight_unit: (prefs.weight_unit as UnitPrefs['weight_unit']) || 'kg',
          energy_unit: (prefs.energy_unit as UnitPrefs['energy_unit']) || 'kcal',
        };

        // Load notification preferences and check subscription status
        try {
          const [notifData, vapidData] = await Promise.all([
            api.get<NotifPrefs>('/notifications/preferences'),
            api.get<{public_key: string}>('/notifications/vapid-public-key'),
          ]);
          notifPrefs = notifData;
          vapidPublicKey = vapidData.public_key;
        } catch {}

        if ('Notification' in window) {
          notifPermission = Notification.permission;
        }
        if ('serviceWorker' in navigator && vapidPublicKey) {
          const reg = await navigator.serviceWorker.ready;
          const sub = await reg.pushManager.getSubscription();
          notifSubscribed = !!sub;
        }
        isAdmin = profile.is_admin;
        fullProfile = profile;
        profileName = profile.name;
        profileAvatarURL = profile.avatar_url ?? null;
        activityLevel = profile.activity_level ?? 'sedentary';
        currentWeight = profile.weight_kg;
        targetWeight = profile.target_weight_kg;

        objective = goals.objective;
        dietPlan = goals.diet_plan;
        fastingWindow = goals.fasting_window ?? '';
        calories = goals.daily_calorie_target;
        proteinG = goals.daily_protein_g;
        carbsG = goals.daily_carbs_g;
        fatG = goals.daily_fat_g;
      } catch {
        goto('/dashboard');
      } finally {
        loading = false;
      }
    })();

    return unsub;
  });

  function getInitials(name: string): string {
    return name
      .split(' ')
      .map(p => p[0])
      .join('')
      .toUpperCase()
      .slice(0, 2) || '?';
  }

  function handleAvatarChange(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    avatarFile = file;

    const reader = new FileReader();
    reader.onload = (ev) => {
      avatarPreview = ev.target?.result as string;
    };
    reader.readAsDataURL(file);
  }

  async function saveProfile() {
    if (!fullProfile) return;
    profileSaving = true;
    profileSaved = false;
    try {
      // Upload avatar if a new file was chosen
      if (avatarFile) {
        avatarUploading = true;
        const form = new FormData();
        form.append('avatar', avatarFile);

        const token = get(authToken) ?? '';
        const res = await fetch('/api/user/avatar', {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}` },
          body: form,
        });
        if (res.ok) {
          const data = await res.json();
          profileAvatarURL = data?.data?.avatar_url ?? profileAvatarURL;
          avatarPreview = null;
          avatarFile = null;
        }
        avatarUploading = false;
      }

      // Update profile name (keeping all other required fields from fullProfile)
      const updated = await api.put<Profile>('/user/profile', {
        name: profileName,
        age: fullProfile.age ?? 25,
        sex: fullProfile.sex ?? 'male',
        height_cm: fullProfile.height_cm ?? 170,
        weight_kg: fullProfile.weight_kg ?? 70,
        target_weight_kg: fullProfile.target_weight_kg ?? 70,
        activity_level: fullProfile.activity_level ?? 'sedentary',
      });

      fullProfile = { ...updated, avatar_url: profileAvatarURL };

      profileSaved = true;
      setTimeout(() => { profileSaved = false; }, 3000);
    } catch {}
    finally {
      profileSaving = false;
      avatarUploading = false;
    }
  }

  async function enableNotifications() {
    if (!notifSupported || !vapidPublicKey) return;
    const permission = await Notification.requestPermission();
    notifPermission = permission;
    if (permission !== 'granted') return;

    const reg = await navigator.serviceWorker.ready;

    // Convert VAPID key from base64url to Uint8Array
    const keyStr = vapidPublicKey.replace(/-/g, '+').replace(/_/g, '/');
    const rawKey = Uint8Array.from(atob(keyStr), c => c.charCodeAt(0));

    const sub = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: rawKey,
    });

    const json = sub.toJSON();
    await api.post('/notifications/subscribe', {
      endpoint: json.endpoint,
      p256dh: (json.keys as Record<string, string>).p256dh,
      auth: (json.keys as Record<string, string>).auth,
      user_agent: navigator.userAgent,
    });

    notifSubscribed = true;
  }

  async function disableNotifications() {
    const reg = await navigator.serviceWorker.ready;
    const sub = await reg.pushManager.getSubscription();
    if (sub) {
      await api.post('/notifications/unsubscribe', { endpoint: sub.endpoint });
      await sub.unsubscribe();
    }
    notifSubscribed = false;
  }

  async function saveNotifPrefs() {
    notifSaving = true;
    notifSaved = false;
    try {
      await api.put('/notifications/preferences', notifPrefs);
      notifSaved = true;
      setTimeout(() => { notifSaved = false; }, 3000);
    } catch {}
    finally { notifSaving = false; }
  }

  async function sendTestNotification() {
    notifTesting = true;
    try { await api.post('/notifications/test', {}); } catch {}
    finally { setTimeout(() => { notifTesting = false; }, 2000); }
  }

  async function save() {
    saving = true;
    saved = false;
    try {
      // Update profile (including activity level) — must send all required fields
      if (fullProfile) {
        await api.put('/user/profile', {
          name: fullProfile.name,
          age: fullProfile.age ?? 25,
          sex: fullProfile.sex ?? 'male',
          height_cm: fullProfile.height_cm ?? 170,
          weight_kg: currentWeight ?? fullProfile.weight_kg ?? 70,
          target_weight_kg: targetWeight ?? fullProfile.target_weight_kg ?? 70,
          activity_level: activityLevel,
        });
        fullProfile = {
          ...fullProfile,
          activity_level: activityLevel,
          weight_kg: currentWeight,
          target_weight_kg: targetWeight,
        };
      }

      // Update goals
      const goals = await api.put<Goals>('/user/goals', {
        objective,
        diet_plan: dietPlan,
        fasting_window: dietPlan === 'intermittent_fasting' && fastingWindow ? fastingWindow : null,
        manual_override: manualOverride,
        ...(manualOverride ? {
          daily_calorie_target: calories,
          daily_protein_g: proteinG,
          daily_carbs_g: carbsG,
          daily_fat_g: fatG,
        } : {}),
      });

      // Save unit preferences
      await api.put('/user/preferences', {
        diet_type: 'omnivore',
        allergies: [],
        food_notes: '',
        eating_context: '',
        ...unitPrefs,
      });
      localStorage.setItem('unit_prefs', JSON.stringify(unitPrefs));

      // Update store
      userGoals.set({
        objective: goals.objective,
        diet_plan: goals.diet_plan,
        fasting_window: goals.fasting_window,
        daily_calorie_target: goals.daily_calorie_target,
        daily_protein_g: goals.daily_protein_g,
        daily_carbs_g: goals.daily_carbs_g,
        daily_fat_g: goals.daily_fat_g,
      });

      // Reflect recalculated values
      calories = goals.daily_calorie_target;
      proteinG = goals.daily_protein_g;
      carbsG = goals.daily_carbs_g;
      fatG = goals.daily_fat_g;

      saved = true;
      setTimeout(() => { saved = false; }, 3000);
    } catch {}
    finally { saving = false; }
  }
</script>

<div class="flex min-h-screen">
  <Sidebar activePage="settings" {isAdmin} />

  <main class="flex-1 overflow-y-auto p-4 pb-20 lg:p-10 lg:pb-10">
    {#if loading}
      <div class="flex h-64 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-slate-700 border-t-joule-500"></div>
      </div>
    {:else}
      <div class="mx-auto max-w-2xl space-y-8">
        <div class="flex items-center justify-between">
          <div>
            <h1 class="text-2xl font-bold text-white">Settings</h1>
            <p class="mt-1 text-sm text-slate-400">Adjust your goals, diet, and macro targets</p>
          </div>
          <div class="flex items-center gap-2">
            <ThemeToggle />
          </div>
        </div>

        <!-- Profile -->
        <div class="rounded-xl border border-slate-800 bg-surface-light p-5">
          <h2 class="mb-5 text-sm font-semibold text-joule-400">Profile</h2>
          <div class="flex items-center gap-5 mb-5">
            <!-- Avatar -->
            <div class="relative shrink-0">
              {#if avatarPreview}
                <img
                  src={avatarPreview}
                  alt="Avatar preview"
                  class="h-20 w-20 rounded-full object-cover border-2 border-joule-500/50"
                />
              {:else if profileAvatarURL}
                <img
                  src={profileAvatarURL}
                  alt="Profile avatar"
                  class="h-20 w-20 rounded-full object-cover border-2 border-slate-700"
                />
              {:else}
                <div class="h-20 w-20 rounded-full bg-joule-500/20 border-2 border-joule-500/30 flex items-center justify-center">
                  <span class="text-2xl font-bold text-joule-400">{getInitials(profileName || fullProfile?.name || '?')}</span>
                </div>
              {/if}
              <!-- Change photo overlay -->
              <label
                class="absolute inset-0 flex items-center justify-center rounded-full bg-black/50 opacity-0 hover:opacity-100 cursor-pointer transition-opacity"
                title="Change photo"
              >
                <svg class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                <input
                  type="file"
                  accept="image/*"
                  class="sr-only"
                  onchange={handleAvatarChange}
                />
              </label>
            </div>
            <div class="flex-1">
              <p class="text-xs text-slate-400 mb-1">Click the photo to change it</p>
              <p class="text-xs text-slate-500">JPG, PNG up to 5MB</p>
              {#if avatarPreview}
                <p class="text-xs text-joule-400 mt-1">New photo selected — save to upload</p>
              {/if}
            </div>
          </div>
          <!-- Name field -->
          <div class="mb-4">
            <label for="profile-name" class="mb-1.5 block text-xs font-medium text-slate-400">Display Name</label>
            <input
              id="profile-name"
              type="text"
              bind:value={profileName}
              placeholder="Your name"
              class="w-full rounded-lg border border-slate-700 bg-surface px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
            />
          </div>
          <!-- Save Profile -->
          <div class="flex items-center gap-3">
            {#if profileSaved}
              <span class="text-sm text-green-400">Profile saved!</span>
            {/if}
            <button
              onclick={saveProfile}
              disabled={profileSaving}
              class="rounded-xl bg-joule-500 px-5 py-2 text-sm font-semibold text-slate-900 hover:bg-joule-400 transition disabled:opacity-50"
            >
              {avatarUploading ? 'Uploading…' : profileSaving ? 'Saving…' : 'Save Profile'}
            </button>
          </div>
        </div>

        <!-- Objective -->
        <div class="rounded-xl border border-slate-800 bg-surface-light p-5">
          <h2 class="mb-4 text-sm font-semibold text-joule-400">Goal</h2>
          <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
            {#each objectives as obj}
              <button
                onclick={() => { objective = obj.value; }}
                class="flex items-start gap-3 rounded-lg border p-3 text-left transition
                  {objective === obj.value
                    ? 'border-joule-500/50 bg-joule-500/10 ring-1 ring-joule-500/20'
                    : 'border-slate-700 hover:border-slate-600'}"
              >
                <div class="mt-0.5 h-4 w-4 shrink-0 rounded-full border-2 flex items-center justify-center
                  {objective === obj.value ? 'border-joule-500' : 'border-slate-600'}">
                  {#if objective === obj.value}
                    <div class="h-2 w-2 rounded-full bg-joule-500"></div>
                  {/if}
                </div>
                <div>
                  <p class="text-sm font-medium text-white">{obj.label}</p>
                  <p class="text-xs text-slate-500">{obj.description}</p>
                </div>
              </button>
            {/each}
          </div>
        </div>

        <!-- Diet Plan -->
        <div class="rounded-xl border border-slate-800 bg-surface-light p-5">
          <h2 class="mb-4 text-sm font-semibold text-joule-400">Diet Plan</h2>
          <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
            {#each dietPlans as plan}
              <button
                onclick={() => { dietPlan = plan.value; }}
                class="rounded-lg border px-3 py-2 text-sm font-medium transition
                  {dietPlan === plan.value
                    ? 'border-joule-500/50 bg-joule-500/10 text-joule-400 ring-1 ring-joule-500/20'
                    : 'border-slate-700 text-slate-400 hover:border-slate-600 hover:text-slate-300'}"
              >
                {plan.label}
              </button>
            {/each}
          </div>

          {#if dietPlan === 'intermittent_fasting'}
            <div class="mt-4">
              <label for="fasting-window" class="mb-1.5 block text-xs font-medium text-slate-400">Fasting Window</label>
              <input
                id="fasting-window"
                type="text"
                bind:value={fastingWindow}
                placeholder="e.g. 16:8 or 7pm–11am"
                class="w-full rounded-lg border border-slate-700 bg-surface px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
              />
            </div>
          {/if}
        </div>

        <!-- Activity Level -->
        <div class="rounded-xl border border-slate-800 bg-surface-light p-5">
          <h2 class="mb-4 text-sm font-semibold text-joule-400">Activity Level</h2>
          <div class="space-y-2">
            {#each activityLevels as level}
              <button
                onclick={() => { activityLevel = level.value; }}
                class="flex w-full items-center gap-3 rounded-lg border p-3 text-left transition
                  {activityLevel === level.value
                    ? 'border-joule-500/50 bg-joule-500/10 ring-1 ring-joule-500/20'
                    : 'border-slate-700 hover:border-slate-600'}"
              >
                <div class="h-4 w-4 shrink-0 rounded-full border-2 flex items-center justify-center
                  {activityLevel === level.value ? 'border-joule-500' : 'border-slate-600'}">
                  {#if activityLevel === level.value}
                    <div class="h-2 w-2 rounded-full bg-joule-500"></div>
                  {/if}
                </div>
                <div>
                  <span class="text-sm font-medium text-white">{level.label}</span>
                  <span class="ml-2 text-xs text-slate-500">{level.description}</span>
                </div>
              </button>
            {/each}
          </div>
        </div>

        <!-- Body Measurements -->
        <div class="rounded-xl border border-slate-800 bg-surface-light p-5">
          <h2 class="mb-4 text-sm font-semibold text-joule-400">Body Measurements</h2>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="current-weight" class="mb-1.5 block text-xs font-medium text-slate-400">Current Weight</label>
              <div class="relative">
                <input
                  id="current-weight"
                  type="number"
                  step="0.1"
                  min="20"
                  max="500"
                  bind:value={currentWeight}
                  placeholder="e.g. 75.0"
                  class="w-full rounded-lg border border-slate-700 bg-surface px-3 py-2 pr-10 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                />
                <span class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-slate-500">kg</span>
              </div>
            </div>
            <div>
              <label for="target-weight" class="mb-1.5 block text-xs font-medium text-slate-400">Target Weight</label>
              <div class="relative">
                <input
                  id="target-weight"
                  type="number"
                  step="0.1"
                  min="20"
                  max="500"
                  bind:value={targetWeight}
                  placeholder="e.g. 70.0"
                  class="w-full rounded-lg border border-slate-700 bg-surface px-3 py-2 pr-10 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
                />
                <span class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-slate-500">kg</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Macro Targets -->
        <div class="rounded-xl border border-slate-800 bg-surface-light p-5">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-sm font-semibold text-joule-400">Macro Targets</h2>
            <label class="flex cursor-pointer items-center gap-2">
              <span class="text-xs text-slate-400">Manual override</span>
              <button
                role="switch"
                aria-checked={manualOverride}
                onclick={() => { manualOverride = !manualOverride; }}
                class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors focus:outline-none
                  {manualOverride ? 'bg-joule-500' : 'bg-slate-700'}"
              >
                <span class="inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform
                  {manualOverride ? 'translate-x-5' : 'translate-x-1'}"></span>
              </button>
            </label>
          </div>

          {#if !manualOverride}
            <p class="mb-4 text-xs text-slate-500">Macros will be auto-calculated from your goal + diet plan when you save.</p>
            <div class="grid grid-cols-2 gap-3 sm:grid-cols-4 opacity-50 pointer-events-none">
              <div class="rounded-lg border border-slate-700 p-3">
                <p class="text-xs text-slate-500">Calories</p>
                <p class="mt-1 text-lg font-bold text-white">{calories}</p>
                <p class="text-xs text-slate-600">kcal</p>
              </div>
              <div class="rounded-lg border border-slate-700 p-3">
                <p class="text-xs text-slate-500">Protein</p>
                <p class="mt-1 text-lg font-bold text-white">{proteinG}</p>
                <p class="text-xs text-slate-600">g</p>
              </div>
              <div class="rounded-lg border border-slate-700 p-3">
                <p class="text-xs text-slate-500">Carbs</p>
                <p class="mt-1 text-lg font-bold text-white">{carbsG}</p>
                <p class="text-xs text-slate-600">g</p>
              </div>
              <div class="rounded-lg border border-slate-700 p-3">
                <p class="text-xs text-slate-500">Fat</p>
                <p class="mt-1 text-lg font-bold text-white">{fatG}</p>
                <p class="text-xs text-slate-600">g</p>
              </div>
            </div>
          {:else}
            <p class="mb-4 text-xs text-slate-500">Set your own macro targets directly.</p>
            <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {#each [
                { label: 'Calories', unit: 'kcal', key: 'calories', val: calories, set: (v: number) => { calories = v; } },
                { label: 'Protein', unit: 'g', key: 'protein', val: proteinG, set: (v: number) => { proteinG = v; } },
                { label: 'Carbs', unit: 'g', key: 'carbs', val: carbsG, set: (v: number) => { carbsG = v; } },
                { label: 'Fat', unit: 'g', key: 'fat', val: fatG, set: (v: number) => { fatG = v; } },
              ] as field}
                <div class="rounded-lg border border-slate-700 p-3">
                  <p class="text-xs text-slate-500">{field.label}</p>
                  <input
                    type="number"
                    value={field.val}
                    oninput={(e) => field.set(parseInt((e.target as HTMLInputElement).value) || 0)}
                    min="0"
                    class="mt-1 w-full bg-transparent text-lg font-bold text-white focus:outline-none"
                  />
                  <p class="text-xs text-slate-600">{field.unit}</p>
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Display Units -->
        <div class="rounded-xl border border-slate-800 bg-surface-light p-6">
          <h2 class="mb-1 text-sm font-semibold" style="color: var(--color-text-primary)">Display Units</h2>
          <p class="mb-4 text-xs text-slate-400">Choose how measurements are shown throughout the app.</p>
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <!-- Height -->
            <div>
              <p class="mb-2 text-xs font-medium text-slate-400">Height</p>
              <div class="flex gap-2">
                {#each ['cm', 'ft'] as u}
                  <button
                    type="button"
                    onclick={() => unitPrefs = { ...unitPrefs, height_unit: u as 'cm' | 'ft' }}
                    class="flex-1 rounded-lg border py-2 text-sm font-medium transition {unitPrefs.height_unit === u ? 'border-joule-500 bg-joule-500/10 text-joule-400' : 'border-slate-700 text-slate-400 hover:border-slate-600'}"
                  >{u === 'ft' ? "ft / in" : "cm"}</button>
                {/each}
              </div>
            </div>
            <!-- Weight -->
            <div>
              <p class="mb-2 text-xs font-medium text-slate-400">Weight</p>
              <div class="flex gap-2">
                {#each ['kg', 'lbs'] as u}
                  <button
                    type="button"
                    onclick={() => unitPrefs = { ...unitPrefs, weight_unit: u as 'kg' | 'lbs' }}
                    class="flex-1 rounded-lg border py-2 text-sm font-medium transition {unitPrefs.weight_unit === u ? 'border-joule-500 bg-joule-500/10 text-joule-400' : 'border-slate-700 text-slate-400 hover:border-slate-600'}"
                  >{u}</button>
                {/each}
              </div>
            </div>
            <!-- Energy -->
            <div>
              <p class="mb-2 text-xs font-medium text-slate-400">Energy</p>
              <div class="flex gap-2">
                {#each ['kcal', 'kJ'] as u}
                  <button
                    type="button"
                    onclick={() => unitPrefs = { ...unitPrefs, energy_unit: u as 'kcal' | 'kJ' }}
                    class="flex-1 rounded-lg border py-2 text-sm font-medium transition {unitPrefs.energy_unit === u ? 'border-joule-500 bg-joule-500/10 text-joule-400' : 'border-slate-700 text-slate-400 hover:border-slate-600'}"
                  >{u}</button>
                {/each}
              </div>
            </div>
          </div>
        </div>

        <!-- Notifications -->
        <div class="rounded-2xl border border-slate-800 bg-slate-900/50 p-6">
          <h2 class="mb-5 text-base font-semibold text-white">Notifications</h2>

          {#if !notifSupported}
            <p class="text-sm text-slate-500">Push notifications are not supported in this browser.</p>
          {:else}
            <!-- Enable / disable toggle -->
            <div class="mb-5 flex items-center justify-between">
              <div>
                <p class="text-sm font-medium text-slate-200">Browser push notifications</p>
                <p class="text-xs text-slate-500 mt-0.5">Works when the app is open or backgrounded</p>
              </div>
              {#if notifSubscribed}
                <button onclick={disableNotifications} class="rounded-lg bg-slate-700 px-4 py-1.5 text-xs font-semibold text-slate-300 hover:bg-slate-600 transition">
                  Disable
                </button>
              {:else}
                <button onclick={enableNotifications} disabled={notifPermission === 'denied'} class="rounded-lg bg-joule-500 px-4 py-1.5 text-xs font-semibold text-slate-900 hover:bg-joule-400 transition disabled:opacity-40">
                  {notifPermission === 'denied' ? 'Blocked in browser' : 'Enable'}
                </button>
              {/if}
            </div>

            {#if notifSubscribed}
              <!-- Reminder toggles -->
              <div class="space-y-3 mb-5">
                <label class="flex items-center justify-between">
                  <span class="text-sm text-slate-300">💧 Water reminders</span>
                  <input type="checkbox" bind:checked={notifPrefs.water_reminders} class="accent-joule-500 h-4 w-4" />
                </label>
                {#if notifPrefs.water_reminders}
                  <div class="ml-4 flex items-center gap-2">
                    <span class="text-xs text-slate-500">Every</span>
                    <select bind:value={notifPrefs.water_interval_hours} class="rounded-lg border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-white">
                      {#each [1,2,3,4] as h}<option value={h}>{h}h</option>{/each}
                    </select>
                  </div>
                {/if}
                <label class="flex items-center justify-between">
                  <span class="text-sm text-slate-300">🍽️ Meal logging reminders</span>
                  <input type="checkbox" bind:checked={notifPrefs.meal_reminders} class="accent-joule-500 h-4 w-4" />
                </label>
                <label class="flex items-center justify-between">
                  <span class="text-sm text-slate-300">⏰ Intermittent fasting window alerts</span>
                  <input type="checkbox" bind:checked={notifPrefs.if_window_reminders} class="accent-joule-500 h-4 w-4" />
                </label>
                <label class="flex items-center justify-between">
                  <span class="text-sm text-slate-300">🔥 Streak at-risk reminder</span>
                  <input type="checkbox" bind:checked={notifPrefs.streak_reminders} class="accent-joule-500 h-4 w-4" />
                </label>
              </div>

              <!-- Quiet hours -->
              <div class="mb-5">
                <p class="mb-2 text-xs font-medium text-slate-400">Quiet hours (no notifications)</p>
                <div class="flex items-center gap-3">
                  <select bind:value={notifPrefs.quiet_start} class="rounded-lg border border-slate-700 bg-slate-800 px-2 py-1.5 text-sm text-white">
                    {#each Array.from({length:24},(_,i)=>i) as h}
                      <option value={h}>{String(h).padStart(2,'0')}:00</option>
                    {/each}
                  </select>
                  <span class="text-xs text-slate-500">to</span>
                  <select bind:value={notifPrefs.quiet_end} class="rounded-lg border border-slate-700 bg-slate-800 px-2 py-1.5 text-sm text-white">
                    {#each Array.from({length:24},(_,i)=>i) as h}
                      <option value={h}>{String(h).padStart(2,'0')}:00</option>
                    {/each}
                  </select>
                </div>
              </div>

              <!-- ntfy topic -->
              <div class="mb-5">
                <label class="block mb-1 text-xs font-medium text-slate-400" for="ntfy-topic">
                  ntfy topic (optional — for reliable delivery when browser is fully closed)
                </label>
                <input id="ntfy-topic" type="text" bind:value={notifPrefs.ntfy_topic}
                  placeholder="e.g. joules-harsh-abc123"
                  class="w-full rounded-xl border border-slate-700 bg-slate-800 px-4 py-2.5 text-sm text-white placeholder-slate-500 focus:border-joule-500 focus:outline-none" />
                <p class="mt-1.5 text-xs text-slate-500">
                  Set up ntfy at <span class="text-slate-400">ntfy.databunker.uk</span>, subscribe to a topic, then enter it here.
                  You'll need the <a href="https://ntfy.sh" target="_blank" class="text-joule-400 underline">ntfy app</a> on your phone.
                </p>
              </div>

              <!-- Save + test -->
              <div class="flex items-center gap-3">
                <button onclick={saveNotifPrefs} disabled={notifSaving} class="rounded-xl bg-joule-500 px-5 py-2 text-sm font-semibold text-slate-900 hover:bg-joule-400 transition disabled:opacity-50">
                  {notifSaving ? 'Saving…' : notifSaved ? '✓ Saved' : 'Save notification settings'}
                </button>
                <button onclick={sendTestNotification} disabled={notifTesting} class="rounded-xl border border-slate-700 px-4 py-2 text-sm text-slate-400 hover:border-slate-500 transition disabled:opacity-50">
                  {notifTesting ? 'Sent!' : 'Send test'}
                </button>
              </div>
            {/if}
          {/if}
        </div>

        <!-- Save -->
        <div class="flex items-center justify-end gap-3">
          {#if saved}
            <span class="text-sm text-green-400">Settings saved!</span>
          {/if}
          <button
            onclick={save}
            disabled={saving}
            class="rounded-xl bg-joule-500 px-6 py-2.5 text-sm font-semibold text-slate-900 hover:bg-joule-400 transition disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Save Settings'}
          </button>
        </div>
      </div>
    {/if}
  </main>
</div>
