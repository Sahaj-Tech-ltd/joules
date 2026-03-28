<script lang="ts">
  import Sidebar from '$components/Sidebar.svelte';
  import ThemeToggle from '$components/ThemeToggle.svelte';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';

  interface UserRow {
    id: string;
    email: string;
    verified: boolean;
    approved: boolean;
    is_admin: boolean;
    created_at: string;
  }

  interface Settings {
    require_approval: boolean;
    ai_provider: string;
    ai_model: string;
    smtp_configured: boolean;
    app_url: string;
    port: string;
  }

  let users = $state<UserRow[]>([]);
  let settings = $state<Settings>({
    require_approval: false,
    ai_provider: 'openai',
    ai_model: '',
    smtp_configured: false,
    app_url: '',
    port: '',
  });
  let loading = $state(true);
  let currentUserID = $state('');
  let actionLoading = $state<string | null>(null);
  let settingsLoading = $state(false);
  let aiSaving = $state(false);
  let aiSaved = $state(false);

  // Editable AI settings (local state)
  let editAIProvider = $state('openai');
  let editAIModel = $state('');

  // Restart modal
  let showRestartModal = $state(false);
  let restarting = $state(false);

  onMount(() => {
    const unsub = authToken.subscribe((token) => {
      if (!token) {
        goto('/login');
        return;
      }
    });

    (async () => {
      try {
        // Check if current user is admin
        const profile = await api.get<{ name: string; is_admin: boolean; onboarding_complete: boolean }>('/user/profile');
        if (!profile.is_admin) {
          goto('/dashboard');
          return;
        }

        // Get current user ID from /auth/me
        const me = await api.get<{ id: string }>('/auth/me');
        currentUserID = me.id;

        const [usersData, settingsData, bannersData] = await Promise.all([
          api.get<UserRow[]>('/admin/users'),
          api.get<Settings>('/admin/settings'),
          api.get<Banner[]>('/admin/banners'),
        ]);
        users = usersData;
        settings = settingsData;
        editAIProvider = settingsData.ai_provider || 'openai';
        editAIModel = settingsData.ai_model || '';
        banners = bannersData ?? [];
      } catch {
        goto('/dashboard');
      } finally {
        loading = false;
      }
    })();

    return unsub;
  });

  async function approveUser(id: string) {
    actionLoading = id + ':approve';
    try {
      await api.post(`/admin/users/${id}/approve`, {});
      users = users.map(u => u.id === id ? { ...u, approved: true } : u);
    } catch {}
    finally { actionLoading = null; }
  }

  async function unapproveUser(id: string) {
    actionLoading = id + ':unapprove';
    try {
      await api.post(`/admin/users/${id}/unapprove`, {});
      users = users.map(u => u.id === id ? { ...u, approved: false } : u);
    } catch {}
    finally { actionLoading = null; }
  }

  async function deleteUser(id: string, email: string) {
    if (!confirm(`Delete user ${email}? This cannot be undone.`)) return;
    actionLoading = id + ':delete';
    try {
      await api.del(`/admin/users/${id}`);
      users = users.filter(u => u.id !== id);
    } catch {}
    finally { actionLoading = null; }
  }

  async function makeAdmin(id: string, email: string) {
    if (!confirm(`Promote ${email} to admin?`)) return;
    actionLoading = id + ':makeadmin';
    try {
      await api.post(`/admin/users/${id}/make-admin`, {});
      users = users.map(u => u.id === id ? { ...u, is_admin: true, approved: true } : u);
    } catch {}
    finally { actionLoading = null; }
  }

  async function removeAdmin(id: string, email: string) {
    if (!confirm(`Remove admin rights from ${email}?`)) return;
    actionLoading = id + ':removeadmin';
    try {
      await api.post(`/admin/users/${id}/remove-admin`, {});
      users = users.map(u => u.id === id ? { ...u, is_admin: false } : u);
    } catch {}
    finally { actionLoading = null; }
  }

  async function toggleRequireApproval() {
    settingsLoading = true;
    const newVal = !settings.require_approval;
    try {
      await api.put('/admin/settings', { require_approval: newVal });
      settings = { ...settings, require_approval: newVal };
    } catch {}
    finally { settingsLoading = false; }
  }

  async function saveAISettings() {
    aiSaving = true;
    aiSaved = false;
    try {
      await api.put('/admin/settings', {
        require_approval: settings.require_approval,
        ai_provider: editAIProvider,
        ai_model: editAIModel,
      });
      settings = { ...settings, ai_provider: editAIProvider, ai_model: editAIModel };
      aiSaved = true;
      setTimeout(() => { aiSaved = false; }, 3000);
    } catch {}
    finally { aiSaving = false; }
  }

  // --- Banners ---
  interface Banner { id: string; title: string; message: string; type: string; created_at: string; }
  let banners = $state<Banner[]>([]);
  let newBannerTitle = $state('');
  let newBannerMsg = $state('');
  let newBannerType = $state('info');
  let bannerSaving = $state(false);

  // --- Logs ---
  interface LogEntry { id: number; level: string; category: string; message: string; details?: any; created_at: string; }
  let logs = $state<LogEntry[]>([]);
  let logCategory = $state('all');
  let logsLoading = $state(false);

  // --- Verify email ---
  async function verifyUser(id: string, email: string) {
    if (!confirm(`Mark ${email}'s email as verified?`)) return;
    actionLoading = id + ':verify';
    try {
      await api.post(`/admin/users/${id}/verify`, {});
      users = users.map(u => u.id === id ? { ...u, verified: true } : u);
    } catch {}
    finally { actionLoading = null; }
  }

  async function loadBanners() {
    try {
      const data = await api.get<Banner[]>('/admin/banners');
      banners = data ?? [];
    } catch {}
  }

  async function createBanner() {
    if (!newBannerMsg.trim()) return;
    bannerSaving = true;
    try {
      const b = await api.post<Banner>('/admin/banners', { title: newBannerTitle, message: newBannerMsg, type: newBannerType });
      banners = [b, ...banners];
      newBannerTitle = ''; newBannerMsg = ''; newBannerType = 'info';
    } catch {}
    finally { bannerSaving = false; }
  }

  async function deleteBanner(id: string) {
    try {
      await api.del(`/admin/banners/${id}`);
      banners = banners.filter(b => b.id !== id);
    } catch {}
  }

  async function loadLogs() {
    logsLoading = true;
    try {
      const data = await api.get<LogEntry[]>(`/admin/logs?category=${logCategory}`);
      logs = data ?? [];
    } catch {}
    finally { logsLoading = false; }
  }

  async function confirmRestart() {
    restarting = true;
    try {
      await api.post('/admin/restart', {});
    } catch {}
    // Wait a moment then close modal — server is restarting
    setTimeout(() => {
      restarting = false;
      showRestartModal = false;
    }, 2000);
  }

  function formatDate(iso: string) {
    return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  }
</script>

<div class="flex min-h-screen">
  <Sidebar activePage="admin" isAdmin={true} />

  <main class="flex-1 overflow-y-auto p-6 pb-20 lg:p-10 lg:pb-10">
    {#if loading}
      <div class="flex h-64 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-slate-700 border-t-joule-500"></div>
      </div>
    {:else}
      <div class="flex items-center justify-between mb-8">
        <div>
          <h1 class="text-2xl font-bold text-white">Admin</h1>
          <p class="mt-1 text-sm text-slate-400">Manage users and server settings</p>
        </div>
        <div class="flex items-center gap-2">
          <ThemeToggle />
          <button
            onclick={() => { authToken.set(null); goto('/login'); }}
            class="rounded-lg border border-slate-700 px-3 py-1.5 text-sm text-slate-400 hover:text-white transition"
          >
            Sign out
          </button>
        </div>
      </div>

      <!-- Server Info -->
      <div class="mb-6 rounded-xl border border-slate-700 bg-surface-light p-5">
        <h2 class="text-sm font-semibold text-joule-400 mb-4">Server Info</h2>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <p class="text-xs text-slate-500 mb-0.5">App URL</p>
            <p class="text-sm text-white font-mono">{settings.app_url || '—'}</p>
          </div>
          <div>
            <p class="text-xs text-slate-500 mb-0.5">Port</p>
            <p class="text-sm text-white font-mono">{settings.port || '—'}</p>
          </div>
          <div>
            <p class="text-xs text-slate-500 mb-0.5">AI Provider</p>
            <p class="text-sm text-white">{settings.ai_provider || '—'}</p>
          </div>
          <div>
            <p class="text-xs text-slate-500 mb-0.5">AI Model</p>
            <p class="text-sm text-white">{settings.ai_model || '—'}</p>
          </div>
          <div>
            <p class="text-xs text-slate-500 mb-0.5">SMTP</p>
            {#if settings.smtp_configured}
              <span class="inline-flex items-center rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-400">Configured</span>
            {:else}
              <span class="inline-flex items-center rounded-full bg-slate-700/50 px-2 py-0.5 text-xs font-medium text-slate-400">Not configured</span>
            {/if}
          </div>
        </div>
      </div>

      <!-- Settings -->
      <div class="mb-6 rounded-xl border border-slate-700 bg-surface-light p-5">
        <h2 class="text-sm font-semibold text-joule-400 mb-4">Server Settings</h2>
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-white">Require Approval for New Signups</p>
            <p class="text-xs text-slate-400 mt-0.5">New accounts will be unapproved until an admin approves them</p>
          </div>
          <button
            onclick={toggleRequireApproval}
            disabled={settingsLoading}
            class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none disabled:opacity-50
              {settings.require_approval ? 'bg-joule-500' : 'bg-slate-700'}"
          >
            <span
              class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform
                {settings.require_approval ? 'translate-x-6' : 'translate-x-1'}"
            ></span>
          </button>
        </div>
      </div>

      <!-- AI Settings -->
      <div class="mb-6 rounded-xl border border-slate-700 bg-surface-light p-5">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-sm font-semibold text-joule-400">AI Settings</h2>
          <span class="inline-flex items-center gap-1.5 rounded-full bg-amber-500/10 px-2.5 py-1 text-xs font-medium text-amber-400">
            <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
            Restart required to apply
          </span>
        </div>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 mb-4">
          <div>
            <label for="ai-provider" class="mb-1.5 block text-xs font-medium text-slate-400">Provider</label>
            <select
              id="ai-provider"
              bind:value={editAIProvider}
              class="w-full rounded-lg border border-slate-700 bg-surface px-3 py-2 text-sm text-white focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
            >
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic</option>
            </select>
          </div>
          <div>
            <label for="ai-model" class="mb-1.5 block text-xs font-medium text-slate-400">Model</label>
            <input
              id="ai-model"
              type="text"
              bind:value={editAIModel}
              placeholder={editAIProvider === 'openai' ? 'gpt-4o' : 'claude-opus-4-6'}
              class="w-full rounded-lg border border-slate-700 bg-surface px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
            />
          </div>
        </div>
        <div class="flex items-center gap-3">
          {#if aiSaved}
            <span class="text-sm text-green-400">Saved! Restart to apply.</span>
          {/if}
          <button
            onclick={saveAISettings}
            disabled={aiSaving}
            class="rounded-lg bg-joule-500 px-4 py-2 text-sm font-semibold text-slate-900 hover:bg-joule-400 transition disabled:opacity-50"
          >
            {aiSaving ? 'Saving…' : 'Save AI Settings'}
          </button>
        </div>
      </div>

      <!-- Restart Server -->
      <div class="mb-8 rounded-xl border border-red-900/40 bg-surface-light p-5">
        <h2 class="text-sm font-semibold text-red-400 mb-2">Danger Zone</h2>
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-white">Restart Server</p>
            <p class="text-xs text-slate-400 mt-0.5">Docker will automatically restart the container. Users will be disconnected for ~10 seconds.</p>
          </div>
          <button
            onclick={() => { showRestartModal = true; }}
            class="rounded-lg border border-red-700 px-4 py-2 text-sm font-semibold text-red-400 hover:bg-red-500/10 transition"
          >
            Restart
          </button>
        </div>
      </div>

      <!-- Users Table -->
      <div class="rounded-xl border border-slate-700 bg-surface-light overflow-hidden">
        <div class="px-5 py-4 border-b border-slate-700">
          <h2 class="text-sm font-semibold text-joule-400">Users ({users.length})</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-slate-700">
                <th class="px-5 py-3 text-left font-medium text-slate-400">Email</th>
                <th class="px-5 py-3 text-left font-medium text-slate-400">Verified</th>
                <th class="px-5 py-3 text-left font-medium text-slate-400">Approved</th>
                <th class="px-5 py-3 text-left font-medium text-slate-400">Role</th>
                <th class="px-5 py-3 text-left font-medium text-slate-400">Joined</th>
                <th class="px-5 py-3 text-left font-medium text-slate-400">Actions</th>
              </tr>
            </thead>
            <tbody>
              {#each users as user}
                <tr class="border-b border-slate-800 last:border-0 hover:bg-slate-800/30">
                  <td class="px-5 py-3 text-white">
                    {user.email}
                    {#if user.id === currentUserID}
                      <span class="ml-1.5 text-xs text-slate-500">(you)</span>
                    {/if}
                  </td>
                  <td class="px-5 py-3">
                    {#if user.verified}
                      <svg class="h-4 w-4 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
                    {:else}
                      <svg class="h-4 w-4 text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
                    {/if}
                  </td>
                  <td class="px-5 py-3">
                    {#if user.approved}
                      <span class="inline-flex items-center rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-400">Approved</span>
                    {:else}
                      <span class="inline-flex items-center rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-400">Pending</span>
                    {/if}
                  </td>
                  <td class="px-5 py-3">
                    {#if user.is_admin}
                      <span class="inline-flex items-center rounded-full bg-joule-500/10 px-2 py-0.5 text-xs font-medium text-joule-400">Admin</span>
                    {:else}
                      <span class="text-slate-500">User</span>
                    {/if}
                  </td>
                  <td class="px-5 py-3 text-slate-400">{formatDate(user.created_at)}</td>
                  <td class="px-5 py-3">
                    {#if user.id !== currentUserID}
                      <div class="flex flex-wrap items-center gap-1.5">
                        {#if !user.is_admin}
                          {#if !user.verified}
                            <button
                              onclick={() => verifyUser(user.id, user.email)}
                              disabled={actionLoading === user.id + ':verify'}
                              class="rounded px-2 py-1 text-xs font-medium text-blue-400 hover:bg-blue-500/10 transition disabled:opacity-50"
                            >Verify Email</button>
                          {/if}
                          {#if user.approved}
                            <button
                              onclick={() => unapproveUser(user.id)}
                              disabled={actionLoading === user.id + ':unapprove'}
                              class="rounded px-2 py-1 text-xs font-medium text-amber-400 hover:bg-amber-500/10 transition disabled:opacity-50"
                            >Unapprove</button>
                          {:else}
                            <button
                              onclick={() => approveUser(user.id)}
                              disabled={actionLoading === user.id + ':approve'}
                              class="rounded px-2 py-1 text-xs font-medium text-green-400 hover:bg-green-500/10 transition disabled:opacity-50"
                            >Approve</button>
                          {/if}
                          <button
                            onclick={() => makeAdmin(user.id, user.email)}
                            disabled={actionLoading === user.id + ':makeadmin'}
                            class="rounded px-2 py-1 text-xs font-medium text-joule-400 hover:bg-joule-500/10 transition disabled:opacity-50"
                          >Make Admin</button>
                          <button
                            onclick={() => deleteUser(user.id, user.email)}
                            disabled={actionLoading === user.id + ':delete'}
                            class="rounded px-2 py-1 text-xs font-medium text-red-400 hover:bg-red-500/10 transition disabled:opacity-50"
                          >Delete</button>
                        {:else}
                          <button
                            onclick={() => removeAdmin(user.id, user.email)}
                            disabled={actionLoading === user.id + ':removeadmin'}
                            class="rounded px-2 py-1 text-xs font-medium text-slate-400 hover:bg-slate-500/10 transition disabled:opacity-50"
                          >Remove Admin</button>
                        {/if}
                      </div>
                    {:else}
                      <span class="text-xs text-slate-600">—</span>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>

      <!-- Banners -->
      <div class="mt-8 rounded-xl border border-slate-700 bg-surface-light p-5">
        <h2 class="mb-4 text-sm font-semibold text-joule-400">Announcement Banners</h2>
        <p class="mb-4 text-xs text-slate-400">Banners appear on the dashboard for all users. They can dismiss individually.</p>

        <!-- Create banner form -->
        <div class="mb-4 space-y-3 rounded-lg border border-slate-700 bg-surface p-4">
          <input
            type="text"
            bind:value={newBannerTitle}
            placeholder="Title (optional)"
            class="w-full rounded-lg border border-slate-700 bg-surface-light px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
          />
          <textarea
            bind:value={newBannerMsg}
            placeholder="Message *"
            rows="2"
            class="w-full rounded-lg border border-slate-700 bg-surface-light px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500 resize-none"
          ></textarea>
          <div class="flex items-center gap-3">
            <select
              bind:value={newBannerType}
              class="rounded-lg border border-slate-700 bg-surface-light px-3 py-2 text-sm text-white focus:border-joule-500 focus:outline-none"
            >
              <option value="info">Info</option>
              <option value="tip">Tip</option>
              <option value="warning">Warning</option>
            </select>
            <button
              onclick={createBanner}
              disabled={!newBannerMsg.trim() || bannerSaving}
              class="rounded-lg bg-joule-500 px-4 py-2 text-sm font-semibold text-slate-900 hover:bg-joule-400 disabled:opacity-50 transition"
            >{bannerSaving ? 'Posting…' : 'Post Banner'}</button>
          </div>
        </div>

        <!-- Existing banners -->
        {#if banners.length === 0}
          <p class="text-sm text-slate-500">No active banners.</p>
        {:else}
          <div class="space-y-2">
            {#each banners as banner}
              <div class="flex items-start justify-between gap-3 rounded-lg border border-slate-700 bg-surface px-4 py-3">
                <div class="min-w-0">
                  {#if banner.title}<p class="text-sm font-medium text-white">{banner.title}</p>{/if}
                  <p class="text-sm text-slate-400">{banner.message}</p>
                  <p class="mt-0.5 text-xs text-slate-600 capitalize">{banner.type} · {formatDate(banner.created_at)}</p>
                </div>
                <button
                  onclick={() => deleteBanner(banner.id)}
                  class="shrink-0 text-slate-500 hover:text-red-400 transition"
                  aria-label="Delete banner"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <!-- System Logs -->
      <div class="mt-8 rounded-xl border border-slate-700 bg-surface-light p-5">
        <div class="mb-4 flex items-center justify-between gap-3">
          <h2 class="text-sm font-semibold text-joule-400">System Logs</h2>
          <div class="flex items-center gap-2">
            <select
              bind:value={logCategory}
              class="rounded-lg border border-slate-700 bg-surface px-3 py-1.5 text-sm text-white focus:border-joule-500 focus:outline-none"
            >
              <option value="all">All</option>
              <option value="smtp">SMTP</option>
              <option value="ai">AI</option>
              <option value="auth">Auth</option>
              <option value="general">General</option>
            </select>
            <button
              onclick={loadLogs}
              disabled={logsLoading}
              class="rounded-lg border border-slate-700 px-3 py-1.5 text-sm font-medium text-slate-400 hover:text-white hover:bg-slate-800 transition disabled:opacity-50"
            >{logsLoading ? 'Loading…' : 'Load Logs'}</button>
          </div>
        </div>

        {#if logs.length === 0}
          <p class="text-sm text-slate-500">Click "Load Logs" to view recent system events.</p>
        {:else}
          <div class="overflow-x-auto">
            <table class="w-full text-xs">
              <thead>
                <tr class="border-b border-slate-700">
                  <th class="pb-2 text-left font-medium text-slate-400 pr-4">Time</th>
                  <th class="pb-2 text-left font-medium text-slate-400 pr-4">Level</th>
                  <th class="pb-2 text-left font-medium text-slate-400 pr-4">Category</th>
                  <th class="pb-2 text-left font-medium text-slate-400">Message</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-800">
                {#each logs as log}
                  <tr class="hover:bg-slate-800/30">
                    <td class="py-2 pr-4 text-slate-500 whitespace-nowrap">{new Date(log.created_at).toLocaleTimeString()}</td>
                    <td class="py-2 pr-4">
                      <span class="inline-flex rounded px-1.5 py-0.5 text-xs font-medium capitalize
                        {log.level === 'error' ? 'bg-red-500/10 text-red-400' :
                         log.level === 'warn' ? 'bg-amber-500/10 text-amber-400' :
                         'bg-blue-500/10 text-blue-400'}">{log.level}</span>
                    </td>
                    <td class="py-2 pr-4 text-slate-400 capitalize">{log.category}</td>
                    <td class="py-2 text-slate-300">{log.message}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    {/if}
  </main>
</div>

<!-- Restart Confirmation Modal -->
{#if showRestartModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
    <div class="mx-4 w-full max-w-md rounded-2xl border border-slate-700 bg-slate-900 p-6 shadow-2xl">
      <div class="mb-4 flex items-center gap-3">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-red-500/10">
          <svg class="h-5 w-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        </div>
        <h3 class="text-lg font-semibold text-white">Restart Server?</h3>
      </div>
      <p class="mb-6 text-sm text-slate-400">
        This will restart the server container. All active users will be disconnected for approximately 10 seconds while Docker restarts the service.
      </p>
      <div class="flex gap-3">
        <button
          onclick={() => { showRestartModal = false; }}
          disabled={restarting}
          class="flex-1 rounded-xl border border-slate-700 px-4 py-2.5 text-sm font-semibold text-slate-400 hover:text-white transition disabled:opacity-50"
        >
          Cancel
        </button>
        <button
          onclick={confirmRestart}
          disabled={restarting}
          class="flex-1 rounded-xl bg-red-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-red-500 transition disabled:opacity-50"
        >
          {restarting ? 'Restarting…' : 'Restart Server'}
        </button>
      </div>
    </div>
  </div>
{/if}
