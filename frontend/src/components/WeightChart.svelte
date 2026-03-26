<script lang="ts" module>
  import {
    Chart, CategoryScale, LinearScale, PointElement, LineElement,
    Tooltip, Filler, LineController
  } from 'chart.js';
  Chart.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Filler, LineController);
</script>

<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  let { data = [] as { date: string; weight_kg: number }[] }: { data: { date: string; weight_kg: number }[] } = $props();

  let canvas: HTMLCanvasElement;
  let chart: Chart;

  onMount(() => {
    if (!canvas || data.length === 0) return;

    chart = new Chart(canvas, {
      type: 'line',
      data: {
        labels: data.map(d => {
          const date = new Date(d.date);
          return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
        }),
        datasets: [{
          label: 'Weight (kg)',
          data: data.map(d => d.weight_kg),
          borderColor: '#f59e0b',
          backgroundColor: 'rgba(245, 158, 11, 0.1)',
          fill: true,
          tension: 0.3,
          pointBackgroundColor: '#f59e0b',
          pointBorderColor: '#f59e0b',
          pointRadius: 4,
          pointHoverRadius: 6,
          borderWidth: 2
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          x: {
            grid: { color: 'rgba(51, 65, 85, 0.5)' },
            ticks: { color: '#64748b', font: { size: 11 } }
          },
          y: {
            grid: { color: 'rgba(51, 65, 85, 0.5)' },
            ticks: { color: '#64748b', font: { size: 11 } },
            beginAtZero: false
          }
        },
        plugins: {
          tooltip: {
            backgroundColor: '#1e293b',
            titleColor: '#e2e8f0',
            bodyColor: '#e2e8f0',
            borderColor: '#334155',
            borderWidth: 1
          }
        }
      }
    });
  });

  onDestroy(() => {
    chart?.destroy();
  });
</script>

<div class="rounded-xl border border-slate-800 bg-surface-light p-6">
  <h3 class="text-sm font-semibold text-slate-100 mb-4">Weight Progress</h3>
  {#if data.length === 0}
    <div class="h-48 flex items-center justify-center">
      <p class="text-sm text-slate-400">No weight data yet</p>
    </div>
  {:else}
    <div class="h-48">
      <canvas bind:this={canvas}></canvas>
    </div>
  {/if}
</div>
