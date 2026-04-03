<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';

  const STORAGE_KEY = 'joules_tour_v1';

  let visible = $state(false);
  let step = $state(0);

  const steps = [
    {
      icon: '👋',
      title: 'Welcome to Joules',
      body: 'Your AI-powered nutrition companion. This quick tour will show you the key features — takes less than a minute.',
      action: null,
    },
    {
      icon: '🍽️',
      title: 'Log your meals',
      body: 'Tap "Log a Meal" to snap a photo and let AI identify your food automatically — or type it manually. The AI reads menus, packaging, and even rough descriptions.',
      action: { label: 'Log a meal now', href: '/log' },
    },
    {
      icon: '🤖',
      title: 'Your AI health coach',
      body: 'Chat with your personal nutrition coach anytime. Ask about your macros, get meal ideas, or discuss your goals. It knows your full history.',
      action: { label: 'Open coach', href: '/coach' },
    },
    {
      icon: '📈',
      title: 'Track your progress',
      body: 'See your calorie trends, weight chart, and streaks on the Progress page. Log your weight daily for the best picture of your journey.',
      action: { label: 'View progress', href: '/progress' },
    },
    {
      icon: '🔔',
      title: 'Push notifications',
      body: 'Get reminders to log meals, drink water, and weigh in. Go to Settings → Connections to set up Google Fit, or Settings → Notifications to configure reminders via ntfy.',
      action: { label: 'Open settings', href: '/settings' },
    },
  ];

  onMount(() => {
    if (!localStorage.getItem(STORAGE_KEY)) {
      // Small delay so the dashboard finishes loading first
      setTimeout(() => { visible = true; }, 800);
    }
  });

  function next() {
    if (step < steps.length - 1) {
      step++;
    } else {
      finish();
    }
  }

  function skip() {
    finish();
  }

  function finish() {
    localStorage.setItem(STORAGE_KEY, '1');
    visible = false;
  }

  function handleAction(href: string) {
    finish();
    goto(href);
  }

  let current = $derived(steps[step]);
  let isLast = $derived(step === steps.length - 1);
</script>

{#if visible}
  <!-- Backdrop -->
  <div
    class="fixed inset-0 z-50 flex items-end justify-center bg-black/70 backdrop-blur-sm sm:items-center"
    role="presentation"
    onclick={skip}
  >
    <!-- Card -->
    <div
      class="relative w-full max-w-sm rounded-t-2xl sm:rounded-2xl border border-border/60 bg-secondary p-6 pb-8 shadow-2xl"
      onclick={(e) => e.stopPropagation()}
      role="dialog"
      aria-modal="true"
      aria-label="App walkthrough"
    >
      <!-- Drag handle (mobile) -->
      <div class="mx-auto mb-6 h-1 w-10 rounded-full bg-accent sm:hidden"></div>

      <!-- Skip -->
      <button
        class="absolute right-4 top-4 text-xs text-muted-foreground hover:text-foreground transition"
        onclick={skip}
      >
        Skip tour
      </button>

      <!-- Icon -->
      <div class="mb-4 text-4xl">{current.icon}</div>

      <!-- Content -->
      <h2 class="mb-2 text-lg font-bold text-foreground">{current.title}</h2>
      <p class="text-sm leading-relaxed text-foreground">{current.body}</p>

      <!-- Optional action link -->
      {#if current.action}
        <button
          class="mt-4 text-xs font-medium text-primary hover:text-primary/80 underline underline-offset-2 transition"
          onclick={() => current.action && handleAction(current.action.href)}
        >
          {current.action.label} →
        </button>
      {/if}

      <!-- Footer: dots + next -->
      <div class="mt-6 flex items-center justify-between">
        <!-- Dot pagination -->
        <div class="flex gap-1.5">
          {#each steps as _, i}
            <div
              class="rounded-full transition-all duration-300 {i === step ? 'w-4 h-1.5 bg-primary' : 'w-1.5 h-1.5 bg-accent'}"
            ></div>
          {/each}
        </div>

        <button
          class="rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition active:scale-[0.97]"
          onclick={next}
        >
          {isLast ? 'Get started' : 'Next'}
        </button>
      </div>
    </div>
  </div>
{/if}
