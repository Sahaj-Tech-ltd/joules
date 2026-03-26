<script lang="ts">
  import Logo from '$components/Logo.svelte';

  let { children } = $props();
</script>

<div class="flex min-h-screen">
  <div
    class="hidden lg:flex lg:w-1/2 lg:flex-col lg:items-center lg:justify-center bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 p-12"
  >
    <div class="max-w-md text-center">
      <div class="mb-8 flex justify-center">
        <Logo size={80} />
      </div>
      <h1 class="text-4xl font-bold tracking-tight text-white">Joule</h1>
      <p class="mt-4 text-lg text-slate-400">
        AI-powered calorie tracking. Snap a photo, get instant nutrition data.
        Reach your goals with personalized coaching.
      </p>
      <div class="mt-8 flex flex-col gap-3 text-sm text-slate-500">
        <div class="flex items-center gap-3">
          <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-joule-500/10 text-joule-400">
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
          </span>
          Photo-based food logging
        </div>
        <div class="flex items-center gap-3">
          <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-joule-500/10 text-joule-400">
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" /></svg>
          </span>
          Smart calorie & macro tracking
        </div>
        <div class="flex items-center gap-3">
          <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-joule-500/10 text-joule-400">
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" /></svg>
          </span>
          Personalized diet plans
        </div>
        <div class="flex items-center gap-3">
          <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-joule-500/10 text-joule-400">
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" /></svg>
          </span>
          AI health coach & motivation
        </div>
      </div>
    </div>
  </div>

  <div class="flex w-full flex-col items-center justify-center p-6 lg:w-1/2">
    <div class="w-full max-w-sm">
      <div class="mb-8 flex items-center gap-3 lg:justify-center">
        <Logo size={36} />
        <span class="text-xl font-bold text-white">Joule</span>
      </div>

      {@render children()}
    </div>
  </div>
</div>
