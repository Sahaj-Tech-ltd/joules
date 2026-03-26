<script lang="ts">
  import { goto } from '$app/navigation';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';

  let email = $state('');
  let password = $state('');
  let error = $state('');
  let loading = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    error = '';
    loading = true;

    try {
      const resp = await api.post<{ access_token: string; expires_at: string }>('/auth/login', {
        email,
        password
      });
      authToken.set(resp.access_token);
      goto('/dashboard');
    } catch (err) {
      error = err instanceof Error ? err.message : 'Login failed';
    } finally {
      loading = false;
    }
  }
</script>

<div>
  <h2 class="text-2xl font-bold text-white">Welcome back</h2>
  <p class="mt-1 text-sm text-slate-400">Sign in to your Joule account</p>

  <form onsubmit={handleSubmit} class="mt-8 space-y-5">
    <div>
      <label for="email" class="mb-1.5 block text-sm font-medium text-slate-300">Email</label>
      <input
        id="email"
        type="email"
        bind:value={email}
        required
        autocomplete="email"
        placeholder="you@example.com"
        class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
      />
    </div>

    <div>
      <label for="password" class="mb-1.5 block text-sm font-medium text-slate-300">Password</label>
      <input
        id="password"
        type="password"
        bind:value={password}
        required
        autocomplete="current-password"
        placeholder="Enter your password"
        class="w-full rounded-lg border border-slate-700 bg-surface-light px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500"
      />
    </div>

    {#if error}
      <div class="rounded-lg border border-red-500/20 bg-red-500/10 px-3.5 py-2.5 text-sm text-red-400">
        {error}
      </div>
    {/if}

    <button
      type="submit"
      disabled={loading || !email || !password}
      class="w-full rounded-lg bg-joule-500 px-3.5 py-2.5 text-sm font-semibold text-slate-900 transition hover:bg-joule-400 focus:outline-none focus:ring-2 focus:ring-joule-500 focus:ring-offset-2 focus:ring-offset-slate-900 disabled:cursor-not-allowed disabled:opacity-50"
    >
      {loading ? 'Signing in...' : 'Sign in'}
    </button>
  </form>

  <p class="mt-6 text-center text-sm text-slate-400">
    Don't have an account?
    <a href="/signup" class="font-medium text-joule-400 hover:text-joule-300">Sign up</a>
  </p>
</div>
