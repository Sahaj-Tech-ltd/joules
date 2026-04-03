<script lang="ts" module>
  import { Chart, ArcElement, Tooltip, DoughnutController } from 'chart.js';
  Chart.register(ArcElement, Tooltip, DoughnutController);
</script>

<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  let { consumed = 0, target = 2000, label = 'Calories', unit = '', color = '#f59e0b', displayConsumed = consumed, displayTarget = target }: { consumed: number; target: number; label?: string; unit?: string; color?: string; displayConsumed?: number; displayTarget?: number } = $props();

  let canvas: HTMLCanvasElement;
  let chart: Chart;
  let remaining = $derived(Math.max(0, target - consumed));
  let pct = $derived(target > 0 ? Math.min(100, Math.round((consumed / target) * 100)) : 0);

  function createChart() {
    if (!canvas) return;
    chart?.destroy();
    chart = new Chart(canvas, {
      type: 'doughnut',
      data: {
        datasets: [{
          data: [consumed, remaining],
          backgroundColor: [color, 'rgba(148, 163, 184, 0.2)'],
          borderWidth: 0
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        cutout: '80%',
        plugins: {
          tooltip: {
            enabled: true,
            callbacks: {
              label: (ctx) => `${ctx.raw} ${unit}`
            }
          }
        }
      }
    });
  }

  onMount(() => {
    createChart();
  });

  $effect(() => {
    void consumed;
    void target;
    void color;
    void unit;
    if (chart) {
      createChart();
    }
  });

  onDestroy(() => {
    chart?.destroy();
  });
</script>

<div class="relative rounded-2xl border border-border bg-card flex flex-col items-center gap-3 p-4 overflow-hidden">
  <div class="absolute top-0 left-0 right-0 h-[2px] rounded-t-2xl" style="background: linear-gradient(90deg, {color}cc, {color}44);"></div>
  <div class="relative w-full max-w-[110px] mx-auto" style="aspect-ratio: 1 / 1;">
    <canvas bind:this={canvas}></canvas>
    <div class="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
      <span class="font-display text-[22px] font-bold leading-none" style="color: {pct >= 100 ? 'var(--destructive)' : 'var(--foreground)'}">{pct}%</span>
    </div>
  </div>
  <div class="text-center">
    <p class="text-[10px] font-semibold uppercase tracking-widest mb-1" style="color: var(--foreground);">{label}</p>
    <p class="font-display text-sm font-semibold" style="color: var(--foreground)">{displayConsumed.toLocaleString()}<span class="text-xs font-normal" style="color: var(--muted-foreground)"> / {displayTarget.toLocaleString()} {unit}</span></p>
  </div>
</div>
