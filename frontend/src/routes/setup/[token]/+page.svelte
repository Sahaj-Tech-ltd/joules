<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';

  let status = $state<'loading' | 'error'>('loading');
  let errorMsg = $state('');

  onMount(async () => {
    const token = page.params.token;
    try {
      const resp = await api.post<{ access_token: string; expires_at: string; must_change_password?: boolean }>(
        '/auth/setup-complete',
        { token }
      );
      authToken.set(resp.access_token);
      goto('/change-password?setup=true');
    } catch {
      status = 'error';
      errorMsg = 'This setup link is invalid or has already been used.';
    }
  });
</script>

<div class="flex min-h-screen items-center justify-center bg-background px-4">
  <div class="w-full max-w-sm text-center">
    {#if status === 'loading'}
      <div class="mx-auto mb-4 h-10 w-10 animate-spin rounded-full border-4 border-primary/20 border-t-primary"></div>
      <p class="text-muted-foreground">Setting up your account…</p>
    {:else}
      <p class="text-destructive">{errorMsg}</p>
      <a href="/login" class="mt-4 inline-block text-sm text-primary underline">Back to login</a>
    {/if}
  </div>
</div>
