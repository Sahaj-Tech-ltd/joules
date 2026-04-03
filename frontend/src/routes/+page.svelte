<script lang="ts">
  import { goto } from '$app/navigation';
  import { authToken } from '$lib/stores';
  import { onMount } from 'svelte';
  import Logo from '$components/Logo.svelte';

  onMount(() => {
    const unsub = authToken.subscribe((token) => {
      if (token) {
        const profile = JSON.parse(localStorage.getItem('user_profile') || '{}');
        if (!profile.onboarding_complete) {
          goto('/onboarding');
        } else {
          goto('/dashboard');
        }
      }
    });
    return unsub;
  });
</script>

<div class="min-h-screen bg-background text-foreground">

  <!-- NAV -->
  <nav class="border-b border-border bg-background/80 backdrop-blur-md sticky top-0 z-50">
    <div class="max-w-6xl mx-auto px-4 py-4 flex items-center justify-between">
      <a href="/" class="flex items-center gap-2.5">
        <Logo size={32} />
        <span class="text-xl font-bold tracking-tight">Joules</span>
      </a>
      <div class="flex items-center gap-3">
        <a
          href="/login"
          class="px-4 py-2 text-sm font-medium text-foreground hover:text-foreground transition-colors"
        >
          Sign In
        </a>
        <a
          href="/signup"
          class="px-4 py-2 text-sm font-semibold rounded-xl bg-amber-500 hover:bg-amber-400 text-foreground transition-colors"
        >
          Get Started
        </a>
      </div>
    </div>
  </nav>

  <!-- HERO -->
  <section class="max-w-6xl mx-auto px-4 pt-24 pb-20 text-center">
    <div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-amber-500/30 bg-amber-500/10 text-amber-400 text-xs font-medium mb-8">
      <svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
        <path fill-rule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clip-rule="evenodd"/>
      </svg>
      AI-powered nutrition tracking
    </div>

    <h1 class="text-5xl sm:text-6xl lg:text-7xl font-extrabold tracking-tight mb-6 leading-tight">
      Track smarter,<br />
      <span class="text-amber-500">not harder.</span>
    </h1>

    <p class="text-lg sm:text-xl text-foreground max-w-2xl mx-auto mb-4 leading-relaxed">
      Joules uses AI to analyze your meals from photos — just snap, tap, done.
    </p>
    <p class="text-sm text-muted-foreground max-w-xl mx-auto mb-12">
      Science-backed calorie and macro tracking built on clinical research, not guesswork.
    </p>

    <div class="flex flex-col sm:flex-row items-center justify-center gap-4">
      <a
        href="/signup"
        class="w-full sm:w-auto px-8 py-3.5 text-base font-semibold rounded-2xl bg-amber-500 hover:bg-amber-400 text-foreground transition-all shadow-lg shadow-amber-500/20 hover:shadow-amber-500/30 hover:-translate-y-0.5"
      >
        Get Started — it's free
      </a>
      <a
        href="/login"
        class="w-full sm:w-auto px-8 py-3.5 text-base font-medium rounded-2xl border border-border hover:border-border text-foreground hover:text-foreground bg-accent/50 hover:bg-accent/50 transition-all"
      >
        Sign In
      </a>
    </div>

    <!-- Hero visual hint -->
    <div class="mt-20 relative">
      <div class="absolute inset-0 bg-gradient-to-t from-background via-transparent to-transparent z-10 pointer-events-none rounded-3xl"></div>
      <div class="grid grid-cols-3 gap-3 max-w-sm mx-auto opacity-60">
        {#each ['Protein', 'Carbs', 'Fat'] as macro, i}
          <div class="rounded-2xl border border-border bg-accent/50 p-4 text-center">
            <div class="text-2xl font-bold {i === 0 ? 'text-amber-400' : i === 1 ? 'text-sky-400' : 'text-rose-400'}">
              {i === 0 ? '142g' : i === 1 ? '210g' : '68g'}
            </div>
            <div class="text-xs text-muted-foreground mt-1">{macro}</div>
          </div>
        {/each}
      </div>
    </div>
  </section>

  <!-- FEATURES -->
  <section class="max-w-6xl mx-auto px-4 py-20">
    <div class="text-center mb-14">
      <h2 class="text-3xl sm:text-4xl font-bold mb-3">Everything you need. Nothing you don't.</h2>
      <p class="text-foreground text-base max-w-xl mx-auto">Built for people who want real results, not another app collecting dust.</p>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <!-- AI Photo Analysis -->
      <div class="rounded-2xl border border-border bg-accent/50 p-6 flex flex-col gap-4">
        <div class="w-10 h-10 rounded-xl bg-amber-500/15 flex items-center justify-center text-amber-400">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z"/>
            <path stroke-linecap="round" stroke-linejoin="round" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
        </div>
        <div>
          <h3 class="font-semibold text-foreground mb-2">AI Photo Analysis</h3>
          <p class="text-sm text-foreground leading-relaxed">Point your camera at any meal. Our AI identifies every ingredient and estimates macros instantly.</p>
        </div>
      </div>

      <!-- Science-backed Macros -->
      <div class="rounded-2xl border border-border bg-accent/50 p-6 flex flex-col gap-4">
        <div class="w-10 h-10 rounded-xl bg-amber-500/15 flex items-center justify-center text-amber-400">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
          </svg>
        </div>
        <div>
          <h3 class="font-semibold text-foreground mb-2">Science-backed Macros</h3>
          <p class="text-sm text-foreground leading-relaxed">Targets calculated from your BMR using Mifflin-St Jeor, adjusted for activity level and goals.</p>
        </div>
      </div>

      <!-- Smart Health Coach -->
      <div class="rounded-2xl border border-border bg-accent/50 p-6 flex flex-col gap-4">
        <div class="w-10 h-10 rounded-xl bg-amber-500/15 flex items-center justify-center text-amber-400">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"/>
          </svg>
        </div>
        <div>
          <h3 class="font-semibold text-foreground mb-2">Smart Health Coach</h3>
          <p class="text-sm text-foreground leading-relaxed">Chat with an AI coach that knows your history, tracks trends, and helps you hit your goals.</p>
        </div>
      </div>

      <!-- Gamification -->
      <div class="rounded-2xl border border-border bg-accent/50 p-6 flex flex-col gap-4">
        <div class="w-10 h-10 rounded-xl bg-amber-500/15 flex items-center justify-center text-amber-400">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z"/>
          </svg>
        </div>
        <div>
          <h3 class="font-semibold text-foreground mb-2">Gamification</h3>
          <p class="text-sm text-foreground leading-relaxed">Achievements, streaks, and confetti — staying consistent should be rewarding.</p>
        </div>
      </div>
    </div>
  </section>

  <!-- HOW IT WORKS -->
  <section class="max-w-6xl mx-auto px-4 py-20">
    <div class="text-center mb-14">
      <h2 class="text-3xl sm:text-4xl font-bold mb-3">Up and running in minutes.</h2>
      <p class="text-foreground text-base max-w-xl mx-auto">No spreadsheets. No guessing. Just a clear path to your goal.</p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 relative">
      <!-- Connector line (desktop) -->
      <div class="hidden md:block absolute top-8 left-1/3 right-1/3 h-px bg-gradient-to-r from-transparent via-amber-500/30 to-transparent"></div>

      {#each [
        {
          step: '01',
          title: 'Set your goals',
          desc: 'Tell Joules your stats — age, weight, height, activity level, and target. We calculate your TDEE and personalised macro split using the Mifflin-St Jeor equation.',
          icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2'
        },
        {
          step: '02',
          title: 'Log meals with AI',
          desc: 'Snap a photo of your plate or type a quick description. The AI breaks down every ingredient, estimates portions, and logs calories and macros automatically.',
          icon: 'M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z M15 13a3 3 0 11-6 0 3 3 0 016 0z'
        },
        {
          step: '03',
          title: 'Track & adapt',
          desc: "Review your dashboard, chat with your AI coach, and watch trends over time. Adjust your plan as your body and goals evolve — Joules adapts with you.",
          icon: 'M13 7h8m0 0v8m0-8l-8 8-4-4-6 6'
        }
      ] as item}
        <div class="rounded-2xl border border-border bg-accent/50 p-6 flex flex-col gap-4 relative">
          <div class="flex items-start gap-4">
            <div class="w-10 h-10 shrink-0 rounded-xl bg-amber-500/15 flex items-center justify-center text-amber-400">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d={item.icon}/>
              </svg>
            </div>
            <span class="text-4xl font-black text-foreground/5 leading-none">{item.step}</span>
          </div>
          <h3 class="font-semibold text-foreground text-lg">{item.title}</h3>
          <p class="text-sm text-foreground leading-relaxed">{item.desc}</p>
        </div>
      {/each}
    </div>
  </section>

  <!-- SCIENCE -->
  <section class="max-w-6xl mx-auto px-4 py-20">
    <div class="rounded-2xl border border-amber-500/15 bg-amber-500/5 p-8 md:p-12">
      <div class="flex flex-col md:flex-row gap-8 items-start">
        <div class="shrink-0">
          <div class="w-12 h-12 rounded-2xl bg-amber-500/20 flex items-center justify-center text-amber-400">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"/>
            </svg>
          </div>
        </div>
        <div>
          <div class="text-xs font-semibold uppercase tracking-widest text-amber-400 mb-3">The Science Behind Joules</div>
          <h2 class="text-2xl sm:text-3xl font-bold mb-4">Clinically validated, not algorithmically guessed.</h2>
          <p class="text-foreground leading-relaxed max-w-2xl">
            Joules calculates your TDEE using the <span class="text-foreground font-medium">Mifflin-St Jeor equation</span>, the gold standard validated in clinical research as the most accurate BMR formula for the general population. Combined with macronutrient targets tailored to your specific goal — cut, bulk, or maintain — you get a truly personalised plan. Not a one-size-fits-all calorie number, but protein, carb, and fat targets calibrated to how your body actually works.
          </p>
          <div class="mt-6 flex flex-wrap gap-3">
            {#each ['Mifflin-St Jeor BMR', 'Activity multipliers', 'Goal-adjusted macros', 'Progressive tracking'] as tag}
              <span class="px-3 py-1 rounded-full text-xs font-medium border border-amber-500/20 bg-amber-500/10 text-amber-300">{tag}</span>
            {/each}
          </div>
        </div>
      </div>
    </div>
  </section>

  <!-- CTA BANNER -->
  <section class="max-w-6xl mx-auto px-4 py-16">
    <div class="rounded-2xl bg-gradient-to-br from-amber-500/20 via-amber-500/10 to-transparent border border-amber-500/20 p-10 text-center">
      <h2 class="text-3xl sm:text-4xl font-bold mb-4">Ready to start tracking smarter?</h2>
      <p class="text-foreground mb-8 max-w-md mx-auto">Join Joules today. Set up takes under two minutes.</p>
      <a
        href="/signup"
        class="inline-block px-10 py-4 text-base font-semibold rounded-2xl bg-amber-500 hover:bg-amber-400 text-foreground transition-all shadow-lg shadow-amber-500/25 hover:shadow-amber-500/40 hover:-translate-y-0.5"
      >
        Create your free account
      </a>
    </div>
  </section>

  <!-- FOOTER -->
  <footer class="border-t border-border mt-8">
    <div class="max-w-6xl mx-auto px-4 py-8 flex flex-col sm:flex-row items-center justify-between gap-4">
      <div class="flex items-center gap-2.5">
        <Logo size={24} />
        <span class="text-sm text-muted-foreground">Joules — self-hosted nutrition tracking</span>
      </div>
      <div class="flex items-center gap-6 text-sm text-muted-foreground">
        <a href="/signup" class="hover:text-foreground transition-colors">Sign Up</a>
        <a href="/login" class="hover:text-foreground transition-colors">Sign In</a>
      </div>
    </div>
  </footer>

</div>
