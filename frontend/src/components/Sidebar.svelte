<script lang="ts">
  import Logo from '$components/Logo.svelte';

  let { activePage = '', isAdmin = false }: { activePage?: string; isAdmin?: boolean } = $props();

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

  // Mobile bottom nav: first 5 core pages (skip settings, show admin if admin)
  const mobileNav = nav.slice(0, 5);
</script>

<!-- Desktop sidebar -->
<aside class="hidden w-64 lg:flex flex-col border-r border-white/5 bg-surface shrink-0">
  <!-- Logo -->
  <div class="px-5 py-5">
    <a href="/dashboard" class="flex items-center gap-2.5 group">
      <Logo size={28} />
      <span class="text-[15px] font-bold tracking-tight text-white">Joules</span>
    </a>
  </div>

  <div class="mx-4 h-px bg-white/5"></div>

  <!-- Nav -->
  <nav class="flex-1 px-3 py-4 space-y-0.5">
    {#each nav as item}
      {@const active = activePage === item.id}
      <a
        href={item.href}
        class="group flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-all duration-150
          {active
            ? 'bg-joule-500/10 text-joule-400 ring-1 ring-inset ring-joule-500/20'
            : 'text-slate-500 hover:text-slate-200 hover:bg-white/5'}"
      >
        <svg
          class="h-[17px] w-[17px] shrink-0 transition-colors {active ? 'text-joule-400' : 'text-slate-600 group-hover:text-slate-400'}"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="1.75"
        >
          {@html item.icon}
        </svg>
        {item.label}
      </a>
    {/each}

    {#if isAdmin}
      <div class="mx-3 my-3 h-px bg-white/5"></div>
      <a
        href="/admin"
        class="group flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-all duration-150
          {activePage === 'admin'
            ? 'bg-joule-500/10 text-joule-400 ring-1 ring-inset ring-joule-500/20'
            : 'text-slate-500 hover:text-slate-200 hover:bg-white/5'}"
      >
        <svg
          class="h-[17px] w-[17px] shrink-0 transition-colors {activePage === 'admin' ? 'text-joule-400' : 'text-slate-600 group-hover:text-slate-400'}"
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
  </nav>
</aside>

<!-- Mobile bottom nav -->
<nav class="lg:hidden fixed bottom-0 left-0 right-0 z-50 border-t border-white/5 bg-surface flex items-stretch h-16 safe-area-pb">
  {#each mobileNav as item}
    {@const active = activePage === item.id}
    <a
      href={item.href}
      class="flex flex-1 flex-col items-center justify-center gap-1 transition-colors
        {active ? 'text-joule-400' : 'text-slate-600 hover:text-slate-400'}"
    >
      <svg
        class="h-5 w-5 shrink-0"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width={active ? '2' : '1.75'}
      >
        {@html item.icon}
      </svg>
      <span class="text-[10px] font-medium leading-none">{item.label}</span>
    </a>
  {/each}
</nav>
