<script lang="ts">
  let { size = 40 }: { size?: number } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 40 40" fill="none" xmlns="http://www.w3.org/2000/svg">
  <path d="M23 2L10 20h9l-3 18L30 16h-9z" fill="url(#bolt-grad)" />
  <defs>
    <linearGradient id="bolt-grad" x1="10" y1="2" x2="30" y2="38" gradientUnits="userSpaceOnUse">
      <stop stop-color="#fbbf24" />
      <stop offset="1" stop-color="#f59e0b" />
    </linearGradient>
  </defs>
</svg>
