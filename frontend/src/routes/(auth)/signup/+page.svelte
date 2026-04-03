<script lang="ts">
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';

  let email = $state('');
  let password = $state('');
  let confirmPassword = $state('');
  let code = $state('');
  let step = $state<'form' | 'verify'>('form');
  let error = $state('');
  let loading = $state(false);
  let passwordErrors = $state<string[]>([]);

  function validatePassword(pw: string): string[] {
    const errors: string[] = [];
    if (pw.length < 8) errors.push('At least 8 characters');
    if (!/[A-Z]/.test(pw)) errors.push('One uppercase letter');
    if (!/[0-9]/.test(pw)) errors.push('One number');
    return errors;
  }

  function onPasswordInput() {
    passwordErrors = validatePassword(password);
  }

  async function handleSignup(e: Event) {
    e.preventDefault();
    error = '';

    if (password !== confirmPassword) {
      error = 'Passwords do not match';
      return;
    }

    const pwErrors = validatePassword(password);
    if (pwErrors.length > 0) {
      error = pwErrors[0];
      return;
    }

    loading = true;
    try {
      await api.post('/auth/signup', { email, password });
      step = 'verify';
    } catch (err) {
      error = err instanceof Error ? err.message : 'Signup failed';
    } finally {
      loading = false;
    }
  }

  async function handleVerify(e: Event) {
    e.preventDefault();
    error = '';
    loading = true;

    try {
      await api.post('/auth/verify', { email, code });
      goto('/login');
    } catch (err) {
      error = err instanceof Error ? err.message : 'Verification failed';
    } finally {
      loading = false;
    }
  }

  function passwordStrength(): { label: string; color: string; width: string } {
    const checks = [
      password.length >= 8,
      /[A-Z]/.test(password),
      /[0-9]/.test(password),
      /[^A-Za-z0-9]/.test(password)
    ];
    const score = checks.filter(Boolean).length;
    if (score <= 1) return { label: 'Weak', color: 'bg-red-500', width: 'w-1/4' };
    if (score === 2) return { label: 'Fair', color: 'bg-orange-500', width: 'w-2/4' };
    if (score === 3) return { label: 'Good', color: 'bg-primary', width: 'w-3/4' };
    return { label: 'Strong', color: 'bg-emerald-500', width: 'w-full' };
  }
</script>

<div>
  {#if step === 'form'}
    <h2 class="text-2xl font-bold text-foreground">Create your account</h2>
    <p class="mt-1 text-sm text-foreground">Start tracking your nutrition with Joules</p>

    <form onsubmit={handleSignup} class="mt-8 space-y-5">
      <div>
        <label for="signup-email" class="mb-1.5 block text-sm font-medium text-foreground">Email</label>
        <input
          id="signup-email"
          type="email"
          bind:value={email}
          required
          autocomplete="email"
          placeholder="you@example.com"
          class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
        />
      </div>

      <div>
        <label for="signup-password" class="mb-1.5 block text-sm font-medium text-foreground">Password</label>
        <input
          id="signup-password"
          type="password"
          bind:value={password}
          required
          autocomplete="new-password"
          placeholder="Create a strong password"
          oninput={onPasswordInput}
          class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
        />
        {#if password}
          <div class="mt-2">
            <div class="flex gap-1">
              <div class="h-1 flex-1 rounded-full bg-accent">
                <div class="{passwordStrength().color} {passwordStrength().width} h-full rounded-full transition-all"></div>
              </div>
            </div>
            <p class="mt-1 text-xs text-muted-foreground">{passwordStrength().label}</p>
          </div>
        {/if}
      </div>

      <div>
        <label for="signup-confirm" class="mb-1.5 block text-sm font-medium text-foreground">Confirm Password</label>
        <input
          id="signup-confirm"
          type="password"
          bind:value={confirmPassword}
          required
          autocomplete="new-password"
          placeholder="Confirm your password"
          class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
        />
      </div>

      {#if error}
        <div class="rounded-lg border border-red-500/20 bg-red-500/10 px-3.5 py-2.5 text-sm text-red-400">
          {error}
        </div>
      {/if}

      <button
        type="submit"
        disabled={loading || !email || !password || !confirmPassword}
        class="w-full rounded-lg bg-primary px-3.5 py-2.5 text-sm font-semibold text-primary-foreground transition hover:bg-primary/80 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50"
      >
        {loading ? 'Creating account...' : 'Create account'}
      </button>
    </form>

    <p class="mt-6 text-center text-sm text-foreground">
      Already have an account?
      <a href="/login" class="font-medium text-primary hover:text-primary/80">Sign in</a>
    </p>
  {:else}
    <h2 class="text-2xl font-bold text-foreground">Verify your email</h2>
    <p class="mt-1 text-sm text-foreground">
      We sent a 6-digit code to <span class="text-foreground">{email}</span>
    </p>

    <div class="mt-4 rounded-lg border border-primary/20 bg-primary/10 px-3.5 py-2.5 text-xs text-primary/80">
      {#if typeof window !== 'undefined'}
        {#await import('$lib/api').then(() => false)}
          <p>If email is not configured, check <code class="rounded bg-accent px-1.5 py-0.5 text-xs">docker compose logs joule</code> for your verification code.</p>
        {/await}
      {:else}
        <p>If email is not configured, check <code class="rounded bg-accent px-1.5 py-0.5 text-xs">docker compose logs joule</code> for your verification code.</p>
      {/if}
    </div>

    <form onsubmit={handleVerify} class="mt-6 space-y-5">
      <div>
        <label for="verify-code" class="mb-1.5 block text-sm font-medium text-foreground">Verification Code</label>
        <input
          id="verify-code"
          type="text"
          bind:value={code}
          required
          maxlength="6"
          inputmode="numeric"
          placeholder="000000"
          autocomplete="one-time-code"
          class="w-full rounded-lg border border-border bg-card px-3.5 py-2.5 text-center text-lg font-mono tracking-[0.5em] text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
        />
      </div>

      {#if error}
        <div class="rounded-lg border border-red-500/20 bg-red-500/10 px-3.5 py-2.5 text-sm text-red-400">
          {error}
        </div>
      {/if}

      <button
        type="submit"
        disabled={loading || code.length !== 6}
        class="w-full rounded-lg bg-primary px-3.5 py-2.5 text-sm font-semibold text-primary-foreground transition hover:bg-primary/80 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50"
      >
        {loading ? 'Verifying...' : 'Verify'}
      </button>
    </form>

    <button
      onclick={() => { step = 'form'; code = ''; error = ''; }}
      class="mt-4 w-full text-sm text-foreground hover:text-foreground transition"
    >
      Back to signup
    </button>
  {/if}
</div>
