<script lang="ts">
  import Logo from '$components/Logo.svelte';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';

  let oldPassword = $state('');
  let newPassword = $state('');
  let confirmPassword = $state('');
  let loading = $state(false);
  let error = $state('');
  let authenticated = $state(false);

  onMount(() => {
    const unsub = authToken.subscribe(token => {
      if (!token) goto('/login');
      authenticated = !!token;
    });
    return unsub;
  });

  async function handleSubmit() {
    error = '';
    if (newPassword.length < 8) {
      error = 'New password must be at least 8 characters.';
      return;
    }
    if (newPassword !== confirmPassword) {
      error = 'New passwords do not match.';
      return;
    }
    loading = true;
    try {
      await api.put('/auth/password', { old_password: oldPassword, new_password: newPassword });
      // Clear the must_change_password flag in localStorage if stored
      const profile = JSON.parse(localStorage.getItem('user_profile') || '{}');
      profile.must_change_password = false;
      localStorage.setItem('user_profile', JSON.stringify(profile));
      goto('/dashboard');
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to change password.';
    } finally {
      loading = false;
    }
  }
</script>

{#if authenticated}
  <div class="flex min-h-screen flex-col items-center justify-center bg-slate-950 px-4">
    <div class="w-full max-w-sm">
      <!-- Logo -->
      <div class="mb-8 flex flex-col items-center gap-3">
        <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-joule-500/10 ring-1 ring-joule-500/20">
          <Logo size={32} />
        </div>
        <div class="text-center">
          <h1 class="text-xl font-bold text-white">Set a new password</h1>
          <p class="mt-1 text-sm text-slate-400">For security, please choose a new password before continuing.</p>
        </div>
      </div>

      <div class="rounded-2xl border border-white/5 bg-white/5 p-6 shadow-2xl">
        <div class="space-y-4">
          <div>
            <label for="old-pw" class="mb-1.5 block text-xs font-medium text-slate-400">Current password</label>
            <input
              id="old-pw"
              type="password"
              bind:value={oldPassword}
              placeholder="Enter your current password"
              class="w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
            />
          </div>
          <div>
            <label for="new-pw" class="mb-1.5 block text-xs font-medium text-slate-400">New password</label>
            <input
              id="new-pw"
              type="password"
              bind:value={newPassword}
              placeholder="At least 8 characters"
              class="w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
            />
          </div>
          <div>
            <label for="confirm-pw" class="mb-1.5 block text-xs font-medium text-slate-400">Confirm new password</label>
            <input
              id="confirm-pw"
              type="password"
              bind:value={confirmPassword}
              placeholder="Repeat new password"
              class="w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
              onkeydown={(e) => e.key === 'Enter' && handleSubmit()}
            />
          </div>

          {#if error}
            <div class="rounded-xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm text-red-400">
              {error}
            </div>
          {/if}

          <button
            type="button"
            onclick={handleSubmit}
            disabled={loading || !oldPassword || !newPassword || !confirmPassword}
            class="w-full rounded-xl bg-joule-500 px-4 py-3 text-sm font-semibold text-slate-900 hover:bg-joule-400 disabled:opacity-50 disabled:cursor-not-allowed transition"
          >
            {loading ? 'Updating…' : 'Change Password'}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
