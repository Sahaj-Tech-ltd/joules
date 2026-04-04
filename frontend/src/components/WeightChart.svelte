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

  function createChart() {
    if (!canvas || data.length === 0) return;
    chart?.destroy();

    const style = getComputedStyle(document.documentElement);
    const chart1 = style.getPropertyValue('--chart-1').trim();
    const borderColor = style.getPropertyValue('--border').trim();
    const mutedFg = style.getPropertyValue('--muted-foreground').trim();
    const lineColor = `oklch(${chart1})`;
    const gridColor = `oklch(${borderColor} / 0.5)`;
    const tickColor = `oklch(${mutedFg})`;

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
          borderColor: lineColor,
          backgroundColor: `oklch(${chart1} / 0.1)`,
          fill: true,
          tension: 0.3,
          pointBackgroundColor: lineColor,
          pointBorderColor: lineColor,
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
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { size: 11 } }
          },
          y: {
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { size: 11 } },
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
  }

  $effect(() => {
    void data;
    createChart();
  });

  onDestroy(() => {
    chart?.destroy();
  });
</script>

<div class="rounded-2xl border border-border bg-card p-5">
  <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-4">Weight Progress</p>
  {#if data.length === 0}
    <div class="py-6 flex flex-col items-center justify-center gap-2">
      <p class="text-sm text-muted-foreground">No weight data yet</p>
      <p class="text-xs text-muted-foreground/60">Log your weight below to start tracking progress</p>
    </div>
  {:else}
    <div class="h-44">
      <canvas bind:this={canvas}></canvas>
    </div>
  {/if}
</div>
