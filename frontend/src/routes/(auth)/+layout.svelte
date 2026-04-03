<script lang="ts">
  import Logo from '$components/Logo.svelte';

  let { children } = $props();
</script>

<div class="flex min-h-screen">
  <div
    class="hidden lg:flex lg:w-1/2 lg:flex-col lg:items-center lg:justify-center bg-background p-12 relative overflow-hidden"
  >
    <!-- Ambient glow -->
    <div class="absolute top-1/3 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[500px] h-[500px] bg-primary/5 rounded-full blur-3xl pointer-events-none"></div>
    <div class="relative max-w-sm text-center z-10">
      <div class="mb-8 flex justify-center">
        <div class="flex h-20 w-20 items-center justify-center rounded-2xl bg-primary/10 ring-1 ring-ring/20">
          <Logo size={48} />
        </div>
      </div>
      <h1 class="text-4xl font-bold tracking-tight text-foreground">Joules</h1>
      <p class="mt-3 text-[15px] text-foreground leading-relaxed">
        AI-powered calorie tracking. Snap a photo, get instant nutrition data.
      </p>
      <div class="mt-10 flex flex-col gap-3.5">
        <div class="flex items-center gap-3.5 text-left rounded-xl bg-white/[0.03] border border-border px-4 py-3">
          <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary ring-1 ring-ring/20">
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
          </span>
          <span class="text-sm text-foreground">Photo-based food logging</span>
        </div>
        <div class="flex items-center gap-3.5 text-left rounded-xl bg-white/[0.03] border border-border px-4 py-3">
          <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary ring-1 ring-ring/20">
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" /></svg>
          </span>
          <span class="text-sm text-foreground">Smart calorie & macro tracking</span>
        </div>
        <div class="flex items-center gap-3.5 text-left rounded-xl bg-white/[0.03] border border-border px-4 py-3">
          <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary ring-1 ring-ring/20">
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" /></svg>
          </span>
          <span class="text-sm text-foreground">AI health coach & motivation</span>
        </div>
        <div class="flex items-center gap-3.5 text-left rounded-xl bg-white/[0.03] border border-border px-4 py-3">
          <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary ring-1 ring-ring/20">
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" /></svg>
          </span>
          <span class="text-sm text-foreground">Personalized diet plans</span>
        </div>
      </div>
    </div>
  </div>

  <div class="flex w-full flex-col items-center justify-center p-6 lg:w-1/2">
    <div class="w-full max-w-sm">
      <div class="mb-8 flex items-center gap-3 lg:justify-center">
        <Logo size={36} />
        <span class="text-xl font-bold text-foreground">Joules</span>
      </div>

      {@render children()}
    </div>
  </div>
</div>
