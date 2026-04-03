<script lang="ts">
  import Sidebar from '$components/Sidebar.svelte';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';

  interface GroupItem {
    id: string; name: string; description: string; type: string;
    invite_code: string; member_count: number; my_role: string;
  }
  interface PublicGroupItem {
    id: string; name: string; description: string; member_count: number;
  }

  let tab = $state<'mine' | 'discover'>('mine');
  let myGroups = $state<GroupItem[]>([]);
  let publicGroups = $state<PublicGroupItem[]>([]);
  let loading = $state(true);
  let discoverLoading = $state(false);
  let discoverLoaded = $state(false);

  // Create modal state
  let showCreate = $state(false);
  let createName = $state('');
  let createDesc = $state('');
  let createType = $state<'private' | 'public'>('private');
  let creating = $state(false);

  // Join modal state
  let showJoin = $state(false);
  let joinCode = $state('');
  let joining = $state(false);
  let joinError = $state('');

  onMount(() => {
    const unsub = authToken.subscribe(t => { if (!t) goto('/login'); });
    loadMyGroups();
    return unsub;
  });

  async function loadMyGroups() {
    loading = true;
    try {
      myGroups = await api.get<GroupItem[]>('/groups');
    } catch {} finally { loading = false; }
  }

  async function loadDiscover() {
    if (discoverLoaded) return;
    discoverLoading = true;
    try {
      publicGroups = await api.get<PublicGroupItem[]>('/groups/discover');
      discoverLoaded = true;
    } catch {} finally { discoverLoading = false; }
  }

  function switchTab(t: 'mine' | 'discover') {
    tab = t;
    if (t === 'discover') loadDiscover();
  }

  async function createGroup() {
    if (creating || !createName.trim()) return;
    creating = true;
    try {
      const g = await api.post<GroupItem>('/groups', { name: createName.trim(), description: createDesc.trim(), type: createType });
      myGroups = [g, ...myGroups];
      showCreate = false;
      createName = ''; createDesc = ''; createType = 'private';
    } catch {} finally { creating = false; }
  }

  async function joinByCode() {
    if (joining || !joinCode.trim()) return;
    joining = true; joinError = '';
    try {
      const g = await api.post<GroupItem>('/groups/join', { invite_code: joinCode.trim() });
      myGroups = [g, ...myGroups];
      showJoin = false; joinCode = '';
      tab = 'mine';
    } catch (e: any) {
      joinError = e?.message ?? 'Invalid invite code';
    } finally { joining = false; }
  }

  async function joinPublic(groupId: string) {
    try {
      const g = await api.post<GroupItem>('/groups/join', { group_id: groupId });
      myGroups = [g, ...myGroups];
      tab = 'mine';
    } catch {}
  }
</script>

<div class="flex min-h-screen overflow-x-hidden">
  <Sidebar activePage="groups" />

  <main class="flex-1 min-w-0 overflow-x-hidden p-4 pb-20 lg:p-10 lg:pb-10">
    <!-- Header -->
    <div class="mb-6 flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-foreground">Groups</h1>
        <p class="mt-1 text-sm text-foreground">Stay accountable with friends</p>
      </div>
      <div class="flex gap-2">
        <button
          onclick={() => { showJoin = true; }}
          class="rounded-lg border border-border px-3 py-2 text-sm font-medium text-foreground hover:text-foreground hover:border-border transition"
        >
          Join
        </button>
        <button
          onclick={() => { showCreate = true; }}
          class="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition"
        >
          + New Group
        </button>
      </div>
    </div>

    <!-- Tabs -->
    <div class="flex gap-1 mb-6 rounded-lg bg-accent/50 p-1 w-fit">
      <button
        onclick={() => switchTab('mine')}
        class="rounded-md px-4 py-1.5 text-sm font-medium transition {tab === 'mine' ? 'bg-accent text-foreground' : 'text-foreground hover:text-foreground/80'}"
      >My Groups</button>
      <button
        onclick={() => switchTab('discover')}
        class="rounded-md px-4 py-1.5 text-sm font-medium transition {tab === 'discover' ? 'bg-accent text-foreground' : 'text-foreground hover:text-foreground/80'}"
      >Discover</button>
    </div>

    <!-- My Groups tab -->
    {#if tab === 'mine'}
      {#if loading}
        <div class="flex h-48 items-center justify-center">
          <div class="h-7 w-7 animate-spin rounded-full border-2 border-border border-t-primary"></div>
        </div>
      {:else if myGroups.length === 0}
        <div class="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-16 text-center">
          <svg class="mb-3 h-10 w-10 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z" />
          </svg>
          <p class="text-sm font-medium text-foreground">No groups yet</p>
          <p class="mt-1 text-xs text-muted-foreground">Create a group or join one with an invite code</p>
        </div>
      {:else}
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {#each myGroups as group}
            <a
              href="/groups/{group.id}"
              class="group rounded-xl border border-border bg-card p-5 hover:border-border transition block"
            >
              <div class="flex items-start justify-between mb-2">
                <h3 class="font-semibold text-foreground group-hover:text-primary transition truncate">{group.name}</h3>
                <span class="ml-2 flex-shrink-0 text-[10px] font-medium px-1.5 py-0.5 rounded-full {group.type === 'public' ? 'bg-blue-500/20 text-blue-400' : 'bg-accent text-foreground'}">{group.type}</span>
              </div>
              {#if group.description}
                <p class="text-xs text-muted-foreground mb-3 line-clamp-2">{group.description}</p>
              {/if}
              <div class="flex items-center justify-between">
                <span class="text-xs text-foreground">{group.member_count} member{group.member_count !== 1 ? 's' : ''}</span>
                {#if group.my_role === 'admin'}
                  <span class="text-[10px] font-medium text-primary bg-primary/10 px-1.5 py-0.5 rounded-full">admin</span>
                {/if}
              </div>
            </a>
          {/each}
        </div>
      {/if}
    {/if}

    <!-- Discover tab -->
    {#if tab === 'discover'}
      {#if discoverLoading}
        <div class="flex h-48 items-center justify-center">
          <div class="h-7 w-7 animate-spin rounded-full border-2 border-border border-t-primary"></div>
        </div>
      {:else if publicGroups.length === 0}
        <div class="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-16 text-center">
          <p class="text-sm text-foreground">No public groups yet</p>
          <p class="mt-1 text-xs text-muted-foreground">Create one and set it to public!</p>
        </div>
      {:else}
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {#each publicGroups as group}
            <div class="rounded-xl border border-border bg-card p-5">
              <div class="flex items-start justify-between mb-2">
                <h3 class="font-semibold text-foreground truncate">{group.name}</h3>
                <span class="ml-2 flex-shrink-0 text-[10px] font-medium px-1.5 py-0.5 rounded-full bg-blue-500/20 text-blue-400">public</span>
              </div>
              {#if group.description}
                <p class="text-xs text-muted-foreground mb-3 line-clamp-2">{group.description}</p>
              {/if}
              <div class="flex items-center justify-between mt-3">
                <span class="text-xs text-foreground">{group.member_count} member{group.member_count !== 1 ? 's' : ''}</span>
                <button
                  onclick={() => joinPublic(group.id)}
                  class="text-xs font-medium text-primary hover:text-primary/80 transition"
                >
                  Join →
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {/if}
  </main>
</div>

<!-- Create group modal -->
{#if showCreate}
  <div class="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/60 backdrop-blur-sm" onclick={() => { showCreate = false; }} role="presentation">
    <div class="w-full max-w-md rounded-t-2xl sm:rounded-2xl border border-border/60 bg-secondary p-6 pb-8" onclick={(e) => e.stopPropagation()} role="dialog">
      <h2 class="mb-5 text-base font-bold text-foreground">Create Group</h2>
      <div class="space-y-4">
        <div>
          <label class="mb-1.5 block text-xs font-medium text-foreground">Group name</label>
          <input bind:value={createName} type="text" placeholder="e.g. Gym Squad" class="w-full rounded-lg border border-border bg-secondary px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none" />
        </div>
        <div>
          <label class="mb-1.5 block text-xs font-medium text-foreground">Description (optional)</label>
          <textarea bind:value={createDesc} placeholder="What's this group about?" rows="2" class="w-full rounded-lg border border-border bg-secondary px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none resize-none"></textarea>
        </div>
        <div>
          <label class="mb-1.5 block text-xs font-medium text-foreground">Visibility</label>
          <div class="flex gap-2">
            <button onclick={() => { createType = 'private'; }} class="flex-1 rounded-lg border py-2 text-sm font-medium transition {createType === 'private' ? 'border-primary bg-primary/10 text-primary' : 'border-border text-foreground'}">Private</button>
            <button onclick={() => { createType = 'public'; }} class="flex-1 rounded-lg border py-2 text-sm font-medium transition {createType === 'public' ? 'border-blue-500 bg-blue-500/10 text-blue-400' : 'border-border text-foreground'}">Public</button>
          </div>
        </div>
      </div>
      <div class="mt-5 flex gap-2">
        <button onclick={() => { showCreate = false; }} class="flex-1 rounded-lg border border-border py-2.5 text-sm text-foreground hover:text-foreground transition">Cancel</button>
        <button onclick={createGroup} disabled={creating || !createName.trim()} class="flex-1 rounded-lg bg-primary py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50">
          {creating ? 'Creating...' : 'Create'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Join by code modal -->
{#if showJoin}
  <div class="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/60 backdrop-blur-sm" onclick={() => { showJoin = false; joinError = ''; }} role="presentation">
    <div class="w-full max-w-sm rounded-t-2xl sm:rounded-2xl border border-border/60 bg-secondary p-6 pb-8" onclick={(e) => e.stopPropagation()} role="dialog">
      <h2 class="mb-4 text-base font-bold text-foreground">Join a Group</h2>
      <label class="mb-1.5 block text-xs font-medium text-foreground">Invite code</label>
      <input bind:value={joinCode} type="text" placeholder="e.g. a3f92c" class="w-full rounded-lg border border-border bg-secondary px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none mb-2" />
      {#if joinError}<p class="text-xs text-red-400 mb-3">{joinError}</p>{/if}
      <div class="flex gap-2 mt-4">
        <button onclick={() => { showJoin = false; joinError = ''; }} class="flex-1 rounded-lg border border-border py-2.5 text-sm text-foreground hover:text-foreground transition">Cancel</button>
        <button onclick={joinByCode} disabled={joining || !joinCode.trim()} class="flex-1 rounded-lg bg-primary py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50">
          {joining ? 'Joining...' : 'Join'}
        </button>
      </div>
    </div>
  </div>
{/if}
