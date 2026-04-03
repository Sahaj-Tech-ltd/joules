<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';

  interface Banner {
    id: string;
    title: string;
    message: string;
    type: string;
    expires_at?: string;
  }

  let banners = $state<Banner[]>([]);
  let dismissed = $state<Set<string>>(new Set());

  onMount(async () => {
    try {
      const stored = JSON.parse(localStorage.getItem('dismissed_banners') || '[]');
      dismissed = new Set(stored);
      const res = await api.get<Banner[]>('/banners');
      banners = (res ?? []).filter((b: Banner) => !dismissed.has(b.id));
    } catch {}
  });

  function dismiss(id: string) {
    dismissed = new Set([...dismissed, id]);
    banners = banners.filter(b => b.id !== id);
    localStorage.setItem('dismissed_banners', JSON.stringify([...dismissed]));
  }

  const typeStyles: Record<string, { bg: string; border: string; text: string; icon: string }> = {
    info:    { bg: 'bg-blue-500/10', border: 'border-blue-500/20', text: 'text-blue-300', icon: 'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z' },
    warning: { bg: 'bg-amber-500/10', border: 'border-amber-500/20', text: 'text-amber-300', icon: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z' },
    tip:     { bg: 'bg-primary/10', border: 'border-primary/20', text: 'text-primary', icon: 'M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z' },
    error:   { bg: 'bg-red-500/10', border: 'border-red-500/20', text: 'text-red-400', icon: 'M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z' },
  };
</script>

{#if banners.length > 0}
  <div class="space-y-2">
    {#each banners as banner}
      {@const style = typeStyles[banner.type] ?? typeStyles.info}
      <div class="flex items-start gap-3 rounded-xl border px-4 py-3 {style.bg} {style.border}">
        <svg class="h-4 w-4 shrink-0 mt-0.5 {style.text}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={style.icon} />
        </svg>
        <div class="flex-1 min-w-0">
          {#if banner.title}
            <p class="text-sm font-semibold {style.text}">{banner.title}</p>
          {/if}
          <p class="text-sm text-foreground {banner.title ? 'mt-0.5' : ''}">{banner.message}</p>
        </div>
        <button
          onclick={() => dismiss(banner.id)}
          class="shrink-0 text-muted-foreground hover:text-foreground transition"
          aria-label="Dismiss"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    {/each}
  </div>
{/if}
