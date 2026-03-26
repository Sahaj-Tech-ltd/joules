<script lang="ts" module>
  import { Chart, ArcElement, Tooltip, DoughnutController } from 'chart.js';
  Chart.register(ArcElement, Tooltip, DoughnutController);
</script>

<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  let { consumed = 0, target = 2000, label = 'Calories', unit = '', color = '#f59e0b' }: { consumed: number; target: number; label?: string; unit?: string; color?: string } = $props();

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
          backgroundColor: [color, 'rgba(51, 65, 85, 0.5)'],
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

<div class="rounded-xl border border-slate-800 bg-surface-light p-4 flex flex-col items-center gap-2">
  <div class="relative" style="width: 120px; height: 120px;">
    <canvas bind:this={canvas} width="120" height="120"></canvas>
    <div class="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
      <span class="text-2xl font-bold text-slate-100">{pct}%</span>
    </div>
  </div>
  <div class="text-center">
    <p class="text-sm text-slate-100 font-medium">{consumed.toLocaleString()} / {target.toLocaleString()} {unit}</p>
    <p class="text-xs text-slate-400 mt-0.5">{label}</p>
  </div>
</div>
