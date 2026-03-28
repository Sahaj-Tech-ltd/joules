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

  onMount(() => {
    if (!canvas) return;
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
        maintainAspectRatio: true,
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
  });

  onDestroy(() => {
    chart?.destroy();
  });
</script>

<div class="relative rounded-xl border border-slate-800 bg-surface-light flex flex-col items-center gap-2.5 p-4 overflow-hidden shadow-sm shadow-black/20">
  <div class="absolute top-0 left-0 right-0 h-[2px] rounded-t-xl" style="background: {color}; opacity: 0.65;"></div>
  <div class="relative" style="width: 120px; height: 120px;">
    <canvas bind:this={canvas} width="120" height="120"></canvas>
    <div class="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
      <span class="text-2xl font-bold" style="color: {pct >= 100 ? '#f87171' : 'var(--color-text-primary)'}">{pct}%</span>
    </div>
  </div>
  <div class="text-center">
    <p class="text-[11px] font-semibold uppercase tracking-wider mb-0.5" style="color: {color};">{label}</p>
    <p class="text-sm font-medium" style="color: var(--color-text-primary)">{displayConsumed.toLocaleString()}<span class="text-slate-500 text-xs"> / {displayTarget.toLocaleString()} {unit}</span></p>
  </div>
</div>
