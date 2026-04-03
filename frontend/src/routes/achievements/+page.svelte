<script lang="ts">
  import Sidebar from '$components/Sidebar.svelte';
  import ThemeToggle from '$components/ThemeToggle.svelte';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';

  interface Achievement {
    id: string;
    type: string;
    title: string;
    description: string;
    unlocked_at: string;
  }

  const allAchievements = [
    { type: 'first_meal', title: 'First Bite', description: 'Logged your first meal', icon: 'utensils' },
    { type: 'first_weight', title: 'Scale It', description: 'Logged your weight for the first time', icon: 'scale' },
    { type: 'first_exercise', title: 'Getting Active', description: 'Logged your first exercise', icon: 'activity' },
    { type: 'first_water', title: 'Hydration Start', description: 'Started tracking water intake', icon: 'water' },
    { type: 'first_chat', title: 'Coach Connection', description: 'Had your first chat with the coach', icon: 'chat' },
    { type: 'streak_3', title: '3-Day Streak', description: 'Logged meals for 3 consecutive days', icon: 'fire' },
    { type: 'streak_7', title: 'Week Warrior', description: 'Logged meals for 7 consecutive days', icon: 'fire' },
    { type: 'calorie_goal', title: 'On Target', description: 'Hit your daily calorie goal', icon: 'target' },
    { type: 'protein_goal', title: 'Protein Power', description: 'Hit your daily protein goal', icon: 'protein' },
    { type: 'perfect_day', title: 'Perfect Day', description: 'Hit both calorie and protein goals', icon: 'star' },
    { type: 'water_goal', title: 'Hydrated', description: 'Drank 2500ml+ in a day', icon: 'droplet' },
  ];

  function getIcon(icon: string) {
    switch (icon) {
      case 'utensils':
        return '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 6v14a2 2 0 002 2h12a2 2 0 002-2V6M4 6l2-4h12l2 4M9 14h.01M15 14h.01" />';
      case 'scale':
        return '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 6l3 1m0 0l-3 9a5.002 5.002 0 006.001 0M6 7l3 9M6 7l6-2m6 2l3-1m-3 1l-3 9a5.002 5.002 0 006.001 0M18 7l3 9m-3-9l-6-2m0-2v2m0 2v6m0-2v6" />';
      case 'activity':
        return '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M22 12h-4l-3 9L9 3l-3 9H2" />';
      case 'water':
        return '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 2.25c-3 4.5-6.75 7.5-6.75 12a6.75 6.75 0 1013.5 0c0-4.5-3.75-7.5-6.75-12z" />';
      case 'chat':
        return '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />';
      case 'fire':
        return '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 18.657A8 8 0 016.343 7.343S7 9 9 10c0-2 .5-5 2.986-7C14 5 16.09 5.777 17.656 7.343A7.975 7.975 0 0120 13a7.975 7.975 0 01-2.343 5.657z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.879 16.121A3 3 0 1012.015 11L11 14l1.879 2.121z" />';
      case 'target':
        return '<circle cx="12" cy="12" r="10" stroke-width="2" /><circle cx="12" cy="12" r="6" stroke-width="2" /><circle cx="12" cy="12" r="2" stroke-width="2" />';
      case 'droplet':
        return '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 21a8 8 0 01-8-8c0-4.5 8-11 8-11s8 6.5 8 11a8 8 0 01-8 8z" />';
      case 'protein':
        return '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18" />';
      case 'star':
        return '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />';
      default:
        return '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />';
    }
  }

  let achievements = $state<Achievement[]>([]);
  let loading = $state(true);

  onMount(() => {
    const unsub = authToken.subscribe((token) => {
      if (!token) goto('/login');
    });

    (async () => {
      try {
        await api.post('/achievements/check', {});
        const data = await api.get<Achievement[]>('/achievements');
        achievements = data;
      } catch {}
      finally { loading = false; }
    })();

    return unsub;
  });

  let unlockedCount = $derived(achievements.length);
  let totalCount = $derived(allAchievements.length);

  function formatDate(iso: string) {
    return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  }
</script>

<div class="flex min-h-screen overflow-x-hidden">
  <Sidebar activePage="achievements" />

  <main class="flex-1 min-w-0 overflow-y-auto overflow-x-hidden p-4 pb-20 lg:p-10 lg:pb-10">
    {#if loading}
      <div class="flex h-64 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-border border-t-primary"></div>
      </div>
    {:else}
      <div class="flex items-center justify-between mb-8">
        <div>
          <h1 class="text-2xl font-bold text-foreground">Achievements</h1>
          <p class="mt-1 text-sm text-foreground">{unlockedCount} of {totalCount} unlocked</p>
        </div>
        <div class="flex items-center gap-2">
          <ThemeToggle />
          <button
            onclick={() => { authToken.set(null); goto('/login'); }}
            class="rounded-lg border border-border px-3 py-1.5 text-sm text-foreground hover:text-foreground transition"
          >
            Sign out
          </button>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {#each allAchievements as def}
          {@const unlocked = achievements.find(a => a.type === def.type)}
          <div class="rounded-xl border p-5 text-center {unlocked ? 'border-primary/50 bg-primary/5' : 'border-border bg-card'}">
            <div class="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full {unlocked ? 'bg-primary/20' : 'bg-accent'}">
              <svg class="h-6 w-6 {unlocked ? 'text-primary' : 'text-muted-foreground'}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                {@html getIcon(def.icon)}
              </svg>
            </div>
            <h3 class="text-sm font-semibold {unlocked ? 'text-foreground' : 'text-foreground'}">{def.title}</h3>
            <p class="mt-1 text-xs {unlocked ? 'text-foreground' : 'text-muted-foreground'}">{def.description}</p>
            {#if unlocked}
              <p class="mt-2 text-xs text-primary">Unlocked {formatDate(unlocked.unlocked_at)}</p>
            {:else}
              <p class="mt-2 text-xs text-muted-foreground">Locked</p>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </main>
</div>