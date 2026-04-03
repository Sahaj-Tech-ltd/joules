<script lang="ts">
  import { theme } from '$lib/stores';
  import { get } from 'svelte/store';

  let current = $state(get(theme));

  function toggle() {
    const next = current === 'dark' ? 'light' : 'dark';
    theme.set(next);
    current = next;
  }
</script>

<button
  onclick={toggle}
  class="flex h-8 w-8 items-center justify-center rounded-lg border transition {current === 'dark' ? 'border-border text-foreground hover:bg-accent' : 'bg-primary border-primary text-primary-foreground hover:bg-primary/80'}"
  aria-label="Toggle theme"
>
  {#if current === 'dark'}
    <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" /></svg>
  {:else}
    <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" /></svg>
  {/if}
</button>
