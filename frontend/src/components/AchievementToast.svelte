<script lang="ts">
  import { browser } from '$app/environment';
  import { newAchievements } from '$lib/achievements';
  import Confetti from './Confetti.svelte';

  interface ToastItem {
    id: string;
    title: string;
    description: string;
    visible: boolean;
    fadingOut: boolean;
  }

  let toasts = $state<ToastItem[]>([]);
  let showConfetti = $state(false);
  let confettiKey = $state(0);

  if (browser) {
    newAchievements.subscribe((achievements) => {
      // Find newly added achievements (ones not already in toasts)
      const existingIds = new Set(toasts.map(t => t.id));
      for (const a of achievements) {
        if (!existingIds.has(a.id)) {
          // Add the toast
          toasts = [...toasts, { id: a.id, title: a.title, description: a.description, visible: false, fadingOut: false }];

          // Trigger confetti for first new achievement in a batch
          if (!showConfetti) {
            confettiKey += 1;
            showConfetti = true;
            setTimeout(() => { showConfetti = false; }, 3500);
          }

          // Animate in
          setTimeout(() => {
            toasts = toasts.map(t => t.id === a.id ? { ...t, visible: true } : t);
          }, 50);

          // Start fade-out after 4s
          setTimeout(() => {
            toasts = toasts.map(t => t.id === a.id ? { ...t, fadingOut: true } : t);
          }, 4000);

          // Remove after fade completes
          setTimeout(() => {
            toasts = toasts.filter(t => t.id !== a.id);
          }, 4600);
        }
      }
    });
  }
</script>

{#if browser}
  {#key confettiKey}
    <Confetti show={showConfetti} />
  {/key}

  <div
    class="fixed z-[9998] flex flex-col gap-3 pointer-events-none"
    style="bottom: 5.5rem; right: 1rem; max-width: 22rem; width: calc(100vw - 2rem);"
    aria-live="polite"
    aria-label="Achievement notifications"
  >
    {#each toasts as toast (toast.id)}
      <div
        class="pointer-events-auto flex items-start gap-3 rounded-xl border border-amber-500/40 dark:border-amber-500/30 bg-secondary px-4 py-3.5 shadow-2xl shadow-black/40 transition-all duration-500"
        style="
          transform: {toast.visible && !toast.fadingOut ? 'translateY(0)' : 'translateY(1.5rem)'};
          opacity: {toast.visible && !toast.fadingOut ? '1' : '0'};
        "
        role="status"
      >
        <!-- Trophy icon -->
        <div class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-amber-400 to-amber-600 shadow-lg shadow-amber-500/30">
          <svg class="h-5 w-5 text-primary-foreground" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
          </svg>
        </div>

        <!-- Content -->
        <div class="min-w-0 flex-1">
          <p class="text-xs font-semibold uppercase tracking-wider text-amber-600 dark:text-amber-400">Achievement Unlocked</p>
          <p class="mt-0.5 text-sm font-bold text-foreground leading-tight">{toast.title}</p>
          {#if toast.description}
            <p class="mt-1 text-xs text-foreground leading-snug">{toast.description}</p>
          {/if}
        </div>

        <!-- Dismiss button -->
        <button
          onclick={() => { toasts = toasts.filter(t => t.id !== toast.id); }}
          class="mt-0.5 shrink-0 rounded p-0.5 text-muted-foreground hover:text-foreground transition"
          aria-label="Dismiss"
        >
          <svg class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
          </svg>
        </button>
      </div>
    {/each}
  </div>
{/if}
