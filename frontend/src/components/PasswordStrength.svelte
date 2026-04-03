<script lang="ts">
  let { password }: { password: string } = $props();

  const symbols = `!@#$%^&*()_+-=[]{}|;':",./<>?`;

  const score = $derived.by(() => {
    if (!password) return 0;
    let s = Math.min(password.length * 4, 40);
    if (/[A-Z]/.test(password)) s += 15;
    if (/[0-9]/.test(password)) s += 15;
    if ([...password].some(c => symbols.includes(c))) s += 20;
    if (password.length >= 12) s += 10;
    return Math.min(s, 100);
  });

  const label = $derived(
    score === 0 ? '' :
    score <= 25 ? 'Weak' :
    score <= 50 ? 'Fair' :
    score <= 75 ? 'Good' : 'Excellent'
  );

  const barColor = $derived(
    score <= 25 ? 'bg-red-600 dark:bg-red-500' :
    score <= 50 ? 'bg-orange-600 dark:bg-orange-500' :
    score <= 75 ? 'bg-yellow-600 dark:bg-yellow-500' : 'bg-green-600 dark:bg-green-500'
  );
</script>

{#if password.length > 0}
  <div class="mt-2 space-y-1">
    <div class="h-1.5 w-full overflow-hidden rounded-full bg-muted">
      <div
        class="h-full rounded-full transition-all duration-300 {barColor}"
        style="width: {score}%"
      ></div>
    </div>
    <div class="flex items-center justify-between">
      <span class="text-xs text-muted-foreground">Strength</span>
      <span class="text-xs font-medium
        {score <= 25 ? 'text-red-600 dark:text-red-500' :
         score <= 50 ? 'text-orange-600 dark:text-orange-500' :
         score <= 75 ? 'text-yellow-600 dark:text-yellow-500' : 'text-green-600 dark:text-green-500'}">
        {label} · {score}/100
      </span>
    </div>
  </div>
{/if}
