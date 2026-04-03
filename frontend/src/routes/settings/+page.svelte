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
    eating_window_start: string | null;
    fasting_streak: number;
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

  // Dietary restrictions
  const restrictionOptions = [
    { value: 'gluten_free', label: 'Gluten Free' },
    { value: 'dairy_free', label: 'Dairy Free' },
    { value: 'nut_allergy', label: 'Nut Allergy' },
    { value: 'vegan', label: 'Vegan' },
    { value: 'vegetarian', label: 'Vegetarian' },
    { value: 'halal', label: 'Halal' },
    { value: 'kosher', label: 'Kosher' },
    { value: 'low_fodmap', label: 'Low FODMAP' },
    { value: 'low_sodium', label: 'Low Sodium' },
    { value: 'diabetic_friendly', label: 'Diabetic Friendly' },
  ];
  let selectedRestrictions = $state<string[]>([]);

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
  let ntfyURL = $state('');  // auto-filled after subscribe

  // Connections
  let googleFitConnected = $state(false);
  let connectionsLoading = $state(true);

  // Coach notes
  let coachNotes = $state('');
  let coachNotesLoading = $state(true);
  let coachNotesSaving = $state(false);
  let coachNotesSaved = $state(false);
  let showCoachNotes = $state(false);

  // Coach memories
  let coachMemories = $state<Array<{ id: string; category: string; content: string; source: string; created_at: string }>>([]);
  let memoriesLoading = $state(true);
  let showMemories = $state(false);

  // Coach reminders
  interface Reminder {
    id: string;
    type: string;
    message: string;
    reminder_time: string;
    enabled: boolean;
    created_at: string;
  }

  let reminders = $state<Reminder[]>([]);
  let remindersLoading = $state(false);

  async function loadReminders() {
    remindersLoading = true;
    try {
      reminders = await api.get<Reminder[]>('/coach/reminders');
    } catch { reminders = []; }
    finally { remindersLoading = false; }
  }

  async function toggleReminder(id: string, enabled: boolean) {
    try {
      await api.put(`/coach/reminders/${id}`, { enabled });
      reminders = reminders.map(r => r.id === id ? { ...r, enabled } : r);
    } catch {}
  }

  async function deleteReminder(id: string) {
    try {
      await api.del(`/coach/reminders/${id}`);
      reminders = reminders.filter(r => r.id !== id);
    } catch {}
  }

  let objective = $state('maintain');
  let dietPlan = $state('balanced');
  let fastingWindow = $state('');
  let activityLevel = $state('sedentary');

  const fastingOptions = [
    { value: '16:8', label: '16:8', desc: 'Beginner-friendly' },
    { value: '18:6', label: '18:6', desc: 'More aggressive' },
    { value: '20:4', label: '20:4', desc: 'Advanced' },
    { value: 'omad', label: 'OMAD', desc: 'One meal a day' },
  ];
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
          api.get<UnitPrefs & { dietary_restrictions?: string[] } & Record<string, string>>('/user/preferences'),
        ]);
        unitPrefs = {
          height_unit: (prefs.height_unit as UnitPrefs['height_unit']) || 'cm',
          weight_unit: (prefs.weight_unit as UnitPrefs['weight_unit']) || 'kg',
          energy_unit: (prefs.energy_unit as UnitPrefs['energy_unit']) || 'kcal',
        };
        selectedRestrictions = prefs.dietary_restrictions ?? [];

        // Load notification preferences and check subscription status
        try {
          const [notifData, vapidData] = await Promise.all([
            api.get<NotifPrefs>('/notifications/preferences'),
            api.get<{public_key: string}>('/notifications/vapid-public-key'),
          ]);
          notifPrefs = notifData;
          if (notifData.ntfy_topic && vapidPublicKey) {
            ntfyURL = `${import.meta.env.VITE_NTFY_BASE_URL || 'https://ntfy.sh'}/${notifData.ntfy_topic}`;
          }
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

    // Load Google Fit connection status independently
    api.get<{ connected: boolean }>('/steps/google/status')
      .then(s => { googleFitConnected = s.connected; })
      .catch(() => {})
      .finally(() => { connectionsLoading = false; });

    // Load coach notes
    api.get<{ notes: string }>('/user/coach-notes')
      .then(res => { coachNotes = res.notes || ''; })
      .catch(() => {})
      .finally(() => { coachNotesLoading = false; });

    // Load coach memories
    api.get<Array<{ id: string; category: string; content: string; source: string; created_at: string }>>('/user/coach-memories')
      .then(res => { coachMemories = res || []; })
      .catch(() => {})
      .finally(() => { memoriesLoading = false; });

    // Load coach reminders
    loadReminders();

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
    const sub_result = await api.post<{ntfy_url?: string; ntfy_topic?: string}>('/notifications/subscribe', {
      endpoint: json.endpoint,
      p256dh: (json.keys as Record<string, string>).p256dh,
      auth: (json.keys as Record<string, string>).auth,
      user_agent: navigator.userAgent,
    });

    notifSubscribed = true;
    if (sub_result?.ntfy_url) ntfyURL = sub_result.ntfy_url;
    if (sub_result?.ntfy_topic) { ntfyURL = `${import.meta.env.VITE_NTFY_BASE_URL || 'https://ntfy.sh'}/${sub_result.ntfy_topic}`; }
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

  async function saveCoachNotes() {
    coachNotesSaving = true;
    try {
      await api.put('/user/coach-notes', { notes: coachNotes });
      coachNotesSaved = true;
      setTimeout(() => { coachNotesSaved = false; }, 2000);
    } catch (e) {
      console.error('Failed to save coach notes', e);
    }
    coachNotesSaving = false;
  }

  async function deleteMemory(id: string) {
    try {
      await api.del(`/user/coach-memories/${id}`);
      coachMemories = coachMemories.filter(m => m.id !== id);
    } catch (e) {
      console.error('Failed to delete memory', e);
    }
  }

  function getCategoryColor(cat: string): string {
    const colors: Record<string, string> = {
      allergy: 'bg-red-500/20 text-red-400',
      health_condition: 'bg-orange-500/20 text-orange-400',
      preference: 'bg-blue-500/20 text-blue-400',
      habit: 'bg-green-500/20 text-green-400',
      routine: 'bg-purple-500/20 text-purple-400',
      goal: 'bg-emerald-500/20 text-emerald-400',
      misc: 'bg-gray-500/20 text-gray-400',
    };
    return colors[cat] || colors.misc;
  }

  function formatCategoryLabel(cat: string): string {
    return cat.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
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

      // Save unit preferences + dietary restrictions
      await api.put('/user/preferences', {
        diet_type: 'omnivore',
        allergies: [],
        food_notes: '',
        eating_context: '',
        dietary_restrictions: selectedRestrictions,
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
        eating_window_start: goals.eating_window_start ?? null,
        fasting_streak: goals.fasting_streak ?? 0,
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

<div class="flex min-h-screen overflow-x-hidden">
  <Sidebar activePage="settings" {isAdmin} />

  <main class="flex-1 min-w-0 overflow-y-auto overflow-x-hidden p-4 lg:p-8" style="padding-bottom: calc(5rem + env(safe-area-inset-bottom, 0px));">
    {#if loading}
      <div class="flex h-64 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-border border-t-primary"></div>
      </div>
    {:else}
      <div class="mx-auto max-w-2xl space-y-8">
        <div class="flex items-center justify-between">
          <div>
            <h1 class="font-display text-2xl font-bold text-foreground">Settings</h1>
            <p class="mt-0.5 text-xs text-muted-foreground">Adjust your goals, diet, and macro targets</p>
          </div>
          <div class="flex items-center gap-2">
            <ThemeToggle />
          </div>
        </div>

        <!-- Profile -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-5">Profile</p>
          <div class="flex items-center gap-5 mb-5">
            <!-- Avatar -->
            <div class="relative shrink-0">
              {#if avatarPreview}
                <img
                  src={avatarPreview}
                  alt="Avatar preview"
                  class="h-20 w-20 rounded-full object-cover border-2 border-primary/50"
                />
              {:else if profileAvatarURL}
                <img
                  src={profileAvatarURL}
                  alt="Profile avatar"
                  class="h-20 w-20 rounded-full object-cover border-2 border-border"
                />
              {:else}
                <div class="h-20 w-20 rounded-full bg-primary/20 border-2 border-primary/30 flex items-center justify-center">
                  <span class="text-2xl font-bold text-primary">{getInitials(profileName || fullProfile?.name || '?')}</span>
                </div>
              {/if}
              <!-- Change photo overlay -->
              <label
                class="absolute inset-0 flex items-center justify-center rounded-full bg-black/50 opacity-0 hover:opacity-100 cursor-pointer transition-opacity"
                title="Change photo"
              >
                <svg class="h-6 w-6 text-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
              <p class="text-xs text-foreground mb-1">Click the photo to change it</p>
              <p class="text-xs text-muted-foreground">JPG, PNG up to 5MB</p>
              {#if avatarPreview}
                <p class="text-xs text-primary mt-1">New photo selected — save to upload</p>
              {/if}
            </div>
          </div>
          <!-- Name field -->
          <div class="mb-4">
            <label for="profile-name" class="mb-1.5 block text-xs font-medium text-foreground">Display Name</label>
            <input
              id="profile-name"
              type="text"
              bind:value={profileName}
              placeholder="Your name"
              class="w-full rounded-xl border border-border bg-secondary px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
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
              class="rounded-xl bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50"
            >
              {avatarUploading ? 'Uploading…' : profileSaving ? 'Saving…' : 'Save Profile'}
            </button>
          </div>
        </div>

        <!-- Objective -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Goal</p>
          <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
            {#each objectives as obj}
              <button
                onclick={() => { objective = obj.value; }}
                class="flex items-start gap-3 rounded-lg border p-3 text-left transition
                  {objective === obj.value
                    ? 'border-primary/50 bg-primary/10 ring-1 ring-ring/20'
                    : 'border-border hover:border-border'}"
              >
                <div class="mt-0.5 h-4 w-4 shrink-0 rounded-full border-2 flex items-center justify-center
                  {objective === obj.value ? 'border-primary' : 'border-border'}">
                  {#if objective === obj.value}
                    <div class="h-2 w-2 rounded-full bg-primary"></div>
                  {/if}
                </div>
                <div>
                  <p class="text-sm font-medium text-foreground">{obj.label}</p>
                  <p class="text-xs text-muted-foreground">{obj.description}</p>
                </div>
              </button>
            {/each}
          </div>
        </div>

        <!-- Diet Plan -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Diet Plan</p>
          <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
            {#each dietPlans as plan}
              <button
                onclick={() => { dietPlan = plan.value; }}
                class="rounded-lg border px-3 py-2 text-sm font-medium transition
                  {dietPlan === plan.value
                    ? 'border-primary/50 bg-primary/10 text-primary ring-1 ring-ring/20'
                    : 'border-border text-foreground hover:border-border hover:text-foreground'}"
              >
                {plan.label}
              </button>
            {/each}
          </div>

          {#if dietPlan === 'intermittent_fasting'}
            <div class="mt-4">
              <label for="fasting-window" class="block text-sm font-medium text-foreground mb-2">Fasting Protocol</label>
              <div class="grid grid-cols-2 gap-2 sm:grid-cols-4" id="fasting-window">
                {#each fastingOptions as opt}
                  <button
                    type="button"
                    onclick={() => fastingWindow = opt.value}
                    class="rounded-lg border p-3 text-center text-sm transition
                      {fastingWindow === opt.value
                        ? 'border-primary bg-primary/10 text-primary'
                        : 'border-border bg-accent/50 text-foreground hover:border-border'}"
                  >
                    <div class="font-semibold">{opt.label}</div>
                    <div class="mt-0.5 text-xs text-foreground">{opt.desc}</div>
                  </button>
                {/each}
              </div>
            </div>
          {/if}
        </div>

        <!-- Activity Level -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Activity Level</p>
          <div class="space-y-2">
            {#each activityLevels as level}
              <button
                onclick={() => { activityLevel = level.value; }}
                class="flex w-full items-center gap-3 rounded-lg border p-3 text-left transition
                  {activityLevel === level.value
                    ? 'border-primary/50 bg-primary/10 ring-1 ring-ring/20'
                    : 'border-border hover:border-border'}"
              >
                <div class="h-4 w-4 shrink-0 rounded-full border-2 flex items-center justify-center
                  {activityLevel === level.value ? 'border-primary' : 'border-border'}">
                  {#if activityLevel === level.value}
                    <div class="h-2 w-2 rounded-full bg-primary"></div>
                  {/if}
                </div>
                <div>
                  <span class="text-sm font-medium text-foreground">{level.label}</span>
                  <span class="ml-2 text-xs text-muted-foreground">{level.description}</span>
                </div>
              </button>
            {/each}
          </div>
        </div>

        <!-- Body Measurements -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Body Measurements</p>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="current-weight" class="mb-1.5 block text-xs font-medium text-foreground">Current Weight</label>
              <div class="relative">
                <input
                  id="current-weight"
                  type="number"
                  step="0.1"
                  min="20"
                  max="500"
                  bind:value={currentWeight}
                  placeholder="e.g. 75.0"
                  class="w-full rounded-xl border border-border bg-secondary px-3 py-2.5 pr-10 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
                />
                <span class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">kg</span>
              </div>
            </div>
            <div>
              <label for="target-weight" class="mb-1.5 block text-xs font-medium text-foreground">Target Weight</label>
              <div class="relative">
                <input
                  id="target-weight"
                  type="number"
                  step="0.1"
                  min="20"
                  max="500"
                  bind:value={targetWeight}
                  placeholder="e.g. 70.0"
                  class="w-full rounded-xl border border-border bg-secondary px-3 py-2.5 pr-10 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 transition-colors"
                />
                <span class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">kg</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Macro Targets -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <div class="mb-4 flex items-center justify-between">
            <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Macro Targets</p>
            <label class="flex cursor-pointer items-center gap-2">
              <span class="text-xs text-foreground">Manual override</span>
              <button
                role="switch"
                aria-checked={manualOverride}
                onclick={() => { manualOverride = !manualOverride; }}
                class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors focus:outline-none
                  {manualOverride ? 'bg-primary' : 'bg-accent'}"
              >
                <span class="inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform
                  {manualOverride ? 'translate-x-5' : 'translate-x-1'}"></span>
              </button>
            </label>
          </div>

          {#if !manualOverride}
            <p class="mb-4 text-xs text-muted-foreground">Macros will be auto-calculated from your goal + diet plan when you save.</p>
            <div class="grid grid-cols-2 gap-3 sm:grid-cols-4 opacity-50 pointer-events-none">
              <div class="rounded-xl border border-border p-3">
                <p class="text-xs text-muted-foreground">Calories</p>
                <p class="mt-1 text-lg font-bold text-foreground">{calories}</p>
                <p class="text-xs text-muted-foreground">kcal</p>
              </div>
              <div class="rounded-xl border border-border p-3">
                <p class="text-xs text-muted-foreground">Protein</p>
                <p class="mt-1 text-lg font-bold text-foreground">{proteinG}</p>
                <p class="text-xs text-muted-foreground">g</p>
              </div>
              <div class="rounded-xl border border-border p-3">
                <p class="text-xs text-muted-foreground">Carbs</p>
                <p class="mt-1 text-lg font-bold text-foreground">{carbsG}</p>
                <p class="text-xs text-muted-foreground">g</p>
              </div>
              <div class="rounded-xl border border-border p-3">
                <p class="text-xs text-muted-foreground">Fat</p>
                <p class="mt-1 text-lg font-bold text-foreground">{fatG}</p>
                <p class="text-xs text-muted-foreground">g</p>
              </div>
            </div>
          {:else}
            <p class="mb-4 text-xs text-muted-foreground">Set your own macro targets directly.</p>
            <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {#each [
                { label: 'Calories', unit: 'kcal', key: 'calories', val: calories, set: (v: number) => { calories = v; } },
                { label: 'Protein', unit: 'g', key: 'protein', val: proteinG, set: (v: number) => { proteinG = v; } },
                { label: 'Carbs', unit: 'g', key: 'carbs', val: carbsG, set: (v: number) => { carbsG = v; } },
                { label: 'Fat', unit: 'g', key: 'fat', val: fatG, set: (v: number) => { fatG = v; } },
              ] as field}
                <div class="rounded-xl border border-border p-3">
                  <p class="text-xs text-muted-foreground">{field.label}</p>
                  <input
                    type="number"
                    value={field.val}
                    oninput={(e) => field.set(parseInt((e.target as HTMLInputElement).value) || 0)}
                    min="0"
                    class="mt-1 w-full bg-transparent text-lg font-bold text-foreground focus:outline-none"
                  />
                  <p class="text-xs text-muted-foreground">{field.unit}</p>
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Display Units -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Display Units</p>
          <p class="mb-4 text-xs text-foreground">Choose how measurements are shown throughout the app.</p>
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <!-- Height -->
            <div>
              <p class="mb-2 text-xs font-medium text-foreground">Height</p>
              <div class="flex gap-2">
                {#each ['cm', 'ft'] as u}
                  <button
                    type="button"
                    onclick={() => unitPrefs = { ...unitPrefs, height_unit: u as 'cm' | 'ft' }}
                    class="flex-1 rounded-lg border py-2 text-sm font-medium transition {unitPrefs.height_unit === u ? 'border-primary bg-primary/10 text-primary' : 'border-border text-foreground hover:border-border'}"
                  >{u === 'ft' ? "ft / in" : "cm"}</button>
                {/each}
              </div>
            </div>
            <!-- Weight -->
            <div>
              <p class="mb-2 text-xs font-medium text-foreground">Weight</p>
              <div class="flex gap-2">
                {#each ['kg', 'lbs'] as u}
                  <button
                    type="button"
                    onclick={() => unitPrefs = { ...unitPrefs, weight_unit: u as 'kg' | 'lbs' }}
                    class="flex-1 rounded-lg border py-2 text-sm font-medium transition {unitPrefs.weight_unit === u ? 'border-primary bg-primary/10 text-primary' : 'border-border text-foreground hover:border-border'}"
                  >{u}</button>
                {/each}
              </div>
            </div>
            <!-- Energy -->
            <div>
              <p class="mb-2 text-xs font-medium text-foreground">Energy</p>
              <div class="flex gap-2">
                {#each ['kcal', 'kJ'] as u}
                  <button
                    type="button"
                    onclick={() => unitPrefs = { ...unitPrefs, energy_unit: u as 'kcal' | 'kJ' }}
                    class="flex-1 rounded-lg border py-2 text-sm font-medium transition {unitPrefs.energy_unit === u ? 'border-primary bg-primary/10 text-primary' : 'border-border text-foreground hover:border-border'}"
                  >{u}</button>
                {/each}
              </div>
            </div>
          </div>
        </div>

        <!-- Dietary Restrictions -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Dietary Restrictions</p>
          <p class="mb-4 text-xs text-foreground">Select any dietary restrictions or allergies that apply to you.</p>
          <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
            {#each restrictionOptions as opt}
              {@const checked = selectedRestrictions.includes(opt.value)}
              <button
                type="button"
                onclick={() => {
                  if (checked) {
                    selectedRestrictions = selectedRestrictions.filter(r => r !== opt.value);
                  } else {
                    selectedRestrictions = [...selectedRestrictions, opt.value];
                  }
                }}
                class="flex items-center gap-2.5 rounded-lg border px-3 py-2.5 text-left transition {checked ? 'border-primary/50 bg-primary/10 ring-1 ring-ring/20' : 'border-border hover:border-border'}"
              >
                <div class="flex h-4 w-4 shrink-0 items-center justify-center rounded border {checked ? 'border-primary bg-primary' : 'border-border'}">
                  {#if checked}
                    <svg class="h-3 w-3 text-primary-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
                    </svg>
                  {/if}
                </div>
                <span class="text-sm {checked ? 'text-primary font-medium' : 'text-foreground'}">{opt.label}</span>
              </button>
            {/each}
          </div>
        </div>

        <!-- Notifications -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Notifications</p>

          {#if !notifSupported}
            <p class="text-sm text-muted-foreground">Push notifications are not supported in this browser.</p>
          {:else}
            <!-- Enable / disable toggle -->
            <div class="mb-5 flex items-center justify-between">
              <div>
                <p class="text-sm font-medium text-foreground/80">Browser push notifications</p>
                <p class="text-xs text-muted-foreground mt-0.5">Works when the app is open or backgrounded</p>
              </div>
              {#if notifSubscribed}
                <button onclick={disableNotifications} class="rounded-lg bg-accent px-4 py-1.5 text-xs font-semibold text-foreground hover:bg-accent transition">
                  Disable
                </button>
              {:else}
                <button onclick={enableNotifications} disabled={notifPermission === 'denied'} class="rounded-lg bg-primary px-4 py-1.5 text-xs font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-40">
                  {notifPermission === 'denied' ? 'Blocked in browser' : 'Enable'}
                </button>
              {/if}
            </div>

            {#if notifSubscribed}
              <!-- Reminder toggles -->
              <div class="space-y-3 mb-5">
                <label class="flex items-center justify-between">
                  <span class="text-sm text-foreground">💧 Water reminders</span>
                  <input type="checkbox" bind:checked={notifPrefs.water_reminders} class="accent-primary h-4 w-4" />
                </label>
                {#if notifPrefs.water_reminders}
                  <div class="ml-4 flex items-center gap-2">
                    <span class="text-xs text-muted-foreground">Every</span>
                    <select bind:value={notifPrefs.water_interval_hours} class="rounded-lg border border-border bg-secondary px-2 py-1 text-xs text-foreground">
                      {#each [1,2,3,4] as h}<option value={h}>{h}h</option>{/each}
                    </select>
                  </div>
                {/if}
                <label class="flex items-center justify-between">
                  <span class="text-sm text-foreground">🍽️ Meal logging reminders</span>
                  <input type="checkbox" bind:checked={notifPrefs.meal_reminders} class="accent-primary h-4 w-4" />
                </label>
                <label class="flex items-center justify-between">
                  <span class="text-sm text-foreground">⏰ Intermittent fasting window alerts</span>
                  <input type="checkbox" bind:checked={notifPrefs.if_window_reminders} class="accent-primary h-4 w-4" />
                </label>
                <label class="flex items-center justify-between">
                  <span class="text-sm text-foreground">🔥 Streak at-risk reminder</span>
                  <input type="checkbox" bind:checked={notifPrefs.streak_reminders} class="accent-primary h-4 w-4" />
                </label>
              </div>

              <!-- Quiet hours -->
              <div class="mb-5">
                <p class="mb-2 text-xs font-medium text-foreground">Quiet hours (no notifications)</p>
                <div class="flex items-center gap-3">
                  <select bind:value={notifPrefs.quiet_start} class="rounded-lg border border-border bg-secondary px-2 py-1.5 text-sm text-foreground">
                    {#each Array.from({length:24},(_,i)=>i) as h}
                      <option value={h}>{String(h).padStart(2,'0')}:00</option>
                    {/each}
                  </select>
                  <span class="text-xs text-muted-foreground">to</span>
                  <select bind:value={notifPrefs.quiet_end} class="rounded-lg border border-border bg-secondary px-2 py-1.5 text-sm text-foreground">
                    {#each Array.from({length:24},(_,i)=>i) as h}
                      <option value={h}>{String(h).padStart(2,'0')}:00</option>
                    {/each}
                  </select>
                </div>
              </div>

              <!-- ntfy — auto-configured, just tap subscribe -->
              <div class="mb-5 rounded-xl border border-border bg-card/50 p-4">
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <p class="text-sm font-medium text-foreground/80">ntfy — reliable when browser is closed</p>
                    <p class="text-xs text-muted-foreground mt-0.5">
                      Works via the <a href="https://ntfy.sh" target="_blank" class="text-primary underline">ntfy app</a> on your phone even when the browser is fully quit.
                    </p>
                  </div>
                  {#if ntfyURL}
                    <a href={ntfyURL.replace('https://', 'ntfy://')} class="shrink-0 rounded-lg bg-accent px-3 py-1.5 text-xs font-semibold text-foreground/80 hover:bg-accent transition">
                      Subscribe ↗
                    </a>
                  {:else}
                    <span class="shrink-0 text-xs text-muted-foreground">Auto-configured on enable</span>
                  {/if}
                </div>
                {#if ntfyURL}
                  <p class="mt-2 text-xs text-muted-foreground font-mono break-all">{ntfyURL}</p>
                {/if}
              </div>

              <!-- Save + test -->
              <div class="flex items-center gap-3">
                <button onclick={saveNotifPrefs} disabled={notifSaving} class="rounded-xl bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50">
                  {notifSaving ? 'Saving…' : notifSaved ? '✓ Saved' : 'Save notification settings'}
                </button>
                <button onclick={sendTestNotification} disabled={notifTesting} class="rounded-xl border border-border px-4 py-2 text-sm text-foreground hover:border-border transition disabled:opacity-50">
                  {notifTesting ? 'Sent!' : 'Send test'}
                </button>
              </div>
            {/if}
          {/if}
        </div>

        <!-- Coach Reminders -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <h2 class="text-xl font-semibold mb-4">Coach Reminders</h2>
          <p class="text-sm text-muted-foreground mb-4">Reminders set by your AI coach. Toggle or remove them anytime.</p>
          <div class="space-y-3">
            {#if remindersLoading}
              <p class="text-muted-foreground text-sm">Loading...</p>
            {:else if reminders.length === 0}
              <p class="text-muted-foreground text-sm">No reminders yet. Ask your coach to set one!</p>
            {:else}
              {#each reminders as r (r.id)}
                <div class="flex items-center justify-between p-3 rounded-lg border border-border bg-card">
                  <div class="flex-1">
                    <div class="flex items-center gap-2">
                      <span class="text-sm font-medium">{r.type}</span>
                      <span class="text-xs text-muted-foreground">{r.reminder_time}</span>
                    </div>
                    <p class="text-sm text-muted-foreground">{r.message}</p>
                  </div>
                  <div class="flex items-center gap-3">
                    <button
                      onclick={() => toggleReminder(r.id, !r.enabled)}
                      class="relative inline-flex h-6 w-12 items-center rounded-full transition-colors {r.enabled ? 'bg-primary' : 'bg-accent'}"
                    >
                      <span class="inline-block h-4 w-4 rounded-full bg-white transition-transform {r.enabled ? 'translate-x-6' : 'translate-x-1'}"></span>
                    </button>
                    <button onclick={() => deleteReminder(r.id)} class="text-muted-foreground hover:text-red-500 text-sm">✕</button>
                  </div>
                </div>
              {/each}
            {/if}
          </div>
          <button onclick={loadReminders} class="mt-3 px-4 py-2 text-sm rounded-lg border border-border hover:bg-accent">
            {remindersLoading ? 'Loading…' : 'Refresh Reminders'}
          </button>
        </div>

        <!-- Connections -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Connections</p>

          <!-- Google Fit -->
          <div class="flex items-center justify-between py-3 border-b border-border">
            <div class="flex items-center gap-3">
              <svg class="w-5 h-5 text-foreground" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"/>
              </svg>
              <div>
                <p class="text-sm font-medium text-foreground/80">Google Fit</p>
                <p class="text-xs text-muted-foreground">Sync your step count automatically</p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              {#if connectionsLoading}
                <span class="text-xs text-muted-foreground">Loading...</span>
              {:else if googleFitConnected}
                <span class="text-xs font-medium px-2 py-0.5 rounded-md bg-green-500/20 text-green-400">Connected</span>
              {:else}
                <button
                  onclick={() => { window.location.href = '/api/steps/google/connect'; }}
                  class="rounded-lg bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground hover:bg-primary/80 transition"
                >
                  Connect Google Fit
                </button>
              {/if}
            </div>
          </div>

          <!-- Apple Health -->
          <div class="flex items-center justify-between py-3">
            <div class="flex items-center gap-3">
              <svg class="w-5 h-5 text-foreground" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12z" />
              </svg>
              <div>
                <p class="text-sm font-medium text-foreground">Apple Health</p>
                <p class="text-xs text-muted-foreground">Apple Health sync requires the iOS app (coming soon)</p>
              </div>
            </div>
            <span class="text-xs text-muted-foreground font-medium">Coming soon</span>
          </div>
        </div>

        <!-- Coach Notes -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <button
            type="button"
            onclick={() => { showCoachNotes = !showCoachNotes; }}
            class="flex w-full items-center justify-between text-left"
          >
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Coach Notes</p>
              <p class="mt-0.5 text-xs text-muted-foreground">Leave persistent instructions for your AI coach</p>
            </div>
            <svg
              class="h-4 w-4 shrink-0 text-muted-foreground transition-transform {showCoachNotes ? 'rotate-180' : ''}"
              fill="none" viewBox="0 0 24 24" stroke="currentColor"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
            </svg>
          </button>

          {#if showCoachNotes}
            <div class="mt-4">
              {#if coachNotesLoading}
                <div class="flex h-20 items-center justify-center">
                  <div class="h-5 w-5 animate-spin rounded-full border-2 border-border border-t-primary"></div>
                </div>
              {:else}
                <textarea
                  bind:value={coachNotes}
                  maxlength={10000}
                  class="w-full min-h-[200px] p-3 rounded-lg border border-border bg-secondary text-foreground resize-y font-mono text-sm focus:ring-2 focus:ring-primary/50 focus:border-transparent focus:outline-none transition-colors"
                  placeholder="Write anything your coach should always know — allergies, preferences, routines, goals, motivational style, etc."
                ></textarea>
                <div class="mt-2 flex items-center justify-between">
                  <p class="text-xs text-muted-foreground">
                    Write anything your coach should always know — allergies, preferences, routines, goals, motivational style, etc. This is included in every conversation.
                  </p>
                </div>
                <div class="mt-3 flex items-center justify-between">
                  <span class="text-xs text-muted-foreground">{coachNotes.length.toLocaleString()} / 10,000</span>
                  <div class="flex items-center gap-3">
                    {#if coachNotesSaved}
                      <span class="text-sm text-green-400">Notes saved!</span>
                    {/if}
                    <button
                      onclick={saveCoachNotes}
                      disabled={coachNotesSaving}
                      class="rounded-xl bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50"
                    >
                      {coachNotesSaving ? 'Saving…' : 'Save Notes'}
                    </button>
                  </div>
                </div>
              {/if}
            </div>
          {/if}
        </div>

        <!-- Coach Memories -->
        <div class="rounded-2xl border border-border bg-card p-5">
          <button
            type="button"
            onclick={() => { showMemories = !showMemories; }}
            class="flex w-full items-center justify-between text-left"
          >
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Coach Memories</p>
              <p class="mt-0.5 text-xs text-muted-foreground">Things your coach remembers about you</p>
            </div>
            <svg
              class="h-4 w-4 shrink-0 text-muted-foreground transition-transform {showMemories ? 'rotate-180' : ''}"
              fill="none" viewBox="0 0 24 24" stroke="currentColor"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
            </svg>
          </button>

          {#if showMemories}
            <div class="mt-4">
              {#if memoriesLoading}
                <div class="flex h-20 items-center justify-center">
                  <div class="h-5 w-5 animate-spin rounded-full border-2 border-border border-t-primary"></div>
                </div>
              {:else if coachMemories.length === 0}
                <p class="text-sm text-muted-foreground py-4 text-center">No memories yet. Your coach will automatically save important facts as you chat.</p>
              {:else}
                <div class="space-y-2">
                  {#each coachMemories as memory}
                    <div class="flex items-start justify-between gap-3 p-3 rounded-lg border border-border">
                      <div class="min-w-0 flex-1">
                        <div class="mb-1.5 flex flex-wrap items-center gap-2">
                          <span class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide {getCategoryColor(memory.category)}">
                            {formatCategoryLabel(memory.category)}
                          </span>
                          <span class="inline-flex items-center rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
                            {memory.source === 'agent' ? '🤖 Agent' : '👤 User'}
                          </span>
                          <span class="text-[10px] text-muted-foreground">
                            {new Date(memory.created_at).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })}
                          </span>
                        </div>
                        <p class="text-sm text-foreground">{memory.content}</p>
                      </div>
                      <button
                        type="button"
                        onclick={() => deleteMemory(memory.id)}
                        class="shrink-0 rounded-lg p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition"
                        title="Delete memory"
                      >
                        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
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
            class="rounded-xl bg-primary px-6 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Save Settings'}
          </button>
        </div>
      </div>
    {/if}
  </main>
</div>
