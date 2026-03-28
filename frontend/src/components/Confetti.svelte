<script lang="ts">
  import { browser } from '$app/environment';
  import { onMount } from 'svelte';

  let { show = false }: { show: boolean } = $props();

  let canvas: HTMLCanvasElement | undefined = $state();
  let animationId: number | null = null;

  interface Particle {
    x: number;
    y: number;
    vx: number;
    vy: number;
    rotation: number;
    rotationSpeed: number;
    color: string;
    size: number;
    opacity: number;
    shape: 'rect' | 'circle';
  }

  const COLORS = [
    '#f59e0b', // joule amber-500
    '#fbbf24', // joule amber-400
    '#fcd34d', // joule amber-300
    '#fde68a', // joule amber-200
    '#3b82f6', // blue-500
    '#60a5fa', // blue-400
    '#8b5cf6', // violet-500
    '#a78bfa', // violet-400
    '#10b981', // emerald-500
    '#34d399', // emerald-400
    '#f43f5e', // rose-500
    '#fb7185', // rose-400
    '#06b6d4', // cyan-500
    '#22d3ee', // cyan-400
    '#eab308', // yellow-500
    '#facc15', // yellow-400
  ];

  function spawnParticles(w: number): Particle[] {
    const count = Math.floor(Math.random() * 41) + 80; // 80-120
    return Array.from({ length: count }, () => ({
      x: Math.random() * w,
      y: -10 - Math.random() * 40,
      vx: (Math.random() - 0.5) * 6,
      vy: Math.random() * 4 + 2,
      rotation: Math.random() * Math.PI * 2,
      rotationSpeed: (Math.random() - 0.5) * 0.2,
      color: COLORS[Math.floor(Math.random() * COLORS.length)],
      size: Math.random() * 8 + 5,
      opacity: 1,
      shape: Math.random() > 0.5 ? 'rect' : 'circle',
    }));
  }

  function runAnimation() {
    if (!canvas || !browser) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;

    let particles = spawnParticles(canvas.width);
    const startTime = performance.now();
    const duration = 3200;

    function draw(now: number) {
      if (!canvas || !ctx) return;
      const elapsed = now - startTime;
      const progress = elapsed / duration;

      ctx.clearRect(0, 0, canvas.width, canvas.height);

      let alive = false;
      for (const p of particles) {
        p.x += p.vx;
        p.y += p.vy;
        p.vy += 0.12; // gravity
        p.vx *= 0.99; // air resistance
        p.rotation += p.rotationSpeed;

        // fade out in last 30% of duration
        if (progress > 0.7) {
          p.opacity = Math.max(0, 1 - (progress - 0.7) / 0.3);
        }

        if (p.y < canvas.height + 20 && p.opacity > 0) {
          alive = true;
          ctx.save();
          ctx.globalAlpha = p.opacity;
          ctx.translate(p.x, p.y);
          ctx.rotate(p.rotation);
          ctx.fillStyle = p.color;

          if (p.shape === 'circle') {
            ctx.beginPath();
            ctx.arc(0, 0, p.size / 2, 0, Math.PI * 2);
            ctx.fill();
          } else {
            ctx.fillRect(-p.size / 2, -p.size / 4, p.size, p.size / 2);
          }

          ctx.restore();
        }
      }

      if (alive && elapsed < duration + 500) {
        animationId = requestAnimationFrame(draw);
      } else {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        animationId = null;
      }
    }

    animationId = requestAnimationFrame(draw);
  }

  $effect(() => {
    if (!browser) return;
    if (show) {
      if (animationId !== null) {
        cancelAnimationFrame(animationId);
        animationId = null;
      }
      runAnimation();
    }
  });

  onMount(() => {
    return () => {
      if (animationId !== null) {
        cancelAnimationFrame(animationId);
      }
    };
  });
</script>

{#if browser}
  <canvas
    bind:this={canvas}
    style="position: fixed; inset: 0; pointer-events: none; z-index: 9999;"
  ></canvas>
{/if}
