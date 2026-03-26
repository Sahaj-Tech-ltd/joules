<script lang="ts">
  import { goto } from '$app/navigation';
  import { authToken } from '$lib/stores';
  import { onMount } from 'svelte';

  onMount(() => {
    const unsub = authToken.subscribe((token) => {
      if (token) {
        const profile = JSON.parse(localStorage.getItem('user_profile') || '{}');
        if (!profile.onboarding_complete) {
          goto('/onboarding');
        } else {
          goto('/dashboard');
        }
      } else {
        goto('/login');
      }
    });
    return unsub;
  });
</script>

<div class="flex h-screen items-center justify-center">
  <div class="h-8 w-8 animate-spin rounded-full border-2 border-joule-500 border-t-transparent"></div>
</div>
