<script lang="ts">
  import Logo from '$components/Logo.svelte';
  import TipsWidget from '$components/TipsWidget.svelte';
  import ThemeToggle from '$components/ThemeToggle.svelte';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';

  interface ChatMessage {
    id: string;
    role: 'user' | 'assistant';
    content: string;
    created_at: string;
  }

  interface ChatResponse {
    id: string;
    role: 'user' | 'assistant';
    content: string;
    created_at: string;
  }

  let messages = $state<ChatMessage[]>([]);
  let tips = $state<string | null>(null);
  let input = $state('');
  let loading = $state(false);
  let loadingHistory = $state(true);
  let messagesContainer: HTMLDivElement | undefined = $state();

  function scrollToBottom() {
    if (messagesContainer) {
      messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }
  }

  function formatTime(iso: string) {
    return new Date(iso).toLocaleTimeString('en-US', {
      hour: 'numeric',
      minute: '2-digit',
      hour12: true
    });
  }

  async function send() {
    if (!input.trim() || loading) return;
    const content = input.trim();
    input = '';
    const tempId = `temp-${Date.now()}`;
    messages = [...messages, { id: tempId, role: 'user', content, created_at: new Date().toISOString() }];
    scrollToBottom();
    loading = true;
    try {
      const res = await api.post<ChatResponse>('/coach/chat', { content });
      messages = [...messages, {
        id: res.id,
        role: 'assistant',
        content: res.content,
        created_at: res.created_at
      }];
    } catch {
      messages = messages.filter(m => m.id !== tempId);
    } finally {
      loading = false;
      scrollToBottom();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      send();
    }
  }

  function autoResize(e: Event) {
    const el = e.target as HTMLTextAreaElement;
    el.style.height = 'auto';
    el.style.height = Math.min(el.scrollHeight, 96) + 'px';
  }

  onMount(() => {
    const unsub = authToken.subscribe((token) => {
      if (!token) goto('/login');
    });

    async function init() {
      try {
        const [history, tipsRes] = await Promise.all([
          api.get<ChatMessage[]>('/coach/chat'),
          api.get<{ tips: string }>('/coach/tips')
        ]);
        messages = history;
        tips = tipsRes.tips;
      } catch {} finally {
        loadingHistory = false;
        setTimeout(scrollToBottom, 50);
      }
    }

    init();
    return unsub;
  });

  $effect(() => {
    if (messages.length > 0) {
      scrollToBottom();
    }
  });
</script>

<div class="flex h-screen">
  <aside class="hidden w-64 border-r border-slate-800 bg-surface p-6 lg:block">
    <div class="flex items-center gap-3">
      <Logo size={32} />
      <span class="text-lg font-bold text-white">Joule</span>
    </div>
    <nav class="mt-8 space-y-1">
      <a
        href="/dashboard"
        class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" /></svg>
        Dashboard
      </a>
      <a
        href="/log"
        class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
        Log Meal
      </a>
      <a
        href="/coach"
        class="flex items-center gap-3 rounded-lg bg-slate-800 px-3 py-2 text-sm font-medium text-white"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" /></svg>
        Health Coach
      </a>
      <a
        href="/progress"
        class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" /></svg>
        Progress
      </a>
      <a
        href="/achievements"
        class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" /></svg>
        Achievements
      </a>
    </nav>
  </aside>

  <main class="flex flex-1 flex-col overflow-hidden">
    <div class="flex items-center justify-between border-b border-slate-800 px-6 py-4 lg:px-10">
      <div>
        <h1 class="text-2xl font-bold text-white">Health Coach</h1>
        <p class="mt-1 text-sm text-slate-400">Your AI-powered nutrition assistant</p>
      </div>
      <div class="flex items-center gap-2">
        <ThemeToggle />
        <button
          onclick={() => { authToken.set(null); goto('/login'); }}
          class="rounded-lg border border-slate-700 px-3 py-1.5 text-sm text-slate-400 hover:text-white transition"
        >
          Sign out
        </button>
      </div>
    </div>

    {#if loadingHistory}
      <div class="flex flex-1 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-slate-700 border-t-joule-500"></div>
      </div>
    {:else if messages.length === 0}
      <div class="flex flex-1 flex-col items-center justify-center gap-8 px-6">
        <div class="max-w-md text-center">
          <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-joule-500/10">
            <svg class="h-8 w-8 text-joule-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" /></svg>
          </div>
          <h2 class="text-xl font-semibold text-white">Hi! I'm your Joule health coach.</h2>
          <p class="mt-2 text-sm text-slate-400">Ask me anything about nutrition, exercise, or your diet plan.</p>
        </div>
        {#if tips}
          <TipsWidget {tips} />
        {/if}
      </div>
    {:else}
      <div class="flex flex-1 flex-col overflow-hidden">
        <div bind:this={messagesContainer} class="flex-1 overflow-y-auto px-6 py-4 lg:px-10">
          <div class="mx-auto max-w-3xl space-y-4">
            {#each messages as msg}
              <div class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}">
                <div class="max-w-[75%] space-y-1">
                  <div class="flex items-center gap-2 {msg.role === 'user' ? 'justify-end' : ''}">
                    <span class="text-xs font-medium {msg.role === 'user' ? 'text-joule-400' : 'text-slate-400'}">
                      {msg.role === 'user' ? 'You' : 'Joule Coach'}
                    </span>
                    <span class="text-xs text-slate-600">{formatTime(msg.created_at)}</span>
                  </div>
                  <div class="rounded-2xl px-4 py-2.5 {msg.role === 'user'
                    ? 'rounded-br-md bg-joule-500 text-slate-900'
                    : 'rounded-bl-md border border-slate-800 bg-surface-lighter text-white'}">
                    <p class="whitespace-pre-wrap text-sm leading-relaxed">{msg.content}</p>
                  </div>
                </div>
              </div>
            {/each}
            {#if loading}
              <div class="flex justify-start">
                <div class="max-w-[75%] space-y-1">
                  <div class="flex items-center gap-2">
                    <span class="text-xs font-medium text-slate-400">Joule Coach</span>
                  </div>
                  <div class="rounded-2xl rounded-bl-md border border-slate-800 bg-surface-lighter px-4 py-3">
                    <div class="flex items-center gap-1">
                      <span class="h-2 w-2 animate-bounce rounded-full bg-slate-400" style="animation-delay: 0ms"></span>
                      <span class="h-2 w-2 animate-bounce rounded-full bg-slate-400" style="animation-delay: 150ms"></span>
                      <span class="h-2 w-2 animate-bounce rounded-full bg-slate-400" style="animation-delay: 300ms"></span>
                    </div>
                  </div>
                </div>
              </div>
            {/if}
          </div>
        </div>

        <div class="border-t border-slate-800 px-6 py-4 lg:px-10">
          <div class="mx-auto flex max-w-3xl items-end gap-3">
            <textarea
              bind:value={input}
              onkeydown={handleKeydown}
              oninput={autoResize}
              rows="1"
              placeholder="Ask your health coach..."
              disabled={loading}
              class="max-h-24 min-h-[42px] flex-1 resize-none rounded-xl border border-slate-700 bg-surface-light px-4 py-2.5 text-sm text-white placeholder:text-slate-500 focus:border-joule-500 focus:outline-none focus:ring-1 focus:ring-joule-500 disabled:opacity-50"
            ></textarea>
            <button
              onclick={send}
              disabled={!input.trim() || loading}
              aria-label="Send message"
              class="flex h-[42px] w-[42px] shrink-0 items-center justify-center rounded-xl bg-joule-500 text-slate-900 hover:bg-joule-400 transition disabled:opacity-50"
            >
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 12L3.269 3.126A59.768 59.768 0 0121.485 12 59.77 59.77 0 013.27 20.876L5.999 12zm0 0h7.5" /></svg>
            </button>
          </div>
        </div>
      </div>
    {/if}
  </main>
</div>
