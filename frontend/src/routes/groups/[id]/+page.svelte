<script lang="ts">
  import Sidebar from '$components/Sidebar.svelte';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { page } from '$app/state';

  interface GroupItem {
    id: string; name: string; description: string; type: string;
    invite_code: string; member_count: number; my_role: string;
  }
  interface LeaderboardEntry {
    rank: number; user_id: string; name: string; role: string; meals_7d: number; calories_7d: number; points: number;
  }
  interface ChallengeProgress { user_id: string; name: string; value: number; }
  interface Challenge {
    id: string; title: string; description: string; metric: string;
    target_value: number; start_date: string; end_date: string; progress: ChallengeProgress[];
  }

  let tab = $state<'leaderboard' | 'challenges'>('leaderboard');
  let group = $state<GroupItem | null>(null);
  let leaderboard = $state<LeaderboardEntry[]>([]);
  let challenges = $state<Challenge[]>([]);
  let loading = $state(true);
  let copied = $state(false);

  // Create challenge modal
  let showChallenge = $state(false);
  let chTitle = $state('');
  let chDesc = $state('');
  let chMetric = $state('meals');
  let chTarget = $state('7');
  let chStart = $state(new Date().toISOString().split('T')[0]);
  let chEnd = $state('');
  let chCreating = $state(false);

  let groupId = $derived(page.params.id);

  onMount(() => {
    const unsub = authToken.subscribe(t => { if (!t) goto('/login'); });
    loadAll();
    return unsub;
  });

  async function loadAll() {
    loading = true;
    try {
      const [g, lb, ch] = await Promise.all([
        api.get<GroupItem>(`/groups/${groupId}`),
        api.get<LeaderboardEntry[]>(`/groups/${groupId}/leaderboard`),
        api.get<Challenge[]>(`/groups/${groupId}/challenges`),
      ]);
      group = g;
      leaderboard = lb ?? [];
      challenges = ch ?? [];
    } catch {
      goto('/groups');
    } finally {
      loading = false;
    }
  }

  async function leaveGroup() {
    if (!confirm('Leave this group?')) return;
    await api.post(`/groups/${groupId}/leave`, {});
    goto('/groups');
  }

  async function deleteGroup() {
    if (!confirm('Delete this group? This cannot be undone.')) return;
    await api.del(`/groups/${groupId}`);
    goto('/groups');
  }

  function copyInviteCode() {
    if (!group) return;
    navigator.clipboard.writeText(group.invite_code);
    copied = true;
    setTimeout(() => { copied = false; }, 2000);
  }

  async function createChallenge() {
    if (chCreating || !chTitle.trim() || !chEnd) return;
    chCreating = true;
    try {
      const c = await api.post<Challenge>(`/groups/${groupId}/challenges`, {
        title: chTitle.trim(),
        description: chDesc.trim(),
        metric: chMetric,
        target_value: parseInt(chTarget, 10) || 7,
        start_date: chStart,
        end_date: chEnd,
      });
      challenges = [c, ...challenges];
      showChallenge = false;
      chTitle = ''; chDesc = ''; chMetric = 'meals'; chTarget = '7'; chEnd = '';
    } catch {} finally { chCreating = false; }
  }

  const metricLabel: Record<string, string> = {
    meals: 'meals logged', calories: 'kcal eaten', steps: 'steps walked', protein: 'g protein'
  };
</script>

<div class="flex min-h-screen overflow-x-hidden">
  <Sidebar activePage="groups" />

  <main class="flex-1 min-w-0 overflow-x-hidden p-4 pb-20 lg:p-10 lg:pb-10">
    {#if loading}
      <div class="flex h-48 items-center justify-center">
        <div class="h-7 w-7 animate-spin rounded-full border-2 border-border border-t-primary"></div>
      </div>
    {:else if group}
      <!-- Header -->
      <div class="mb-6">
        <div class="flex items-start justify-between gap-4 mb-1">
          <div class="min-w-0">
            <div class="flex items-center gap-2 mb-1">
              <a href="/groups" class="text-xs text-muted-foreground hover:text-foreground transition">← Groups</a>
            </div>
            <h1 class="text-2xl font-bold text-foreground truncate">{group.name}</h1>
            {#if group.description}
              <p class="mt-1 text-sm text-foreground">{group.description}</p>
            {/if}
          </div>
          <div class="flex gap-2 flex-shrink-0">
            {#if group.my_role === 'admin'}
              <button onclick={deleteGroup} class="rounded-lg border border-red-900/50 px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/10 transition">Delete</button>
            {:else}
              <button onclick={leaveGroup} class="rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:text-foreground transition">Leave</button>
            {/if}
          </div>
        </div>

        <!-- Group meta row -->
        <div class="flex flex-wrap items-center gap-3 mt-3">
          <span class="text-xs text-foreground">{group.member_count} member{group.member_count !== 1 ? 's' : ''}</span>
          <span class="text-muted-foreground">·</span>
          <span class="text-xs text-foreground">{group.type}</span>
          {#if group.type === 'private'}
            <span class="text-muted-foreground">·</span>
            <button
              onclick={copyInviteCode}
              class="flex items-center gap-1.5 text-xs font-mono text-foreground hover:text-primary transition"
            >
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" /></svg>
              {copied ? 'Copied!' : group.invite_code}
            </button>
          {/if}
        </div>
      </div>

      <!-- Tabs -->
      <div class="flex gap-1 mb-6 rounded-lg bg-accent/50 p-1 w-fit">
        <button onclick={() => { tab = 'leaderboard'; }} class="rounded-md px-4 py-1.5 text-sm font-medium transition {tab === 'leaderboard' ? 'bg-accent text-foreground' : 'text-foreground hover:text-foreground/80'}">Leaderboard</button>
        <button onclick={() => { tab = 'challenges'; }} class="rounded-md px-4 py-1.5 text-sm font-medium transition {tab === 'challenges' ? 'bg-accent text-foreground' : 'text-foreground hover:text-foreground/80'}">Challenges</button>
      </div>

      <!-- Leaderboard tab -->
      {#if tab === 'leaderboard'}
        <div class="rounded-xl border border-border bg-card overflow-hidden">
          <div class="px-5 py-4 border-b border-border">
            <h2 class="text-sm font-semibold text-foreground">This week's rankings</h2>
            <p class="text-xs text-muted-foreground mt-0.5">Ranked by meals logged in the last 7 days</p>
          </div>
          {#if leaderboard.length === 0}
            <div class="flex items-center justify-center py-12 text-sm text-muted-foreground">No activity yet</div>
          {:else}
            {#each leaderboard as entry}
              <div class="flex items-center gap-4 px-5 py-3.5 border-b border-border/60 last:border-0">
                <!-- Rank -->
                <div class="w-7 flex-shrink-0 text-center">
                  {#if entry.rank === 1}
                    <span class="text-lg">🥇</span>
                  {:else if entry.rank === 2}
                    <span class="text-lg">🥈</span>
                  {:else if entry.rank === 3}
                    <span class="text-lg">🥉</span>
                  {:else}
                    <span class="text-sm font-bold text-muted-foreground">#{entry.rank}</span>
                  {/if}
                </div>
                <!-- Avatar placeholder + name -->
                <div class="flex items-center gap-2.5 flex-1 min-w-0">
                  <div class="h-8 w-8 flex-shrink-0 rounded-full bg-gradient-to-br from-primary/80 to-primary flex items-center justify-center text-xs font-bold text-primary-foreground">
                    {entry.name.charAt(0).toUpperCase()}
                  </div>
                  <div class="min-w-0">
                    <p class="text-sm font-medium text-foreground truncate">{entry.name}</p>
                    {#if entry.role === 'admin'}
                      <p class="text-[10px] text-primary">admin</p>
                    {/if}
                  </div>
                </div>
                <!-- Stats -->
                <div class="text-right flex-shrink-0">
                  <p class="text-sm font-semibold text-foreground">{entry.meals_7d} meals</p>
                  <p class="text-xs text-muted-foreground">{entry.calories_7d.toLocaleString()} kcal</p>
                  <p class="text-xs text-primary">{entry.points?.toLocaleString() ?? 0} XP</p>
                </div>
              </div>
            {/each}
          {/if}
        </div>
      {/if}

      <!-- Challenges tab -->
      {#if tab === 'challenges'}
        <div class="flex items-center justify-between mb-4">
          <p class="text-sm text-foreground">{challenges.length} challenge{challenges.length !== 1 ? 's' : ''}</p>
          {#if group.my_role === 'admin'}
            <button
              onclick={() => { showChallenge = true; }}
              class="rounded-lg bg-primary px-3.5 py-1.5 text-xs font-semibold text-primary-foreground hover:bg-primary/80 transition"
            >
              + New Challenge
            </button>
          {/if}
        </div>

        {#if challenges.length === 0}
          <div class="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-14 text-center">
            <p class="text-sm text-foreground">No challenges yet</p>
            {#if group.my_role === 'admin'}
              <p class="mt-1 text-xs text-muted-foreground">Create a challenge to motivate your group</p>
            {/if}
          </div>
        {:else}
          <div class="space-y-4">
            {#each challenges as ch}
              <div class="rounded-xl border border-border bg-card p-5">
                <div class="flex items-start justify-between mb-3">
                  <div>
                    <h3 class="font-semibold text-foreground">{ch.title}</h3>
                    {#if ch.description}<p class="text-xs text-foreground mt-0.5">{ch.description}</p>{/if}
                  </div>
                  <div class="text-right flex-shrink-0 ml-4">
                    <p class="text-xs text-muted-foreground">{ch.start_date} → {ch.end_date}</p>
                    <p class="text-xs text-primary mt-0.5">Goal: {ch.target_value} {metricLabel[ch.metric] ?? ch.metric}</p>
                  </div>
                </div>
                <!-- Progress list -->
                {#if ch.progress.length > 0}
                  <div class="space-y-2 mt-3 pt-3 border-t border-border">
                    {#each ch.progress as p, i}
                      {@const pct = ch.target_value > 0 ? Math.min(100, Math.round((p.value / ch.target_value) * 100)) : 0}
                      <div class="flex items-center gap-3">
                        <span class="text-xs text-muted-foreground w-4 flex-shrink-0">#{i+1}</span>
                        <span class="text-xs text-foreground flex-shrink-0 w-24 truncate">{p.name}</span>
                        <div class="flex-1 h-1.5 rounded-full bg-accent overflow-hidden">
                          <div class="h-full rounded-full {pct >= 100 ? 'bg-emerald-500' : 'bg-primary'} transition-all" style="width:{pct}%"></div>
                        </div>
                        <span class="text-xs text-foreground flex-shrink-0 w-16 text-right">{p.value} / {ch.target_value}</span>
                      </div>
                    {/each}
                  </div>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      {/if}
    {/if}
  </main>
</div>

<!-- Create challenge modal -->
{#if showChallenge}
  <div class="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/60 backdrop-blur-sm" onclick={() => { showChallenge = false; }} role="presentation">
    <div class="w-full max-w-md rounded-t-2xl sm:rounded-2xl border border-border/60 bg-secondary p-6 pb-8" onclick={(e) => e.stopPropagation()} role="dialog">
      <h2 class="mb-5 text-base font-bold text-foreground">New Challenge</h2>
      <div class="space-y-3">
        <div>
          <label class="mb-1 block text-xs font-medium text-foreground">Title</label>
          <input bind:value={chTitle} type="text" placeholder="7-day logging streak" class="w-full rounded-lg border border-border bg-secondary px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-foreground">Description (optional)</label>
          <input bind:value={chDesc} type="text" placeholder="Who can log meals every day?" class="w-full rounded-lg border border-border bg-secondary px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none" />
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="mb-1 block text-xs font-medium text-foreground">Metric</label>
            <select bind:value={chMetric} class="w-full rounded-lg border border-border bg-secondary px-3 py-2.5 text-sm text-foreground focus:border-primary focus:outline-none">
              <option value="meals">Meals logged</option>
              <option value="calories">Calories eaten</option>
              <option value="steps">Steps walked</option>
              <option value="protein">Protein (g)</option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-foreground">Target</label>
            <input bind:value={chTarget} type="number" min="1" placeholder="7" class="w-full rounded-lg border border-border bg-secondary px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none" />
          </div>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="mb-1 block text-xs font-medium text-foreground">Start date</label>
            <input bind:value={chStart} type="date" class="w-full rounded-lg border border-border bg-secondary px-3 py-2.5 text-sm text-foreground focus:border-primary focus:outline-none" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-foreground">End date</label>
            <input bind:value={chEnd} type="date" class="w-full rounded-lg border border-border bg-secondary px-3 py-2.5 text-sm text-foreground focus:border-primary focus:outline-none" />
          </div>
        </div>
      </div>
      <div class="mt-5 flex gap-2">
        <button onclick={() => { showChallenge = false; }} class="flex-1 rounded-lg border border-border py-2.5 text-sm text-foreground hover:text-foreground transition">Cancel</button>
        <button onclick={createChallenge} disabled={chCreating || !chTitle.trim() || !chEnd} class="flex-1 rounded-lg bg-primary py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50">
          {chCreating ? 'Creating...' : 'Create'}
        </button>
      </div>
    </div>
  </div>
{/if}
