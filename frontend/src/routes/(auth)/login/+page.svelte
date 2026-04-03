<script lang="ts">
  import { goto } from '$app/navigation';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';

  let email = $state('');
  let password = $state('');
  let error = $state('');
  let loading = $state(false);

  let isPendingApproval = $derived(error.toLowerCase().includes('pending'));

  async function handleSubmit(e: Event) {
    e.preventDefault();
    error = '';
    loading = true;

    try {
      const resp = await api.post<{ access_token: string; expires_at: string; must_change_password?: boolean }>('/auth/login', {
        email,
        password
      });
      authToken.set(resp.access_token);
      if (resp.must_change_password) {
        goto('/change-password');
      } else {
        goto('/dashboard');
      }
    } catch (err) {
      error = err instanceof Error ? err.message : 'Login failed';
    } finally {
      loading = false;
    }
  }
</script>

<div>
  <h2 class="text-2xl font-bold text-foreground">Welcome back</h2>
  <p class="mt-1 text-sm text-foreground">Sign in to your Joules account</p>

  <form onsubmit={handleSubmit} class="mt-8 space-y-5">
    <div>
      <label for="email" class="mb-1.5 block text-sm font-medium text-foreground">Email</label>
      <input
        id="email"
        type="email"
        bind:value={email}
        required
        autocomplete="email"
        placeholder="you@example.com"
        class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
      />
    </div>

    <div>
      <label for="password" class="mb-1.5 block text-sm font-medium text-foreground">Password</label>
      <input
        id="password"
        type="password"
        bind:value={password}
        required
        autocomplete="current-password"
        placeholder="Enter your password"
        class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
      />
    </div>

    {#if error}
      <div class="rounded-lg border px-3.5 py-2.5 text-sm
        {isPendingApproval
          ? 'border-amber-500/20 bg-amber-500/10 text-amber-400'
          : 'border-red-500/20 bg-red-500/10 text-red-400'}">
        {error}
        {#if isPendingApproval}
          <p class="mt-1 text-xs opacity-75">Contact the server admin to get access.</p>
        {/if}
      </div>
    {/if}

    <button
      type="submit"
      disabled={loading || !email || !password}
      class="w-full rounded-lg bg-primary px-3.5 py-2.5 text-sm font-semibold text-primary-foreground transition hover:bg-primary/80 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50"
    >
      {loading ? 'Signing in...' : 'Sign in'}
    </button>
  </form>

  <p class="mt-6 text-center text-sm text-foreground">
    Don't have an account?
    <a href="/signup" class="font-medium text-primary hover:text-primary/80">Sign up</a>
  </p>
</div>
