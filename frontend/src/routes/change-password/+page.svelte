<script lang="ts">
  import Logo from '$components/Logo.svelte';
  import PasswordStrength from '$components/PasswordStrength.svelte';
  import { page } from '$app/state';
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

  const isSetup = $derived(page.url.searchParams.get('setup') === 'true');

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
    if (!/[!@#$%^&*()\-_=+\[\]{}|;':",./<>?]/.test(newPassword)) {
      error = 'Password must contain at least one symbol.';
      return;
    }
    loading = true;
    try {
      await api.put('/auth/password', {
        old_password: isSetup ? '' : oldPassword,
        new_password: newPassword
      });
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
  <div class="flex min-h-screen flex-col items-center justify-center bg-background px-4">
    <div class="w-full max-w-sm">
      <!-- Logo -->
      <div class="mb-8 flex flex-col items-center gap-3">
        <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/10 ring-1 ring-ring/20">
          <Logo size={32} />
        </div>
        <div class="text-center">
          <h1 class="text-xl font-bold text-foreground">
            {isSetup ? 'Create your password' : 'Set a new password'}
          </h1>
          <p class="mt-1 text-sm text-muted-foreground">
            {isSetup
              ? 'Welcome to Joules. Set a strong password to get started.'
              : 'For security, please choose a new password before continuing.'}
          </p>
        </div>
      </div>

      <div class="rounded-2xl border border-border bg-accent/50 p-6 shadow-2xl">
        <div class="space-y-4">
          {#if !isSetup}
          <div>
            <label for="old-pw" class="mb-1.5 block text-xs font-medium text-foreground">Current password</label>
            <input
              id="old-pw"
              type="password"
              bind:value={oldPassword}
              placeholder="Enter your current password"
              class="w-full rounded-xl border border-border bg-secondary px-4 py-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
          {/if}
          <div>
            <label for="new-pw" class="mb-1.5 block text-xs font-medium text-foreground">New password</label>
            <input
              id="new-pw"
              type="password"
              bind:value={newPassword}
              placeholder="At least 8 characters"
              class="w-full rounded-xl border border-border bg-secondary px-4 py-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
            />
            <PasswordStrength password={newPassword} />
          </div>
          <div>
            <label for="confirm-pw" class="mb-1.5 block text-xs font-medium text-foreground">Confirm new password</label>
            <input
              id="confirm-pw"
              type="password"
              bind:value={confirmPassword}
              placeholder="Repeat new password"
              class="w-full rounded-xl border border-border bg-secondary px-4 py-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
              onkeydown={(e) => e.key === 'Enter' && handleSubmit()}
            />
          </div>

          {#if error}
            <div class="rounded-xl border border-destructive/20 bg-destructive/10 px-4 py-3 text-sm text-destructive">
              {error}
            </div>
          {/if}

          <button
            type="button"
            onclick={handleSubmit}
            disabled={loading || (!isSetup && !oldPassword) || !newPassword || !confirmPassword}
            class="w-full rounded-xl bg-primary px-4 py-3 text-sm font-semibold text-primary-foreground hover:bg-primary/80 disabled:opacity-50 disabled:cursor-not-allowed transition"
          >
            {loading ? 'Updating…' : 'Change Password'}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
