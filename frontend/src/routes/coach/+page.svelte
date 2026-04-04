<script lang="ts">
  import Sidebar from '$components/Sidebar.svelte';
  import MarkdownRenderer from '$components/MarkdownRenderer.svelte';
  import ThemeToggle from '$components/ThemeToggle.svelte';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';
  import { showAchievement } from '$lib/achievements';
  import { goto } from '$app/navigation';
  import { onMount, tick } from 'svelte';

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

  interface Conversation {
    id: string;
    label: string;
    date: string;
    preview: string;
    messageIds: string[];
  }

  let allMessages = $state<ChatMessage[]>([]);
  let conversations = $state<Conversation[]>([]);
  let activeConversation = $state<string | null>(null);
  let displayMessages = $state<ChatMessage[]>([]);
  let input = $state('');
  let loading = $state(false);
  let loadingHistory = $state(true);
  let messagesContainer: HTMLDivElement | undefined = $state();
  let sidebarOpen = $state(false);

  function groupConversations(messages: ChatMessage[]): Conversation[] {
    if (messages.length === 0) return [];
    const SESSION_GAP_MS = 3 * 60 * 60 * 1000;
    const groups: Conversation[] = [];
    let current: ChatMessage[] = [];
    let lastTime = 0;

    const sorted = [...messages].reverse();
    for (const msg of sorted) {
      const msgTime = new Date(msg.created_at).getTime();
      if (lastTime > 0 && (lastTime - msgTime) > SESSION_GAP_MS && current.length > 0) {
        groups.push(buildConversation(current));
        current = [];
      }
      current.unshift(msg);
      lastTime = msgTime;
    }
    if (current.length > 0) {
      groups.push(buildConversation(current));
    }
    return groups;
  }

  function buildConversation(msgs: ChatMessage[]): Conversation {
    const first = msgs[0];
    const dateStr = new Date(first.created_at).toLocaleDateString('en-US', {
      month: 'short', day: 'numeric'
    });
    const firstUser = msgs.find(m => m.role === 'user');
    let preview = firstUser ? firstUser.content.slice(0, 60) : 'New conversation';
    if (preview.length >= 60) preview += '...';
    return {
      id: first.id,
      label: dateStr,
      date: first.created_at,
      preview,
      messageIds: msgs.map(m => m.id),
    };
  }

  function selectConversation(conv: Conversation) {
    activeConversation = conv.id;
    displayMessages = allMessages.filter(m => conv.messageIds.includes(m.id));
    sidebarOpen = false;
    tick().then(scrollToBottom);
  }

  let isNewChat = $state(false);

  function startNewChat() {
    activeConversation = null;
    displayMessages = [];
    sidebarOpen = false;
    isNewChat = true;
  }

  function refreshConversations() {
    conversations = groupConversations(allMessages);
    if (conversations.length > 0 && !activeConversation && !isNewChat) {
      selectConversation(conversations[0]);
    }
  }

  function scrollToBottom() {
    if (messagesContainer) {
      messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }
  }

  function formatTime(iso: string) {
    return new Date(iso).toLocaleTimeString('en-US', {
      hour: 'numeric', minute: '2-digit', hour12: true
    });
  }

  function formatDate(iso: string) {
    return new Date(iso).toLocaleDateString('en-US', {
      month: 'short', day: 'numeric', year: 'numeric'
    });
  }

  async function checkAchievements() {
    try {
      const seenRaw = localStorage.getItem('seen_achievement_ids');
      const seen: string[] = seenRaw ? JSON.parse(seenRaw) : [];
      const seenSet = new Set<string>(seen);
      const unlocked = await api.post<{ id: string; title: string; description: string }[]>('/achievements/check', {});
      if (Array.isArray(unlocked)) {
        const newIds: string[] = [];
        for (const a of unlocked) {
          if (!seenSet.has(a.id)) {
            showAchievement({ id: a.id, title: a.title, description: a.description });
            newIds.push(a.id);
          }
        }
        if (newIds.length > 0) {
          localStorage.setItem('seen_achievement_ids', JSON.stringify([...seen, ...newIds]));
        }
      }
    } catch {}
  }

  async function send() {
    if (!input.trim() || loading) return;
    const content = input.trim();
    input = '';
    const tempId = `temp-${Date.now()}`;
    const optimistic: ChatMessage = { id: tempId, role: 'user', content, created_at: new Date().toISOString() };
    displayMessages = [...displayMessages, optimistic];
    scrollToBottom();
    loading = true;
    try {
      const res = await api.post<ChatResponse>('/coach/chat', { content });
      displayMessages = [...displayMessages, {
        id: res.id, role: 'assistant', content: res.content, created_at: res.created_at
      }];
      allMessages.unshift(optimistic);
      allMessages.unshift({ id: res.id, role: 'assistant', content: res.content, created_at: res.created_at });
      refreshConversations();
      isNewChat = false;
      checkAchievements();
    } catch {
      displayMessages = displayMessages.filter(m => m.id !== tempId);
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
    el.style.height = Math.min(el.scrollHeight, 120) + 'px';
  }

  onMount(() => {
    const unsub = authToken.subscribe((token) => {
      if (!token) goto('/login');
    });

    async function init() {
      try {
        const history = await api.get<ChatMessage[]>('/coach/chat');
        allMessages = history;
        refreshConversations();
      } catch {} finally {
        loadingHistory = false;
        tick().then(() => setTimeout(scrollToBottom, 100));
      }
    }

    init();
    return unsub;
  });

  $effect(() => {
    if (displayMessages.length > 0) {
      tick().then(scrollToBottom);
    }
  });
</script>

<div class="flex h-screen">
  <Sidebar activePage="coach" />

  {#if sidebarOpen}
    <div class="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm lg:hidden" onclick={() => sidebarOpen = false} role="presentation"></div>
  {/if}

  <div class="fixed inset-y-0 left-0 top-0 z-50 flex w-72 transform flex-col border-r border-border bg-secondary transition-transform duration-200 lg:static lg:z-auto lg:translate-x-0 {sidebarOpen ? 'translate-x-0' : '-translate-x-full'}" style="margin-left: 0;">
    <div class="flex items-center justify-between border-b border-border px-4 py-3">
      <h2 class="text-sm font-semibold text-foreground">Conversations</h2>
      <button onclick={startNewChat} class="flex h-8 w-8 items-center justify-center rounded-lg hover:bg-accent transition" title="New chat">
        <svg class="h-4 w-4 text-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" /></svg>
      </button>
    </div>
    <div class="flex-1 overflow-y-auto">
      {#if conversations.length === 0}
        <p class="px-4 py-8 text-center text-xs text-muted-foreground">No conversations yet</p>
      {:else}
        {#each conversations as conv, i}
          {@const isActive = activeConversation === conv.id}
          <button
            onclick={() => selectConversation(conv)}
            class="w-full border-b border-border px-4 py-3 text-left transition hover:bg-accent/50 {isActive ? 'bg-primary/10 border-l-2 border-l-primary' : ''}"
          >
            <div class="flex items-center justify-between">
              <span class="text-xs font-medium text-foreground">{conv.label}</span>
              <span class="text-[10px] text-muted-foreground">{conv.messageIds.length} msgs</span>
            </div>
            <p class="mt-0.5 truncate text-xs text-muted-foreground">{conv.preview}</p>
          </button>
        {/each}
      {/if}
    </div>
  </div>

  <main class="flex flex-1 flex-col overflow-hidden min-w-0">
    <div class="flex items-center justify-between border-b border-border px-4 py-3 lg:px-6">
      <div class="flex items-center gap-3 min-w-0">
        <button onclick={() => sidebarOpen = !sidebarOpen} class="flex h-8 w-8 items-center justify-center rounded-lg hover:bg-accent transition lg:hidden">
          <svg class="h-5 w-5 text-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" /></svg>
        </button>
        <div class="min-w-0">
          <h1 class="font-display text-lg font-bold text-foreground">Health Coach</h1>
          <p class="text-xs text-muted-foreground truncate">Your AI-powered nutrition & fitness assistant</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <button
          onclick={startNewChat}
          class="hidden sm:flex items-center gap-1.5 rounded-xl border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:text-primary hover:border-primary/30 transition"
        >
          <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" /></svg>
          New Chat
        </button>
        <ThemeToggle />
        <button
          onclick={() => { authToken.set(null); goto('/login'); }}
          class="rounded-xl border border-border bg-accent/50 px-3 py-1.5 text-xs font-medium text-foreground hover:bg-accent/50 transition"
        >
          Sign out
        </button>
      </div>
    </div>

    {#if loadingHistory}
      <div class="flex flex-1 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-border border-t-primary"></div>
      </div>
    {:else}
      <div class="flex flex-1 flex-col overflow-hidden">
        {#if displayMessages.length === 0 && activeConversation === null}
          <div class="flex flex-1 flex-col items-center justify-center gap-6 px-6 pb-4">
            <div class="max-w-md text-center">
              <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-primary/20 to-primary/5">
                <svg class="h-8 w-8 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" /></svg>
              </div>
              <h2 class="text-xl font-semibold text-foreground">Hi! I'm your Joules health coach.</h2>
              <p class="mt-2 text-sm text-muted-foreground">I can log your meals, track workouts, and give personalized advice. Just tell me what you're up to!</p>
            </div>
            <div class="flex flex-wrap justify-center gap-2 max-w-lg">
              {#each ['Log my breakfast: 2 eggs and toast', 'I just ran for 30 minutes', 'How am I doing today?', 'Give me a high-protein dinner idea'] as suggestion}
                <button
                  onclick={() => { input = suggestion; send(); }}
                  class="rounded-xl border border-border px-4 py-2.5 text-xs text-foreground hover:border-primary/50 hover:bg-primary/5 transition"
                >
                  {suggestion}
                </button>
              {/each}
            </div>
            <div class="flex items-center gap-6 mt-2 text-xs text-muted-foreground">
              <div class="flex items-center gap-1.5">
                <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                <span>Real-time logging</span>
              </div>
              <div class="flex items-center gap-1.5">
                <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" /></svg>
                <span>Nutrition lookup</span>
              </div>
              <div class="flex items-center gap-1.5">
                <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" /></svg>
                <span>Smart tracking</span>
              </div>
            </div>
          </div>
        {:else}
          <div bind:this={messagesContainer} class="flex-1 overflow-y-auto px-4 py-4 lg:px-6">
            <div class="mx-auto max-w-3xl space-y-4">
              {#each displayMessages as msg, i}
                {@const showDate = i === 0 || (new Date(msg.created_at).toDateString() !== new Date(displayMessages[i-1].created_at).toDateString())}
                {#if showDate}
                  <div class="flex items-center gap-3 py-2">
                    <div class="h-px flex-1 bg-border"></div>
                    <span class="text-[11px] text-muted-foreground">{formatDate(msg.created_at)}</span>
                    <div class="h-px flex-1 bg-border"></div>
                  </div>
                {/if}
                <div class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}">
                  <div class="max-w-[80%] min-w-0 space-y-1">
                    <div class="flex items-center gap-2 {msg.role === 'user' ? 'justify-end' : ''}">
                      {#if msg.role === 'assistant'}
                        <div class="flex h-5 w-5 items-center justify-center rounded-full bg-primary/15">
                          <svg class="h-3 w-3 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" /></svg>
                        </div>
                      {/if}
                      <span class="text-xs font-medium {msg.role === 'user' ? 'text-primary' : 'text-foreground'}">
                        {msg.role === 'user' ? 'You' : 'Joules'}
                      </span>
                      <span class="text-[11px] text-muted-foreground">{formatTime(msg.created_at)}</span>
                    </div>
                    <div class="rounded-2xl px-4 py-3 {msg.role === 'user'
                      ? 'rounded-br-sm bg-primary text-primary-foreground'
                      : 'rounded-bl-sm border border-border bg-card text-foreground'}">
                      {#if msg.role === 'user'}
                        <p class="whitespace-pre-wrap text-sm leading-relaxed">{msg.content}</p>
                      {:else}
                        <div class="text-sm leading-relaxed">
                          <MarkdownRenderer text={msg.content} />
                        </div>
                      {/if}
                    </div>
                  </div>
                </div>
              {/each}
              {#if loading}
                <div class="flex justify-start">
                  <div class="max-w-[80%] min-w-0 space-y-1">
                    <div class="flex items-center gap-2">
                      <div class="flex h-5 w-5 items-center justify-center rounded-full bg-primary/15">
                        <svg class="h-3 w-3 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" /></svg>
                      </div>
                      <span class="text-xs font-medium text-foreground">Joules</span>
                    </div>
                    <div class="rounded-2xl rounded-bl-sm border border-border bg-card px-4 py-3">
                      <div class="flex items-center gap-1">
                        <span class="h-2 w-2 animate-bounce rounded-full bg-muted-foreground" style="animation-delay: 0ms"></span>
                        <span class="h-2 w-2 animate-bounce rounded-full bg-muted-foreground" style="animation-delay: 150ms"></span>
                        <span class="h-2 w-2 animate-bounce rounded-full bg-muted-foreground" style="animation-delay: 300ms"></span>
                      </div>
                    </div>
                  </div>
                </div>
              {/if}
            </div>
          </div>
        {/if}

        <div class="border-t border-border px-4 pt-3 pb-24 lg:px-6 lg:pb-4" style="padding-bottom: calc(5.5rem + env(safe-area-inset-bottom, 0px));">
          <div class="mx-auto flex max-w-3xl items-end gap-3">
            <textarea
              bind:value={input}
              onkeydown={handleKeydown}
              oninput={autoResize}
              rows="1"
              placeholder="Tell Joules what you're eating, doing, or ask anything..."
              disabled={loading}
              class="max-h-[120px] min-h-[44px] flex-1 resize-none rounded-2xl border border-border bg-card px-4 py-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-ring/20 disabled:opacity-50 transition-colors"
            ></textarea>
            <button
              onclick={send}
              disabled={!input.trim() || loading}
              aria-label="Send message"
              class="flex h-[44px] w-[44px] shrink-0 items-center justify-center rounded-2xl bg-primary text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50"
            >
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 12L3.269 3.126A59.768 59.768 0 0121.485 12 59.77 59.77 0 013.27 20.876L5.999 12zm0 0h7.5" /></svg>
            </button>
          </div>
        </div>
      </div>
    {/if}
  </main>
</div>
