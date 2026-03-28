<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { browser } from '$app/environment';

  let visible = $state(false);
  let platform = $state<'ios' | 'android-chrome' | null>(null);
  let deferredPrompt = $state<any>(null);

  function dismiss() {
    visible = false;
    if (browser) localStorage.setItem('pwa_nudge_dismissed', '1');
  }

  async function installApp() {
    if (!deferredPrompt) return;
    deferredPrompt.prompt();
    const { outcome } = await deferredPrompt.userChoice;
    if (outcome === 'accepted') {
      visible = false;
      if (browser) localStorage.setItem('pwa_nudge_dismissed', '1');
    }
    deferredPrompt = null;
  }

  function handleBeforeInstall(e: Event) {
    e.preventDefault();
    deferredPrompt = e;
    if (localStorage.getItem('pwa_nudge_dismissed')) return;
    platform = 'android-chrome';
    visible = true;
  }

  onMount(() => {
    if (!browser) return;
    if (localStorage.getItem('pwa_nudge_dismissed')) return;

    // Only show on mobile
    const isMobile = window.matchMedia('(hover: none) and (pointer: coarse)').matches;
    if (!isMobile) return;

    // Already installed as PWA
    if (window.matchMedia('(display-mode: standalone)').matches) return;

    const ua = navigator.userAgent;
    const isIOS = /iPad|iPhone|iPod/.test(ua) && !(ua.includes('CriOS') || ua.includes('FxiOS'));
    const isAndroidChrome = /Android/.test(ua) && /Chrome\//.test(ua) && !/Edge\//.test(ua);

    if (isIOS) {
      platform = 'ios';
      // Delay the nudge by 3 seconds so it doesn't pop up immediately
      setTimeout(() => { visible = true; }, 3000);
    } else if (isAndroidChrome) {
      window.addEventListener('beforeinstallprompt', handleBeforeInstall);
    }
  });

  onDestroy(() => {
    if (browser) window.removeEventListener('beforeinstallprompt', handleBeforeInstall);
  });
</script>

{#if visible}
  <!-- Slide-up banner from bottom, above mobile nav -->
  <div
    class="fixed bottom-16 left-3 right-3 z-40 rounded-2xl border border-joule-500/30 bg-slate-900 shadow-2xl shadow-black/40 p-4"
    style="animation: slideUpBanner 0.35s cubic-bezier(0.34, 1.56, 0.64, 1) both;"
  >
    <div class="flex items-start gap-3">
      <!-- Icon -->
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-joule-500/10">
        <svg class="h-5 w-5 text-joule-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z" />
        </svg>
      </div>

      <div class="flex-1 min-w-0">
        <p class="text-sm font-semibold text-white">Add Joules to your home screen</p>
        {#if platform === 'ios'}
          <p class="mt-0.5 text-xs text-slate-400 leading-relaxed">
            Tap <span class="inline-flex items-center gap-0.5 text-joule-400 font-medium">
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
              </svg>
              Share
            </span> then <strong class="text-white">"Add to Home Screen"</strong>
          </p>
        {:else if platform === 'android-chrome'}
          <p class="mt-0.5 text-xs text-slate-400">Get the full app experience with one tap.</p>
        {/if}
      </div>

      <!-- Actions -->
      <div class="flex items-center gap-2 shrink-0">
        {#if platform === 'android-chrome' && deferredPrompt}
          <button
            onclick={installApp}
            class="rounded-lg bg-joule-500 px-3 py-1.5 text-xs font-semibold text-slate-900 hover:bg-joule-400 transition"
          >
            Install
          </button>
        {/if}
        <button
          onclick={dismiss}
          class="flex h-7 w-7 items-center justify-center rounded-lg text-slate-500 hover:text-white hover:bg-white/10 transition"
          aria-label="Dismiss"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  @keyframes slideUpBanner {
    from { transform: translateY(120%); opacity: 0; }
    to { transform: translateY(0); opacity: 1; }
  }
</style>
