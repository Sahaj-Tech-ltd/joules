<script lang="ts">
  import '../app.css';
  import AchievementToast from '$components/AchievementToast.svelte';
  import InstallBanner from '$components/InstallBanner.svelte';

  let { children } = $props();
</script>

{@render children()}
<AchievementToast />
<InstallBanner />
