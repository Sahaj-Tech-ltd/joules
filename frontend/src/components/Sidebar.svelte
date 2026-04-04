<script lang="ts">
  import Logo from '$components/Logo.svelte';
  import { api } from '$lib/api';
  import { features } from '$lib/features';
  import { onMount } from 'svelte';

  let { activePage = '', isAdmin = false }: { activePage?: string; isAdmin?: boolean } = $props();

  let habitLevel = $state(0);
  let habitPoints = $state(0);
  let habitLevelPct = $state(0);

  onMount(() => {
    api.get<{level: number; total_points: number; level_progress_pct: number}>('/habits/summary')
      .then(s => {
        habitLevel = s.level;
        habitPoints = s.total_points;
        habitLevelPct = s.level_progress_pct;
      }).catch(() => {});
  });

  const nav = [
    {
      id: 'dashboard',
      href: '/dashboard',
      label: 'Dashboard',
      icon: '<path stroke-linecap="round" stroke-linejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />'
    },
    {
      id: 'log',
      href: '/log',
      label: 'Log Meal',
      icon: '<path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />'
    },
    {
      id: 'coach',
      href: '/coach',
      label: 'Health Coach',
      icon: '<path stroke-linecap="round" stroke-linejoin="round" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />'
    },
    {
      id: 'progress',
      href: '/progress',
      label: 'Progress',
      icon: '<path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />'
    },
    {
      id: 'groups',
      href: '/groups',
      label: 'Groups',
      icon: '<path stroke-linecap="round" stroke-linejoin="round" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z" />'
    },
    {
      id: 'achievements',
      href: '/achievements',
      label: 'Achievements',
      icon: '<path stroke-linecap="round" stroke-linejoin="round" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />'
    },
    {
      id: 'settings',
      href: '/settings',
      label: 'Settings',
      icon: '<path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />'
    },
  ];

  // Mobile bottom nav: 5 core pages
  const mobileNav = ['dashboard', 'log', 'coach', 'progress'].map(id => nav.find(n => n.id === id)!);
</script>

<!-- Desktop sidebar -->
<aside class="hidden w-64 lg:flex flex-col border-r border-border bg-secondary shrink-0">
  <!-- Logo -->
  <div class="px-5 py-5">
    <a href="/dashboard" class="flex items-center gap-2.5 group">
      <Logo size={28} />
      <span class="font-display text-[16px] font-bold tracking-tight text-foreground">Joules</span>
    </a>
  </div>

  <div class="mx-4 h-px bg-accent/50"></div>

  <!-- Nav -->
  <nav class="flex-1 px-3 py-4 space-y-0.5">
    {#each nav as item}
      {@const active = activePage === item.id}
      {@const gated = item.id === 'coach' || item.id === 'groups' || item.id === 'achievements'}
      {#if !gated || $features[item.id]}
        <a
          href={item.href}
          class="group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-150
            {active
              ? 'bg-primary/10 text-primary ring-1 ring-inset ring-ring/20'
              : 'text-muted-foreground hover:text-foreground/80 hover:bg-accent/50'}"
        >
          <svg
            class="h-[17px] w-[17px] shrink-0 transition-colors {active ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground'}"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="1.75"
          >
            {@html item.icon}
          </svg>
          {item.label}
        </a>
      {/if}
    {/each}

    {#if isAdmin}
      <div class="mx-3 my-3 h-px bg-accent/50"></div>
      <a
        href="/admin"
        class="group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-150
          {activePage === 'admin'
            ? 'bg-primary/10 text-primary ring-1 ring-inset ring-ring/20'
            : 'text-muted-foreground hover:text-foreground/80 hover:bg-accent/50'}"
      >
        <svg
          class="h-[17px] w-[17px] shrink-0 transition-colors {activePage === 'admin' ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground'}"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="1.75"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
        </svg>
        Admin Panel
      </a>
    {/if}

    {#if habitLevel > 0}
      <div class="mt-auto pt-4 border-t border-border">
        <div class="px-3 py-2.5 rounded-xl bg-accent/50">
          <div class="flex items-center justify-between mb-1.5">
            <span class="text-xs text-foreground font-medium font-display">Lv.{habitLevel}</span>
            <span class="text-xs text-muted-foreground">{habitPoints} XP</span>
          </div>
          <div class="h-1.5 w-full rounded-full bg-accent/50 overflow-hidden">
            <div class="h-full rounded-full bg-gradient-to-r from-primary to-primary/80" style="width:{habitLevelPct}%"></div>
          </div>
        </div>
      </div>
    {/if}
  </nav>
</aside>

<!-- Mobile bottom nav -->
<nav class="lg:hidden fixed bottom-0 left-0 right-0 z-50 border-t border-border bg-card/85 backdrop-blur-xl flex items-stretch"
  style="height: calc(4rem + env(safe-area-inset-bottom, 0px)); padding-bottom: env(safe-area-inset-bottom, 0px);">
  {#each mobileNav.filter(n => n.id !== 'log' || activePage !== 'log') as item}
    {@const active = activePage === item.id}
    {@const gated = item.id === 'coach'}
    {#if !gated || $features[item.id]}
    {@const isLog = item.id === 'log'}
    <a
      href={item.href}
      class="flex flex-1 flex-col items-center justify-center gap-1 pt-2 pb-1 transition-colors relative
        {active ? 'text-primary' : 'text-muted-foreground hover:text-foreground'}"
    >
      {#if active}
        <span class="absolute top-1.5 left-1/2 -translate-x-1/2 w-4 h-0.5 rounded-full bg-primary/80"></span>
      {/if}
      {#if isLog && !active}
        <span class="flex items-center justify-center w-9 h-9 rounded-2xl bg-primary shadow-lg shadow-primary/30 -mt-1">
          <svg class="h-5 w-5 text-primary-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.25">
            {@html item.icon}
          </svg>
        </span>
        <span class="text-[9px] font-medium leading-none text-primary/80">Log</span>
      {:else if isLog && active}
        <span class="flex items-center justify-center w-9 h-9 rounded-2xl bg-primary/80 shadow-lg shadow-primary/40 -mt-1">
          <svg class="h-5 w-5 text-primary-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.25">
            {@html item.icon}
          </svg>
        </span>
        <span class="text-[9px] font-medium leading-none text-primary">Log</span>
      {:else}
        <svg
          class="h-5 w-5 shrink-0"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width={active ? '2' : '1.75'}
        >
          {@html item.icon}
        </svg>
        <span class="text-[9px] font-medium leading-none">{item.label === 'Health Coach' ? 'Coach' : item.label}</span>
      {/if}
    </a>
    {/if}
  {/each}
</nav>
