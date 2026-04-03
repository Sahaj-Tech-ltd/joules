<script lang="ts">
  let { mood = 'okay', streak_days = 0, level = 1, level_name = '' }: {
    mood: string;
    streak_days: number;
    level: number;
    level_name: string;
  } = $props();

  const moodColors: Record<string, string> = {
    thriving: '#fbbf24',
    happy:    '#86efac',
    okay:     '#93c5fd',
    sad:      '#c4b5fd',
    sleeping: '#94a3b8',
  };

  let bodyColor = $derived(moodColors[mood] ?? moodColors['okay']);

  // Slightly darker shade for body shading
  const moodShadows: Record<string, string> = {
    thriving: '#d97706',
    happy:    '#4ade80',
    okay:     '#60a5fa',
    sad:      '#a78bfa',
    sleeping: '#64748b',
  };
  let shadowColor = $derived(moodShadows[mood] ?? moodShadows['okay']);

  let animClass = $derived(
    mood === 'thriving' ? 'pet-bounce'   :
    mood === 'happy'    ? 'pet-float'    :
    mood === 'sad'      ? 'pet-droop'    :
    mood === 'sleeping' ? 'pet-sleep'    :
    ''
  );
</script>

<div class="flex flex-col items-center select-none">
  <div class="pet-wrapper {animClass}">
    <svg
      width="120"
      height="120"
      viewBox="0 0 120 120"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      <!-- Blob body: irregular rounded shape -->
      <path
        d="M60 10
           C78 8, 96 18, 104 34
           C112 50, 112 68, 102 82
           C92 96, 76 108, 60 108
           C44 108, 28 96, 18 82
           C8 68, 8 50, 16 34
           C24 18, 42 12, 60 10Z"
        fill="{bodyColor}"
      />
      <!-- Body shading for depth -->
      <path
        d="M60 10
           C78 8, 96 18, 104 34
           C112 50, 112 68, 102 82
           C92 96, 76 108, 60 108
           C44 108, 28 96, 18 82
           C8 68, 8 50, 16 34
           C24 18, 42 12, 60 10Z"
        fill="url(#bodyGrad)"
      />

      <!-- Gradient definitions -->
      <defs>
        <radialGradient id="bodyGrad" cx="40%" cy="30%" r="60%">
          <stop offset="0%" stop-color="white" stop-opacity="0.25" />
          <stop offset="100%" stop-color="{shadowColor}" stop-opacity="0.3" />
        </radialGradient>
      </defs>

      <!-- THRIVING: sparkle star eyes + big smile + rosy cheeks -->
      {#if mood === 'thriving'}
        <!-- Left star eye -->
        <g transform="translate(40,46)">
          <path d="M0,-7 L1.5,-1.5 L7,0 L1.5,1.5 L0,7 L-1.5,1.5 L-7,0 L-1.5,-1.5Z" fill="#1e293b" />
          <path d="M0,-4.5 L0.8,-0.8 L4.5,0 L0.8,0.8 L0,4.5 L-0.8,0.8 L-4.5,0 L-0.8,-0.8Z" fill="white" opacity="0.6" />
        </g>
        <!-- Right star eye -->
        <g transform="translate(80,46)">
          <path d="M0,-7 L1.5,-1.5 L7,0 L1.5,1.5 L0,7 L-1.5,1.5 L-7,0 L-1.5,-1.5Z" fill="#1e293b" />
          <path d="M0,-4.5 L0.8,-0.8 L4.5,0 L0.8,0.8 L0,4.5 L-0.8,0.8 L-4.5,0 L-0.8,-0.8Z" fill="white" opacity="0.6" />
        </g>
        <!-- Rosy cheeks -->
        <ellipse cx="32" cy="64" rx="8" ry="5" fill="#f97316" opacity="0.35" />
        <ellipse cx="88" cy="64" rx="8" ry="5" fill="#f97316" opacity="0.35" />
        <!-- Big smile -->
        <path d="M38 74 Q60 92 82 74" stroke="#1e293b" stroke-width="3.5" stroke-linecap="round" fill="none" />
        <!-- Tongue -->
        <ellipse cx="60" cy="83" rx="8" ry="5" fill="#f87171" />

      <!-- HAPPY: oval eyes + smile -->
      {:else if mood === 'happy'}
        <!-- Left eye -->
        <ellipse cx="42" cy="48" rx="7" ry="8.5" fill="#1e293b" />
        <ellipse cx="44" cy="45" rx="2.5" ry="2.5" fill="white" opacity="0.7" />
        <!-- Right eye -->
        <ellipse cx="78" cy="48" rx="7" ry="8.5" fill="#1e293b" />
        <ellipse cx="80" cy="45" rx="2.5" ry="2.5" fill="white" opacity="0.7" />
        <!-- Smile -->
        <path d="M44 72 Q60 84 76 72" stroke="#1e293b" stroke-width="3" stroke-linecap="round" fill="none" />

      <!-- OKAY: round eyes, neutral mouth -->
      {:else if mood === 'okay'}
        <!-- Left eye -->
        <circle cx="42" cy="48" r="7" fill="#1e293b" />
        <circle cx="44" cy="46" r="2.5" fill="white" opacity="0.7" />
        <!-- Right eye -->
        <circle cx="78" cy="48" r="7" fill="#1e293b" />
        <circle cx="80" cy="46" r="2.5" fill="white" opacity="0.7" />
        <!-- Straight mouth -->
        <line x1="50" y1="74" x2="70" y2="74" stroke="#1e293b" stroke-width="3" stroke-linecap="round" />

      <!-- SAD: downcast oval eyes + frown + tear -->
      {:else if mood === 'sad'}
        <!-- Left eye (angled down) -->
        <ellipse cx="42" cy="50" rx="6.5" ry="7.5" fill="#1e293b" />
        <ellipse cx="43.5" cy="47.5" rx="2" ry="2" fill="white" opacity="0.7" />
        <!-- Right eye (angled down) -->
        <ellipse cx="78" cy="50" rx="6.5" ry="7.5" fill="#1e293b" />
        <ellipse cx="79.5" cy="47.5" rx="2" ry="2" fill="white" opacity="0.7" />
        <!-- Frown -->
        <path d="M44 78 Q60 68 76 78" stroke="#1e293b" stroke-width="3" stroke-linecap="round" fill="none" />
        <!-- Teardrop left eye -->
        <path d="M40 60 Q38 65 40 68 Q42 65 40 60Z" fill="#93c5fd" opacity="0.85" />

      <!-- SLEEPING: closed eyes (∪ shape) + ZZZ -->
      {:else if mood === 'sleeping'}
        <!-- Left closed eye (arc) -->
        <path d="M35 48 Q42 56 49 48" stroke="#1e293b" stroke-width="3.5" stroke-linecap="round" fill="none" />
        <!-- Right closed eye (arc) -->
        <path d="M71 48 Q78 56 85 48" stroke="#1e293b" stroke-width="3.5" stroke-linecap="round" fill="none" />
        <!-- Slight smile while sleeping -->
        <path d="M48 72 Q60 78 72 72" stroke="#1e293b" stroke-width="2.5" stroke-linecap="round" fill="none" />
        <!-- ZZZ floating above (styled in the bubble) -->
        <text x="72" y="22" font-size="9" font-weight="bold" fill="#1e293b" opacity="0.55" font-family="sans-serif">z</text>
        <text x="80" y="14" font-size="11" font-weight="bold" fill="#1e293b" opacity="0.7" font-family="sans-serif">z</text>
        <text x="90" y="7" font-size="13" font-weight="bold" fill="#1e293b" opacity="0.85" font-family="sans-serif">Z</text>
      {/if}
    </svg>
  </div>

  <div class="mt-2 text-center">
    <p class="text-sm font-semibold text-foreground">
      {#if streak_days > 0}
        🔥 {streak_days} day streak
      {:else}
        No streak yet
      {/if}
    </p>
    <p class="text-xs text-foreground mt-0.5">Level {level} · {level_name}</p>
  </div>
</div>

<style>
  .pet-wrapper {
    display: inline-block;
  }

  @keyframes bounce {
    from { transform: translateY(0); }
    to   { transform: translateY(-8px); }
  }

  @keyframes float {
    0%, 100% { transform: translateY(0); }
    50%       { transform: translateY(-4px); }
  }

  @keyframes droop {
    0%, 100% { transform: translateY(0) rotate(0deg); }
    50%       { transform: translateY(2px) rotate(-1deg); }
  }

  @keyframes sleep-float {
    0%, 100% { transform: translateY(0); }
    50%       { transform: translateY(-3px); }
  }

  .pet-bounce {
    animation: bounce 0.8s ease-in-out infinite alternate;
  }

  .pet-float {
    animation: float 2s ease-in-out infinite;
  }

  .pet-droop {
    animation: droop 3s ease-in-out infinite;
  }

  .pet-sleep {
    animation: sleep-float 3.5s ease-in-out infinite;
  }
</style>
