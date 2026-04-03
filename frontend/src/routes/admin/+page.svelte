<script lang="ts">
  import Sidebar from '$components/Sidebar.svelte';
  import ThemeToggle from '$components/ThemeToggle.svelte';
  import { authToken } from '$lib/stores';
  import { api } from '$lib/api';
  import { get } from 'svelte/store';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';

  interface UserRow {
    id: string;
    email: string;
    verified: boolean;
    approved: boolean;
    is_admin: boolean;
    created_at: string;
  }

  interface Settings {
    require_approval: boolean;
    ai_provider: string;
    ai_model: string;
    vision_model: string;
    ocr_model: string;
    routing_model: string;
    custom_base_url: string;
    custom_api_key: string;
    smtp_configured: boolean;
    smtp_host: string;
    smtp_user: string;
    smtp_port: string;
    app_url: string;
    port: string;
  }

  let users = $state<UserRow[]>([]);
  let settings = $state<Settings>({
    require_approval: false,
    ai_provider: 'openai',
    ai_model: '',
    vision_model: '',
    ocr_model: '',
    routing_model: '',
    custom_base_url: '',
    custom_api_key: '',
    smtp_configured: false,
    smtp_host: '',
    smtp_user: '',
    smtp_port: '',
    app_url: '',
    port: '',
  });
  let loading = $state(true);
  let currentUserID = $state('');
  let actionLoading = $state<string | null>(null);
  let settingsLoading = $state(false);
  let aiSaving = $state(false);
  let aiSaved = $state(false);

  // Editable AI settings (local state)
  let editAIProvider = $state('openai');
  let editVisionModel = $state('');
  let editAIModel = $state('');
  let editOCRModel = $state('');
  let editCustomBaseURL = $state('');
  let editCustomAPIKey = $state('');

  let modelsList = $state<Array<{ id: string; owned_by?: string; created?: number }>>([]);
  let modelsLoading = $state(false);
  let modelsError = $state('');
  let testResult = $state<{ success: boolean; latency_ms: number; response_preview: string; error: string } | null>(null);
  let testLoading = $state(false);

  // SMTP settings
  let editSMTPHost = $state('');
  let editSMTPPort = $state('');
  let editSMTPUser = $state('');
  let editSMTPPass = $state('');
  let smtpSaving = $state(false);
  let smtpSaved = $state(false);
  let showSMTPForm = $state(false);

  // Restart modal
  let showRestartModal = $state(false);
  let restarting = $state(false);

  // AI Prompts
  let prompts = $state<Record<string, string>>({});
  let promptsLoading = $state(false);
  let promptSaving = $state(false);
  let promptSaved = $state(false);
  let showPrompts = $state(false);
  let editingPrompt = $state<string | null>(null);

  // Feature Flags
  let features = $state<Record<string, boolean>>({});
  let featuresLoading = $state(false);
  let featureSaving = $state(false);
  let featureSaved = $state(false);
  let showFeatures = $state(false);

  // Coach Config
  let coachConfig = $state({ max_iterations: 3, context_window_size: 20, max_message_length: 2000 });
  let coachConfigLoading = $state(false);
  let coachConfigSaving = $state(false);
  let coachConfigSaved = $state(false);
  let showCoachConfig = $state(false);

  // TDEE Config
  let tdeeConfig = $state({
    activity_multipliers: {} as Record<string, number>,
    objective_multipliers: {} as Record<string, number>,
    macro_splits: {} as Record<string, Record<string, number>>,
    min_calorie_target: 1200,
  });
  let tdeeConfigLoading = $state(false);
  let tdeeConfigSaving = $state(false);
  let tdeeConfigSaved = $state(false);
  let showTDEEConfig = $state(false);

  // System Health
  let showHealthcheck = $state(false);
  let healthData = $state<any>(null);
  let healthLoading = $state(false);

  // Food Database
  let foodsCount = $state(0);
  let importStatus = $state('');
  let importing = $state(false);
  let importResult = $state<{ imported: number; skipped: number } | null>(null);

  async function loadFoodsStats() {
    try {
      const data = await api.get<{ count: number; import_status: string }>('/admin/foods/stats');
      foodsCount = data.count;
      importStatus = data.import_status;
    } catch {}
  }

  async function handleFoodImport(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    importing = true;
    importResult = null;
    try {
      const formData = new FormData();
      formData.append('file', file);
      const token = get(authToken);
      const res = await fetch('/api/admin/foods/import', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` },
        body: formData,
      });
      const data = await res.json();
      importResult = data.data;
      loadFoodsStats();
    } catch {
      importResult = { imported: 0, skipped: 0 };
    } finally {
      importing = false;
    }
  }

  // Nutrition Cache
  let cacheEntries = $state<any[]>([]);
  let cacheTotal = $state(0);
  let cacheSearch = $state('');
  let cacheLoading = $state(false);

  async function loadNutritionCache() {
    cacheLoading = true;
    try {
      const params = cacheSearch ? `?q=${encodeURIComponent(cacheSearch)}` : '';
      const data = await api.get<{ entries: any[]; total: number }>(`/admin/nutrition-cache${params}`);
      cacheEntries = data.entries;
      cacheTotal = data.total;
    } catch { cacheEntries = []; }
    finally { cacheLoading = false; }
  }

  async function deleteCacheEntry(id: string) {
    try {
      await api.del(`/admin/nutrition-cache/${id}`);
      loadNutritionCache();
    } catch {}
  }

  async function clearCache() {
    if (!confirm('Clear all nutrition cache entries?')) return;
    try {
      await api.post('/admin/nutrition-cache/clear', {});
      loadNutritionCache();
    } catch {}
  }

  onMount(() => {
    const unsub = authToken.subscribe((token) => {
      if (!token) {
        goto('/login');
        return;
      }
    });

    (async () => {
      try {
        // Check if current user is admin
        const profile = await api.get<{ name: string; is_admin: boolean; onboarding_complete: boolean }>('/user/profile');
        if (!profile.is_admin) {
          goto('/dashboard');
          return;
        }

        // Get current user ID from /auth/me
        const me = await api.get<{ id: string }>('/auth/me');
        currentUserID = me.id;

        const [usersData, settingsData, bannersData] = await Promise.all([
          api.get<UserRow[]>('/admin/users'),
          api.get<Settings>('/admin/settings'),
          api.get<Banner[]>('/admin/banners'),
        ]);
        users = usersData;
        settings = settingsData;
        editAIProvider = settingsData.ai_provider || 'openai';
        editVisionModel = settingsData.vision_model || '';
        editAIModel = settingsData.ai_model || '';
        editOCRModel = settingsData.ocr_model || '';
        editCustomBaseURL = settingsData.custom_base_url || '';
        editCustomAPIKey = settingsData.custom_api_key || '';
        editSMTPHost = settingsData.smtp_host || '';
        editSMTPPort = settingsData.smtp_port || '';
        editSMTPUser = settingsData.smtp_user || '';
        banners = bannersData ?? [];
      } catch {
        goto('/dashboard');
      } finally {
        loading = false;
      }
    })();

    // Load food DB stats and nutrition cache independently
    loadFoodsStats();
    loadNutritionCache();

    return unsub;
  });

  async function approveUser(id: string) {
    actionLoading = id + ':approve';
    try {
      await api.post(`/admin/users/${id}/approve`, {});
      users = users.map(u => u.id === id ? { ...u, approved: true } : u);
    } catch {}
    finally { actionLoading = null; }
  }

  async function unapproveUser(id: string) {
    actionLoading = id + ':unapprove';
    try {
      await api.post(`/admin/users/${id}/unapprove`, {});
      users = users.map(u => u.id === id ? { ...u, approved: false } : u);
    } catch {}
    finally { actionLoading = null; }
  }

  async function deleteUser(id: string, email: string) {
    if (!confirm(`Delete user ${email}? This cannot be undone.`)) return;
    actionLoading = id + ':delete';
    try {
      await api.del(`/admin/users/${id}`);
      users = users.filter(u => u.id !== id);
    } catch {}
    finally { actionLoading = null; }
  }

  async function makeAdmin(id: string, email: string) {
    if (!confirm(`Promote ${email} to admin?`)) return;
    actionLoading = id + ':makeadmin';
    try {
      await api.post(`/admin/users/${id}/make-admin`, {});
      users = users.map(u => u.id === id ? { ...u, is_admin: true, approved: true } : u);
    } catch {}
    finally { actionLoading = null; }
  }

  async function removeAdmin(id: string, email: string) {
    if (!confirm(`Remove admin rights from ${email}?`)) return;
    actionLoading = id + ':removeadmin';
    try {
      await api.post(`/admin/users/${id}/remove-admin`, {});
      users = users.map(u => u.id === id ? { ...u, is_admin: false } : u);
    } catch {}
    finally { actionLoading = null; }
  }

  async function toggleRequireApproval() {
    settingsLoading = true;
    const newVal = !settings.require_approval;
    try {
      await api.put('/admin/settings', { require_approval: newVal });
      settings = { ...settings, require_approval: newVal };
    } catch {}
    finally { settingsLoading = false; }
  }

  async function fetchModels() {
    modelsLoading = true;
    modelsError = '';
    modelsList = [];
    try {
      const body: any = { provider: editAIProvider };
      if (editAIProvider === 'custom' && editCustomBaseURL) {
        body.base_url = editCustomBaseURL;
      }
      if (editAIProvider === 'custom' && editCustomAPIKey) {
        body.api_key = editCustomAPIKey;
      }
      const res = await api.post<Array<{ id: string; owned_by?: string; created?: number }>>('/admin/models', body);
      modelsList = res || [];
      modelsList.sort((a: any, b: any) => (b.created || 0) - (a.created || 0));
    } catch (e: any) {
      modelsError = e.message || 'Failed to fetch models';
    }
    modelsLoading = false;
  }

  async function testAI() {
    testLoading = true;
    testResult = null;
    try {
      const res = await api.post<{ success: boolean; latency_ms: number; response_preview: string; error: string }>('/admin/ai/test', {
        provider: editAIProvider,
        model: editAIModel,
        api_key: editAIProvider === 'custom' ? editCustomAPIKey : undefined,
        base_url: editAIProvider === 'custom' ? editCustomBaseURL : undefined,
      });
      testResult = res;
    } catch (e: any) {
      testResult = { success: false, latency_ms: 0, response_preview: '', error: e.message || 'Test failed' };
    }
    testLoading = false;
  }

  async function loadHealthcheck() {
    healthLoading = true;
    try {
      healthData = await api.get('/admin/healthcheck');
    } catch (e) {
      console.error('Failed to load healthcheck', e);
    }
    healthLoading = false;
  }

  async function saveAISettings() {
    aiSaving = true;
    aiSaved = false;
    try {
      await api.put('/admin/settings', {
        ai_provider: editAIProvider,
        ai_model: editAIModel,
        vision_model: editVisionModel,
        ocr_model: editOCRModel,
        custom_base_url: editCustomBaseURL,
        custom_api_key: editCustomAPIKey,
      });
      settings = {
        ...settings,
        ai_provider: editAIProvider,
        ai_model: editAIModel,
        vision_model: editVisionModel,
        ocr_model: editOCRModel,
        custom_base_url: editCustomBaseURL,
        custom_api_key: editCustomAPIKey,
      };
      aiSaved = true;
      setTimeout(() => { aiSaved = false; }, 3000);
    } catch (e) {
      console.error('Failed to save AI settings', e);
    }
    aiSaving = false;
  }

  async function saveSMTPSettings() {
    smtpSaving = true;
    smtpSaved = false;
    try {
      await api.put('/admin/settings', {
        require_approval: settings.require_approval,
        smtp_host: editSMTPHost,
        smtp_port: editSMTPPort,
        smtp_user: editSMTPUser,
        ...(editSMTPPass ? { smtp_pass: editSMTPPass } : {}),
      });
      settings = { ...settings, smtp_host: editSMTPHost, smtp_user: editSMTPUser, smtp_port: editSMTPPort, smtp_configured: !!editSMTPHost };
      editSMTPPass = '';
      smtpSaved = true;
      showSMTPForm = false;
      setTimeout(() => { smtpSaved = false; }, 3000);
    } catch {}
    finally { smtpSaving = false; }
  }

  // --- Banners ---
  interface Banner { id: string; title: string; message: string; type: string; expires_at?: string; created_at: string; }
  let banners = $state<Banner[]>([]);
  let newBannerTitle = $state('');
  let newBannerMsg = $state('');
  let newBannerType = $state('info');
  let newBannerExpiry = $state(''); // duration in hours, empty = never
  let bannerSaving = $state(false);

  // --- Logs ---
  interface LogEntry { id: number; level: string; category: string; message: string; details?: any; created_at: string; }
  let logs = $state<LogEntry[]>([]);
  let logCategory = $state('all');
  let logsLoading = $state(false);

  // --- Verify email ---
  async function verifyUser(id: string, email: string) {
    if (!confirm(`Mark ${email}'s email as verified?`)) return;
    actionLoading = id + ':verify';
    try {
      await api.post(`/admin/users/${id}/verify`, {});
      users = users.map(u => u.id === id ? { ...u, verified: true } : u);
    } catch {}
    finally { actionLoading = null; }
  }

  async function loadBanners() {
    try {
      const data = await api.get<Banner[]>('/admin/banners');
      banners = data ?? [];
    } catch {}
  }

  async function createBanner() {
    if (!newBannerMsg.trim()) return;
    bannerSaving = true;
    try {
      let expiresAt: string | undefined;
      if (newBannerExpiry) {
        const hrs = parseFloat(newBannerExpiry);
        if (!isNaN(hrs) && hrs > 0) {
          expiresAt = new Date(Date.now() + hrs * 3600 * 1000).toISOString();
        }
      }
      const b = await api.post<Banner>('/admin/banners', {
        title: newBannerTitle,
        message: newBannerMsg,
        type: newBannerType,
        ...(expiresAt ? { expires_at: expiresAt } : {}),
      });
      banners = [b, ...banners];
      newBannerTitle = ''; newBannerMsg = ''; newBannerType = 'info'; newBannerExpiry = '';
    } catch {}
    finally { bannerSaving = false; }
  }

  async function deleteBanner(id: string) {
    try {
      await api.del(`/admin/banners/${id}`);
      banners = banners.filter(b => b.id !== id);
    } catch {}
  }

  async function loadLogs() {
    logsLoading = true;
    try {
      const data = await api.get<LogEntry[]>(`/admin/logs?category=${logCategory}`);
      logs = data ?? [];
    } catch {}
    finally { logsLoading = false; }
  }

  async function loadPrompts() {
    promptsLoading = true;
    try {
      prompts = await api.get<Record<string, string>>('/admin/prompts');
      showPrompts = true;
    } catch { } finally { promptsLoading = false; }
  }

  async function savePrompt(key: string) {
    promptSaving = true;
    promptSaved = false;
    try {
      await api.put('/admin/prompts', { [key]: prompts[key] });
      promptSaved = true;
      editingPrompt = null;
      setTimeout(() => { promptSaved = false; }, 2000);
    } catch { } finally { promptSaving = false; }
  }

  async function loadFeatures() {
    featuresLoading = true;
    try {
      features = await api.get<Record<string, boolean>>('/admin/features');
      showFeatures = true;
    } catch { } finally { featuresLoading = false; }
  }

  async function saveFeatures() {
    featureSaving = true;
    featureSaved = false;
    try {
      await api.put('/admin/features', features);
      featureSaved = true;
      setTimeout(() => { featureSaved = false; }, 2000);
    } catch { } finally { featureSaving = false; }
  }

  async function loadCoachConfig() {
    coachConfigLoading = true;
    try {
      coachConfig = await api.get<{ max_iterations: number; context_window_size: number; max_message_length: number }>('/admin/coach-config');
      showCoachConfig = true;
    } catch { } finally { coachConfigLoading = false; }
  }

  async function saveCoachConfig() {
    coachConfigSaving = true;
    coachConfigSaved = false;
    try {
      await api.put('/admin/coach-config', coachConfig);
      coachConfigSaved = true;
      setTimeout(() => { coachConfigSaved = false; }, 2000);
    } catch { } finally { coachConfigSaving = false; }
  }

  async function loadTDEEConfig() {
    tdeeConfigLoading = true;
    try {
      tdeeConfig = await api.get<typeof tdeeConfig>('/admin/tdee-config');
      showTDEEConfig = true;
    } catch { } finally { tdeeConfigLoading = false; }
  }

  async function saveTDEEConfig() {
    tdeeConfigSaving = true;
    tdeeConfigSaved = false;
    try {
      await api.put('/admin/tdee-config', tdeeConfig);
      tdeeConfigSaved = true;
      setTimeout(() => { tdeeConfigSaved = false; }, 2000);
    } catch { } finally { tdeeConfigSaving = false; }
  }

  const promptLabels: Record<string, string> = {
    'prompt_vision': 'Vision Food Identification',
    'prompt_ocr': 'OCR Text Parsing',
    'prompt_coach': 'Coach Chat System Prompt',
    'prompt_tips': 'Daily Tips Prompt',
    'prompt_nutrition_lookup': 'Nutrition Lookup Parse',
    'prompt_compact_l1': 'Context Compaction (Detailed)',
    'prompt_compact_l2': 'Context Compaction (Aggressive)',
  };

  const featureKeys = ['coach', 'ai_food_id', 'barcode', 'groups', 'gamification', 'fasting', 'achievements', 'steps', 'export', 'recipes', 'tips', 'notifications'];

  const featureLabels: Record<string, string> = {
    'coach': 'AI Coach Chat',
    'ai_food_id': 'AI Food Identification',
    'barcode': 'Barcode Scanning',
    'groups': 'Social Groups',
    'gamification': 'Gamification & Points',
    'fasting': 'Intermittent Fasting',
    'achievements': 'Achievements',
    'steps': 'Step Tracking',
    'export': 'Data Export (CSV)',
    'recipes': 'Recipes',
    'tips': 'Daily Tips',
    'notifications': 'Push Notifications',
  };

  const activityLevels: [string, string][] = [
    ['sedentary', 'Sedentary'],
    ['light', 'Lightly Active'],
    ['moderate', 'Moderately Active'],
    ['active', 'Active'],
    ['very_active', 'Very Active'],
  ];

  const objectives: [string, string][] = [
    ['cut_fat', 'Cut Fat'],
    ['feel_better', 'Feel Better'],
    ['maintain', 'Maintain'],
    ['build_muscle', 'Build Muscle'],
  ];

  const dietPlans: [string, string][] = [
    ['calorie_deficit', 'Calorie Deficit'],
    ['keto', 'Keto'],
    ['intermittent_fasting', 'Intermittent Fasting'],
    ['paleo', 'Paleo'],
    ['mediterranean', 'Mediterranean'],
    ['balanced', 'Balanced'],
  ];

  const macroKeys = ['carbs', 'protein', 'fat'];

  async function confirmRestart() {
    restarting = true;
    try {
      await api.post('/admin/restart', {});
    } catch {}
    // Wait a moment then close modal — server is restarting
    setTimeout(() => {
      restarting = false;
      showRestartModal = false;
    }, 2000);
  }

  function formatDate(iso: string) {
    return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  }
</script>

<div class="flex min-h-screen">
  <Sidebar activePage="admin" isAdmin={true} />

  <main class="flex-1 min-w-0 overflow-y-auto p-6 pb-20 lg:p-10 lg:pb-10">
    {#if loading}
      <div class="flex h-64 items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-border border-t-primary"></div>
      </div>
    {:else}
      <div class="flex items-center justify-between mb-8">
        <div>
          <h1 class="text-2xl font-bold text-foreground">Admin</h1>
          <p class="mt-1 text-sm text-foreground">Manage users and server settings</p>
        </div>
        <div class="flex items-center gap-2">
          <ThemeToggle />
          <button
            onclick={() => { authToken.set(null); goto('/login'); }}
            class="rounded-lg border border-border px-3 py-1.5 text-sm text-foreground hover:text-foreground transition"
          >
            Sign out
          </button>
        </div>
      </div>

      <!-- Server Info -->
      <div class="mb-6 rounded-xl border border-border bg-card p-5">
        <h2 class="text-sm font-semibold text-primary mb-4">Server Info</h2>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <p class="text-xs text-muted-foreground mb-0.5">App URL</p>
            <p class="text-sm text-foreground font-mono">{settings.app_url || '—'}</p>
          </div>
          <div>
            <p class="text-xs text-muted-foreground mb-0.5">Port</p>
            <p class="text-sm text-foreground font-mono">{settings.port || '—'}</p>
          </div>
          <div>
            <p class="text-xs text-muted-foreground mb-0.5">AI Provider</p>
            <p class="text-sm text-foreground">{settings.ai_provider || '—'}</p>
          </div>
          <div>
            <p class="text-xs text-muted-foreground mb-0.5">AI Model</p>
            <p class="text-sm text-foreground">{settings.ai_model || '—'}</p>
          </div>
          <div>
            <p class="text-xs text-muted-foreground mb-0.5">SMTP</p>
            {#if settings.smtp_configured}
              <span class="inline-flex items-center rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-400">Configured</span>
            {:else}
              <span class="inline-flex items-center rounded-full bg-accent/50 px-2 py-0.5 text-xs font-medium text-foreground">Not configured</span>
            {/if}
          </div>
        </div>
      </div>

      <!-- Settings -->
      <div class="mb-6 rounded-xl border border-border bg-card p-5">
        <h2 class="text-sm font-semibold text-primary mb-4">Server Settings</h2>
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-foreground">Require Approval for New Signups</p>
            <p class="text-xs text-foreground mt-0.5">New accounts will be unapproved until an admin approves them</p>
          </div>
          <button
            onclick={toggleRequireApproval}
            disabled={settingsLoading}
            class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none disabled:opacity-50
              {settings.require_approval ? 'bg-primary' : 'bg-accent'}"
          >
            <span
              class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform
                {settings.require_approval ? 'translate-x-6' : 'translate-x-1'}"
            ></span>
          </button>
        </div>
      </div>

      <!-- AI Settings -->
      <div class="mb-6 rounded-xl border border-border bg-card p-5">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-sm font-semibold text-primary">AI Settings</h2>
          {#if aiSaved}
            <span class="text-sm text-green-400">Saved ✓</span>
          {/if}
        </div>

        <!-- Provider -->
        <div class="mb-4">
          <label for="ai-provider" class="mb-1.5 block text-xs font-medium text-foreground">Provider</label>
          <select
            id="ai-provider"
            bind:value={editAIProvider}
            onchange={() => { editVisionModel = ''; editAIModel = ''; editOCRModel = ''; modelsList = []; testResult = null; }}
            class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring sm:w-48"
          >
            <option value="openai">OpenAI</option>
            <option value="anthropic">Anthropic</option>
            <option value="custom">Custom (OpenAI-compatible)</option>
          </select>
        </div>

        <!-- Custom provider fields -->
        {#if editAIProvider === 'custom'}
          <div class="mb-4 space-y-3">
            <div>
              <label class="block text-xs font-medium text-foreground mb-1">Custom Base URL</label>
              <input type="text" bind:value={editCustomBaseURL} placeholder="https://api.deepseek.com"
                class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none" />
            </div>
            <div>
              <label class="block text-xs font-medium text-foreground mb-1">API Key</label>
              <input type="password" bind:value={editCustomAPIKey} placeholder="sk-..."
                class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none" />
            </div>
          </div>
        {/if}

        <!-- Fetch models button -->
        <div class="mb-4">
          <button
            onclick={fetchModels}
            disabled={modelsLoading}
            class="text-xs px-3 py-1.5 rounded-lg border border-border bg-secondary hover:bg-accent text-foreground disabled:opacity-50 transition"
          >
            {modelsLoading ? 'Loading...' : '↻ Fetch Available Models'}
          </button>
          {#if modelsError}
            <p class="text-xs text-red-500 mt-1">{modelsError}</p>
          {/if}
        </div>

        <!-- Model pickers -->
        <div class="space-y-3 mb-4">
          <div>
            <label for="vision-model" class="mb-1.5 block text-xs font-medium text-foreground">
              Vision Model
              <span class="ml-1 text-muted-foreground">— photo analysis</span>
            </label>
            <select
              id="vision-model"
              bind:value={editVisionModel}
              class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
            >
              <option value="">Same as Primary Model</option>
              {#each modelsList as m}
                <option value={m.id}>{m.id}</option>
              {/each}
            </select>
            {#if !editVisionModel}
              <p class="mt-1 text-xs text-muted-foreground">Falls back to Primary Model if not set</p>
            {/if}
          </div>

          <div>
            <label for="ai-model" class="mb-1.5 block text-xs font-medium text-foreground">
              Primary Model
              <span class="ml-1 text-muted-foreground">— chat, tips, tools</span>
            </label>
            <select
              id="ai-model"
              bind:value={editAIModel}
              class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
            >
              <option value="">Select a model...</option>
              {#each modelsList as m}
                <option value={m.id}>{m.id}</option>
              {/each}
            </select>
            {#if modelsList.length === 0 && !modelsLoading}
              <p class="mt-1 text-xs text-muted-foreground">Click "Fetch Available Models" above to load models from the API</p>
            {/if}
          </div>

          <div>
            <label for="ocr-model" class="mb-1.5 block text-xs font-medium text-foreground">
              OCR Model
              <span class="ml-1 text-muted-foreground">— text parsing</span>
            </label>
            <select
              id="ocr-model"
              bind:value={editOCRModel}
              class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
            >
              <option value="">Same as Primary Model</option>
              {#each modelsList as m}
                <option value={m.id}>{m.id}</option>
              {/each}
            </select>
            {#if !editOCRModel}
              <p class="mt-1 text-xs text-muted-foreground">Falls back to Primary Model if not set</p>
            {/if}
          </div>
        </div>

        <!-- Manual model name input -->
        <details class="mb-4">
          <summary class="text-xs text-muted-foreground cursor-pointer hover:text-foreground transition">Or type a model name manually</summary>
          <div class="space-y-2 mt-2">
            <input type="text" bind:value={editVisionModel} placeholder="Vision model (e.g. gpt-4.1-mini-2025-04-14)"
              class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:border-ring focus:outline-none" />
            <input type="text" bind:value={editAIModel} placeholder="Primary model (e.g. gpt-5.4-mini-2026-03-17)"
              class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:border-ring focus:outline-none" />
            <input type="text" bind:value={editOCRModel} placeholder="OCR model (e.g. gpt-4.1-nano)"
              class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:border-ring focus:outline-none" />
          </div>
        </details>

        <!-- Actions -->
        <div class="flex items-center gap-3">
          <button
            onclick={saveAISettings}
            disabled={aiSaving}
            class="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50"
          >
            {aiSaving ? 'Saving…' : 'Save AI Settings'}
          </button>
          <button
            onclick={testAI}
            disabled={testLoading || !editAIModel}
            class="rounded-lg border border-border bg-secondary px-4 py-2 text-sm font-medium text-foreground hover:bg-accent disabled:opacity-50 transition"
          >
            {testLoading ? 'Testing…' : 'Test Connection'}
          </button>
        </div>

        <!-- Test result -->
        {#if testResult}
          <div class="mt-3 p-3 rounded-lg border {testResult.success ? 'border-green-500/30 bg-green-500/10' : 'border-red-500/30 bg-red-500/10'}">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium text-foreground">{testResult.success ? '✓ Connected' : '✗ Failed'}</span>
              {#if testResult.success}
                <span class="text-xs text-muted-foreground">{testResult.latency_ms}ms</span>
              {/if}
            </div>
            {#if testResult.response_preview}
              <p class="text-sm text-muted-foreground mt-1">Response: "{testResult.response_preview}"</p>
            {/if}
            {#if testResult.error}
              <p class="text-sm text-red-400 mt-1">{testResult.error}</p>
            {/if}
          </div>
        {/if}
      </div>

      <!-- SMTP Settings -->
      <div class="mb-6 rounded-xl border border-border bg-card p-5">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-sm font-semibold text-primary">SMTP / Email</h2>
          <div class="flex items-center gap-2">
            {#if smtpSaved}
              <span class="text-sm text-green-400">Saved!</span>
            {/if}
            <button
              onclick={() => { showSMTPForm = !showSMTPForm; }}
              class="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:text-foreground hover:bg-accent transition"
            >{showSMTPForm ? 'Cancel' : 'Edit'}</button>
          </div>
        </div>
        {#if !showSMTPForm}
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <div>
              <p class="text-xs text-muted-foreground mb-0.5">Host</p>
              <p class="text-sm text-foreground font-mono">{settings.smtp_host || '—'}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground mb-0.5">Port</p>
              <p class="text-sm text-foreground font-mono">{settings.smtp_port || '—'}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground mb-0.5">User</p>
              <p class="text-sm text-foreground font-mono">{settings.smtp_user || '—'}</p>
            </div>
          </div>
        {:else}
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 mb-3">
            <div>
              <label class="mb-1.5 block text-xs font-medium text-foreground">Host</label>
              <input type="text" bind:value={editSMTPHost} placeholder="mail.example.com"
                class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none" />
            </div>
            <div>
              <label class="mb-1.5 block text-xs font-medium text-foreground">Port</label>
              <input type="text" bind:value={editSMTPPort} placeholder="465 or 587"
                class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none" />
            </div>
            <div>
              <label class="mb-1.5 block text-xs font-medium text-foreground">Username</label>
              <input type="text" bind:value={editSMTPUser} placeholder="hello@example.com"
                class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none" />
            </div>
            <div>
              <label class="mb-1.5 block text-xs font-medium text-foreground">Password <span class="text-muted-foreground">(leave blank to keep current)</span></label>
              <input type="password" bind:value={editSMTPPass} placeholder="••••••••"
                class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none" />
            </div>
          </div>
          <button
            onclick={saveSMTPSettings}
            disabled={smtpSaving}
            class="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50"
          >{smtpSaving ? 'Saving…' : 'Save SMTP'}</button>
        {/if}
      </div>

      <!-- AI Prompts -->
      <div class="mb-6 rounded-2xl border border-border bg-card p-5">
        <div class="flex items-center justify-between mb-4">
          <h2 class="font-display text-lg font-bold text-foreground">AI Prompts</h2>
          <div class="flex items-center gap-2">
            {#if promptSaved}
              <span class="text-sm text-green-400 flex items-center gap-1">
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
                Saved!
              </span>
            {/if}
            <button
              onclick={loadPrompts}
              disabled={promptsLoading}
              class="px-4 py-2 rounded-xl bg-primary text-primary-foreground font-semibold text-sm hover:bg-primary/80 transition disabled:opacity-50"
            >{promptsLoading ? 'Loading…' : 'Load Prompts'}</button>
          </div>
        </div>

        {#if showPrompts}
          <div class="space-y-3">
            {#each Object.keys(promptLabels) as key}
              <div class="rounded-lg border border-border bg-secondary p-4">
                <div class="flex items-center justify-between mb-2">
                  <button
                    onclick={() => { editingPrompt = editingPrompt === key ? null : key; }}
                    class="text-sm font-medium text-foreground hover:text-primary transition"
                  >
                    <span class="mr-2 inline-block transition-transform {editingPrompt === key ? 'rotate-90' : ''}">▶</span>
                    {promptLabels[key]}
                  </button>
                  {#if editingPrompt === key}
                    <button
                      onclick={() => savePrompt(key)}
                      disabled={promptSaving}
                      class="rounded-lg bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50"
                    >{promptSaving ? 'Saving…' : 'Save'}</button>
                  {/if}
                </div>
                {#if editingPrompt === key}
                  <textarea
                    bind:value={prompts[key]}
                    rows="8"
                    class="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-ring focus:outline-none resize-y font-mono"
                  ></textarea>
                {:else}
                  <p class="text-xs text-muted-foreground truncate">{prompts[key]?.slice(0, 120) ?? '—'}{prompts[key]?.length > 120 ? '…' : ''}</p>
                {/if}
              </div>
            {/each}
          </div>
        {:else}
          <p class="text-sm text-muted-foreground">Click "Load Prompts" to view and edit AI system prompts.</p>
        {/if}
      </div>

      <!-- Feature Flags -->
      <div class="mb-6 rounded-2xl border border-border bg-card p-5">
        <div class="flex items-center justify-between mb-4">
          <h2 class="font-display text-lg font-bold text-foreground">Feature Flags</h2>
          <div class="flex items-center gap-2">
            {#if featureSaved}
              <span class="text-sm text-green-400 flex items-center gap-1">
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
                Saved!
              </span>
            {/if}
            <button
              onclick={loadFeatures}
              disabled={featuresLoading}
              class="px-4 py-2 rounded-xl bg-primary text-primary-foreground font-semibold text-sm hover:bg-primary/80 transition disabled:opacity-50"
            >{featuresLoading ? 'Loading…' : 'Load Features'}</button>
          </div>
        </div>

        {#if showFeatures}
          <div class="space-y-3">
            {#each featureKeys as key}
              <div class="flex items-center justify-between rounded-lg border border-border bg-secondary px-4 py-3">
                <span class="text-sm font-medium text-foreground">{featureLabels[key] ?? key}</span>
                <button
                  onclick={() => { features = { ...features, [key]: !features[key] }; }}
                  class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none
                    {features[key] ? 'bg-primary' : 'bg-accent'}"
                >
                  <span
                    class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform
                      {features[key] ? 'translate-x-6' : 'translate-x-1'}"
                  ></span>
                </button>
              </div>
            {/each}
          </div>
          <div class="mt-4 flex items-center gap-3">
            <button
              onclick={saveFeatures}
              disabled={featureSaving}
              class="rounded-xl bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50"
            >{featureSaving ? 'Saving…' : 'Save Features'}</button>
          </div>
        {:else}
          <p class="text-sm text-muted-foreground">Click "Load Features" to view and toggle feature flags.</p>
        {/if}
      </div>

      <!-- Coach Config -->
      <div class="mb-6 rounded-2xl border border-border bg-card p-5">
        <div class="flex items-center justify-between mb-4">
          <h2 class="font-display text-lg font-bold text-foreground">Coach Config</h2>
          <div class="flex items-center gap-2">
            {#if coachConfigSaved}
              <span class="text-sm text-green-400 flex items-center gap-1">
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
                Saved!
              </span>
            {/if}
            <button
              onclick={loadCoachConfig}
              disabled={coachConfigLoading}
              class="px-4 py-2 rounded-xl bg-primary text-primary-foreground font-semibold text-sm hover:bg-primary/80 transition disabled:opacity-50"
            >{coachConfigLoading ? 'Loading…' : 'Load Config'}</button>
          </div>
        </div>

        {#if showCoachConfig}
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div>
              <label class="mb-1.5 block text-xs font-medium text-foreground">Max Iterations</label>
              <input
                type="number"
                bind:value={coachConfig.max_iterations}
                min="1"
                max="20"
                class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground focus:border-ring focus:outline-none"
              />
            </div>
            <div>
              <label class="mb-1.5 block text-xs font-medium text-foreground">Context Window Size</label>
              <input
                type="number"
                bind:value={coachConfig.context_window_size}
                min="5"
                max="100"
                class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground focus:border-ring focus:outline-none"
              />
            </div>
            <div>
              <label class="mb-1.5 block text-xs font-medium text-foreground">Max Message Length</label>
              <input
                type="number"
                bind:value={coachConfig.max_message_length}
                min="100"
                max="10000"
                class="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground focus:border-ring focus:outline-none"
              />
            </div>
          </div>
          <div class="mt-4 flex items-center gap-3">
            <button
              onclick={saveCoachConfig}
              disabled={coachConfigSaving}
              class="rounded-xl bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50"
            >{coachConfigSaving ? 'Saving…' : 'Save Config'}</button>
          </div>
        {:else}
          <p class="text-sm text-muted-foreground">Click "Load Config" to view and edit coach agent settings.</p>
        {/if}
      </div>

      <!-- TDEE Config -->
      <div class="mb-6 rounded-2xl border border-border bg-card p-5">
        <div class="flex items-center justify-between mb-4">
          <h2 class="font-display text-lg font-bold text-foreground">TDEE Config</h2>
          <div class="flex items-center gap-2">
            {#if tdeeConfigSaved}
              <span class="text-sm text-green-400 flex items-center gap-1">
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
                Saved!
              </span>
            {/if}
            <button
              onclick={loadTDEEConfig}
              disabled={tdeeConfigLoading}
              class="px-4 py-2 rounded-xl bg-primary text-primary-foreground font-semibold text-sm hover:bg-primary/80 transition disabled:opacity-50"
            >{tdeeConfigLoading ? 'Loading…' : 'Load TDEE Config'}</button>
          </div>
        </div>

        {#if showTDEEConfig}
          <!-- Activity Multipliers -->
          <div class="mb-6">
            <h3 class="text-sm font-semibold text-foreground mb-3">Activity Multipliers</h3>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="border-b border-border">
                    <th class="pb-2 text-left font-medium text-foreground pr-4">Level</th>
                    <th class="pb-2 text-left font-medium text-foreground">Multiplier</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border">
                  {#each activityLevels as [key, label]}
                    <tr class="hover:bg-accent/30">
                      <td class="py-2 pr-4 text-foreground">{label}</td>
                      <td class="py-2">
                        <input
                          type="number"
                          bind:value={tdeeConfig.activity_multipliers[key]}
                          step="0.01"
                          class="w-28 rounded-lg border border-border bg-secondary px-3 py-1.5 text-sm text-foreground focus:border-ring focus:outline-none"
                        />
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>

          <!-- Objective Multipliers -->
          <div class="mb-6">
            <h3 class="text-sm font-semibold text-foreground mb-3">Objective Multipliers</h3>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="border-b border-border">
                    <th class="pb-2 text-left font-medium text-foreground pr-4">Objective</th>
                    <th class="pb-2 text-left font-medium text-foreground">Multiplier</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border">
                  {#each objectives as [key, label]}
                    <tr class="hover:bg-accent/30">
                      <td class="py-2 pr-4 text-foreground">{label}</td>
                      <td class="py-2">
                        <input
                          type="number"
                          bind:value={tdeeConfig.objective_multipliers[key]}
                          step="0.01"
                          class="w-28 rounded-lg border border-border bg-secondary px-3 py-1.5 text-sm text-foreground focus:border-ring focus:outline-none"
                        />
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>

          <!-- Min Calorie Target -->
          <div class="mb-6">
            <label class="mb-1.5 block text-sm font-medium text-foreground">Min Calorie Target</label>
            <input
              type="number"
              bind:value={tdeeConfig.min_calorie_target}
              min="800"
              max="3000"
              class="w-28 rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground focus:border-ring focus:outline-none"
            />
          </div>

          <!-- Macro Splits -->
          <div class="mb-6">
            <h3 class="text-sm font-semibold text-foreground mb-3">Macro Splits (%)</h3>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="border-b border-border">
                    <th class="pb-2 text-left font-medium text-foreground pr-4">Diet Plan</th>
                    {#each macroKeys as mk}
                      <th class="pb-2 text-left font-medium text-foreground capitalize pr-2">{mk}</th>
                    {/each}
                    <th class="pb-2 text-right font-medium text-muted-foreground">Total</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border">
                  {#each dietPlans as [key, label]}
                    <tr class="hover:bg-accent/30">
                      <td class="py-2 pr-4 text-foreground">{label}</td>
                      {#each macroKeys as mk}
                        <td class="py-2 pr-2">
                          <input
                            type="number"
                            bind:value={tdeeConfig.macro_splits[key][mk]}
                            min="0"
                            max="100"
                            class="w-20 rounded-lg border border-border bg-secondary px-2 py-1.5 text-sm text-foreground focus:border-ring focus:outline-none"
                          />
                        </td>
                      {/each}
                      <td class="py-2 text-right text-xs text-muted-foreground">
                        {(tdeeConfig.macro_splits[key].carbs ?? 0) + (tdeeConfig.macro_splits[key].protein ?? 0) + (tdeeConfig.macro_splits[key].fat ?? 0)}%
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>

          <div class="flex items-center gap-3">
            <button
              onclick={saveTDEEConfig}
              disabled={tdeeConfigSaving}
              class="rounded-xl bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 transition disabled:opacity-50"
            >{tdeeConfigSaving ? 'Saving…' : 'Save TDEE Config'}</button>
          </div>
        {:else}
          <p class="text-sm text-muted-foreground">Click "Load TDEE Config" to view and edit TDEE calculation parameters.</p>
        {/if}
      </div>

      <!-- System Health -->
      <div class="mb-6 rounded-2xl border border-border bg-card p-5">
        <button
          onclick={() => { showHealthcheck = !showHealthcheck; if (showHealthcheck && !healthData) loadHealthcheck(); }}
          class="flex items-center justify-between w-full text-left"
        >
          <h2 class="text-lg font-semibold text-foreground">System Health</h2>
          <span class="text-muted-foreground text-xl">{showHealthcheck ? '▾' : '▸'}</span>
        </button>
        {#if showHealthcheck}
          <div class="mt-4 space-y-2">
            {#if healthLoading}
              <p class="text-sm text-muted-foreground">Loading...</p>
            {:else if healthData}
              <div class="flex items-center gap-2">
                <span class="inline-block w-3 h-3 rounded-full {healthData.postgres?.status === 'ok' ? 'bg-green-500' : 'bg-red-500'}"></span>
                <span class="text-sm text-foreground">PostgreSQL</span>
                {#if healthData.postgres?.latency_ms}
                  <span class="text-xs text-muted-foreground">{healthData.postgres.latency_ms}ms</span>
                {/if}
              </div>
              <div class="flex items-center gap-2">
                <span class="inline-block w-3 h-3 rounded-full {healthData.ai?.status === 'configured' ? 'bg-green-500' : healthData.ai?.status === 'no_api_key' ? 'bg-yellow-500' : 'bg-red-500'}"></span>
                <span class="text-sm text-foreground">AI API</span>
                <span class="text-xs text-muted-foreground">{healthData.ai?.status || 'unknown'}</span>
              </div>
              <div class="text-sm text-muted-foreground mt-2">
                Uptime: {Math.floor(healthData.uptime_seconds / 3600)}h {Math.floor((healthData.uptime_seconds % 3600) / 60)}m
              </div>
            {/if}
          </div>
        {/if}
      </div>

      <!-- Food Database -->
      <div class="mb-6 rounded-xl border border-border bg-card p-5">
        <h2 class="text-xl font-semibold mb-4">Food Database</h2>
        <div class="space-y-3">
          <div class="flex items-center gap-4">
            <div class="p-4 rounded-lg border border-border bg-card flex-1">
              <p class="text-2xl font-bold">{foodsCount}</p>
              <p class="text-sm text-muted-foreground">Foods in database</p>
            </div>
            <div class="p-4 rounded-lg border border-border bg-card flex-1">
              <p class="text-sm text-muted-foreground">Last Import</p>
              <p class="text-sm">{importStatus || 'Never'}</p>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <label class="px-4 py-2 text-sm rounded-lg border border-border hover:bg-accent cursor-pointer">
              {importing ? 'Importing...' : 'Import CSV'}
              <input type="file" accept=".csv" onchange={handleFoodImport} class="hidden" />
            </label>
          </div>
          {#if importResult}
            <p class="text-sm text-muted-foreground">
              Imported {importResult.imported} foods, skipped {importResult.skipped} rows.
            </p>
          {/if}
          <p class="text-sm text-muted-foreground">
            CSV format: name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, barcode, brand
          </p>
        </div>
      </div>

      <!-- Nutrition Cache -->
      <div class="mb-6 rounded-xl border border-border bg-card p-5">
        <h2 class="text-xl font-semibold mb-4">Nutrition Cache</h2>
        <div class="space-y-3">
          <div class="flex items-center gap-4">
            <div class="p-4 rounded-lg border border-border bg-card flex-1">
              <p class="text-2xl font-bold">{cacheTotal}</p>
              <p class="text-sm text-muted-foreground">Cached entries</p>
            </div>
            <div class="flex gap-2 flex-1">
              <input
                type="text"
                placeholder="Search cache..."
                bind:value={cacheSearch}
                onkeydown={(e) => { if (e.key === 'Enter') loadNutritionCache(); }}
                class="px-3 py-2 text-sm rounded-lg border border-border bg-card flex-1"
              />
              <button onclick={loadNutritionCache} class="px-4 py-2 text-sm rounded-lg border border-border hover:bg-accent">
                {cacheLoading ? 'Loading...' : 'Search'}
              </button>
              <button onclick={clearCache} class="px-4 py-2 text-sm rounded-lg border border-red-500/50 text-red-500 hover:bg-red-500/10">
                Clear All
              </button>
            </div>
          </div>
          <div class="space-y-2 max-h-96 overflow-y-auto">
            {#each cacheEntries as entry (entry.id)}
              <div class="flex items-center justify-between p-3 rounded-lg border border-border bg-card">
                <div class="flex-1">
                  <div class="flex items-center gap-2">
                    <span class="text-sm font-medium">{entry.name}</span>
                    <span class="text-xs px-2 py-0.5 rounded bg-accent">{entry.source}</span>
                  </div>
                  <p class="text-xs text-muted-foreground">Query: {entry.query} | {entry.calories}kcal P:{entry.protein_g}g C:{entry.carbs_g}g F:{entry.fat_g}g</p>
                </div>
                <button onclick={() => deleteCacheEntry(entry.id)} class="text-muted-foreground hover:text-red-500 text-sm">✕</button>
              </div>
            {/each}
          </div>
        </div>
      </div>

      <!-- Restart Server -->
      <div class="mb-8 rounded-xl border border-red-900/40 bg-card p-5">
        <h2 class="text-sm font-semibold text-red-400 mb-2">Danger Zone</h2>
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-foreground">Restart Server</p>
            <p class="text-xs text-foreground mt-0.5">Docker will automatically restart the container. Users will be disconnected for ~10 seconds.</p>
          </div>
          <button
            onclick={() => { showRestartModal = true; }}
            class="rounded-lg border border-red-700 px-4 py-2 text-sm font-semibold text-red-400 hover:bg-red-500/10 transition"
          >
            Restart
          </button>
        </div>
      </div>

      <!-- Users Table -->
      <div class="rounded-xl border border-border bg-card overflow-hidden">
        <div class="px-5 py-4 border-b border-border">
          <h2 class="text-sm font-semibold text-primary">Users ({users.length})</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-border">
                <th class="px-5 py-3 text-left font-medium text-foreground">Email</th>
                <th class="px-5 py-3 text-left font-medium text-foreground">Verified</th>
                <th class="px-5 py-3 text-left font-medium text-foreground">Approved</th>
                <th class="px-5 py-3 text-left font-medium text-foreground">Role</th>
                <th class="px-5 py-3 text-left font-medium text-foreground">Joined</th>
                <th class="px-5 py-3 text-left font-medium text-foreground">Actions</th>
              </tr>
            </thead>
            <tbody>
              {#each users as user}
                <tr class="border-b border-border last:border-0 hover:bg-accent/30">
                  <td class="px-5 py-3 text-foreground">
                    {user.email}
                    {#if user.id === currentUserID}
                      <span class="ml-1.5 text-xs text-muted-foreground">(you)</span>
                    {/if}
                  </td>
                  <td class="px-5 py-3">
                    {#if user.verified}
                      <svg class="h-4 w-4 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
                    {:else}
                      <svg class="h-4 w-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
                    {/if}
                  </td>
                  <td class="px-5 py-3">
                    {#if user.approved}
                      <span class="inline-flex items-center rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-400">Approved</span>
                    {:else}
                      <span class="inline-flex items-center rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-400">Pending</span>
                    {/if}
                  </td>
                  <td class="px-5 py-3">
                    {#if user.is_admin}
                      <span class="inline-flex items-center rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">Admin</span>
                    {:else}
                      <span class="text-muted-foreground">User</span>
                    {/if}
                  </td>
                  <td class="px-5 py-3 text-foreground">{formatDate(user.created_at)}</td>
                  <td class="px-5 py-3">
                    {#if user.id !== currentUserID}
                      <div class="flex flex-wrap items-center gap-1.5">
                        <a
                          href="/admin/users/{user.id}"
                          class="rounded px-2 py-1 text-xs font-medium text-foreground hover:bg-accent transition"
                        >View</a>
                        {#if !user.is_admin}
                          {#if !user.verified}
                            <button
                              onclick={() => verifyUser(user.id, user.email)}
                              disabled={actionLoading === user.id + ':verify'}
                              class="rounded px-2 py-1 text-xs font-medium text-blue-400 hover:bg-blue-500/10 transition disabled:opacity-50"
                            >Verify Email</button>
                          {/if}
                          {#if user.approved}
                            <button
                              onclick={() => unapproveUser(user.id)}
                              disabled={actionLoading === user.id + ':unapprove'}
                              class="rounded px-2 py-1 text-xs font-medium text-amber-400 hover:bg-amber-500/10 transition disabled:opacity-50"
                            >Unapprove</button>
                          {:else}
                            <button
                              onclick={() => approveUser(user.id)}
                              disabled={actionLoading === user.id + ':approve'}
                              class="rounded px-2 py-1 text-xs font-medium text-green-400 hover:bg-green-500/10 transition disabled:opacity-50"
                            >Approve</button>
                          {/if}
                          <button
                            onclick={() => makeAdmin(user.id, user.email)}
                            disabled={actionLoading === user.id + ':makeadmin'}
                            class="rounded px-2 py-1 text-xs font-medium text-primary hover:bg-primary/10 transition disabled:opacity-50"
                          >Make Admin</button>
                          <button
                            onclick={() => deleteUser(user.id, user.email)}
                            disabled={actionLoading === user.id + ':delete'}
                            class="rounded px-2 py-1 text-xs font-medium text-red-400 hover:bg-red-500/10 transition disabled:opacity-50"
                          >Delete</button>
                        {:else}
                          <button
                            onclick={() => removeAdmin(user.id, user.email)}
                            disabled={actionLoading === user.id + ':removeadmin'}
                            class="rounded px-2 py-1 text-xs font-medium text-foreground hover:bg-accent/50 transition disabled:opacity-50"
                          >Remove Admin</button>
                        {/if}
                      </div>
                    {:else}
                      <span class="text-xs text-muted-foreground">—</span>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>

      <!-- Banners -->
      <div class="mt-8 rounded-xl border border-border bg-card p-5">
        <h2 class="mb-4 text-sm font-semibold text-primary">Announcement Banners</h2>
        <p class="mb-4 text-xs text-foreground">Banners appear on the dashboard for all users. They can dismiss individually.</p>

        <!-- Create banner form -->
        <div class="mb-4 space-y-3 rounded-lg border border-border bg-secondary p-4">
          <input
            type="text"
            bind:value={newBannerTitle}
            placeholder="Title (optional)"
            class="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
          />
          <textarea
            bind:value={newBannerMsg}
            placeholder="Message *"
            rows="2"
            class="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring resize-none"
          ></textarea>
          <div class="flex flex-wrap items-center gap-3">
            <select
              bind:value={newBannerType}
              class="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none"
            >
              <option value="info">Info</option>
              <option value="tip">Tip</option>
              <option value="warning">Warning</option>
            </select>
            <select
              bind:value={newBannerExpiry}
              class="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none"
            >
              <option value="">No expiry</option>
              <option value="1">1 hour</option>
              <option value="6">6 hours</option>
              <option value="24">1 day</option>
              <option value="72">3 days</option>
              <option value="168">7 days</option>
            </select>
            <button
              onclick={createBanner}
              disabled={!newBannerMsg.trim() || bannerSaving}
              class="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/80 disabled:opacity-50 transition"
            >{bannerSaving ? 'Posting…' : 'Post Banner'}</button>
          </div>
        </div>

        <!-- Existing banners -->
        {#if banners.length === 0}
          <p class="text-sm text-muted-foreground">No active banners.</p>
        {:else}
          <div class="space-y-2">
            {#each banners as banner}
              <div class="flex items-start justify-between gap-3 rounded-lg border border-border bg-secondary px-4 py-3">
                <div class="min-w-0">
                  {#if banner.title}<p class="text-sm font-medium text-foreground">{banner.title}</p>{/if}
                  <p class="text-sm text-foreground">{banner.message}</p>
                  <p class="mt-0.5 text-xs text-muted-foreground capitalize">
                    {banner.type} · {formatDate(banner.created_at)}
                    {#if banner.expires_at} · expires {formatDate(banner.expires_at)}{/if}
                  </p>
                </div>
                <button
                  onclick={() => deleteBanner(banner.id)}
                  class="shrink-0 text-muted-foreground hover:text-red-400 transition"
                  aria-label="Delete banner"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <!-- System Logs -->
      <div class="mt-8 rounded-xl border border-border bg-card p-5">
        <div class="mb-4 flex items-center justify-between gap-3">
          <h2 class="text-sm font-semibold text-primary">System Logs</h2>
          <div class="flex items-center gap-2">
            <select
              bind:value={logCategory}
              class="rounded-lg border border-border bg-secondary px-3 py-1.5 text-sm text-foreground focus:border-primary focus:outline-none"
            >
              <option value="all">All</option>
              <option value="smtp">SMTP</option>
              <option value="ai">AI</option>
              <option value="auth">Auth</option>
              <option value="general">General</option>
            </select>
            <button
              onclick={loadLogs}
              disabled={logsLoading}
              class="rounded-lg border border-border px-3 py-1.5 text-sm font-medium text-foreground hover:text-foreground hover:bg-accent transition disabled:opacity-50"
            >{logsLoading ? 'Loading…' : 'Load Logs'}</button>
          </div>
        </div>

        {#if logs.length === 0}
          <p class="text-sm text-muted-foreground">Click "Load Logs" to view recent system events.</p>
        {:else}
          <div class="overflow-x-auto">
            <table class="w-full text-xs">
              <thead>
                <tr class="border-b border-border">
                  <th class="pb-2 text-left font-medium text-foreground pr-4">Time</th>
                  <th class="pb-2 text-left font-medium text-foreground pr-4">Level</th>
                  <th class="pb-2 text-left font-medium text-foreground pr-4">Category</th>
                  <th class="pb-2 text-left font-medium text-foreground">Message</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                {#each logs as log}
                  <tr class="hover:bg-accent/30">
                    <td class="py-2 pr-4 text-muted-foreground whitespace-nowrap">{new Date(log.created_at).toLocaleTimeString()}</td>
                    <td class="py-2 pr-4">
                      <span class="inline-flex rounded px-1.5 py-0.5 text-xs font-medium capitalize
                        {log.level === 'error' ? 'bg-red-500/10 text-red-400' :
                         log.level === 'warn' ? 'bg-amber-500/10 text-amber-400' :
                         'bg-blue-500/10 text-blue-400'}">{log.level}</span>
                    </td>
                    <td class="py-2 pr-4 text-foreground capitalize">{log.category}</td>
                    <td class="py-2 text-foreground">{log.message}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    {/if}
  </main>
</div>

<!-- Restart Confirmation Modal -->
{#if showRestartModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
    <div class="mx-4 w-full max-w-md rounded-2xl border border-border bg-secondary p-6 shadow-2xl">
      <div class="mb-4 flex items-center gap-3">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-red-500/10">
          <svg class="h-5 w-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        </div>
        <h3 class="text-lg font-semibold text-foreground">Restart Server?</h3>
      </div>
      <p class="mb-6 text-sm text-foreground">
        This will restart the server container. All active users will be disconnected for approximately 10 seconds while Docker restarts the service.
      </p>
      <div class="flex gap-3">
        <button
          onclick={() => { showRestartModal = false; }}
          disabled={restarting}
          class="flex-1 rounded-xl border border-border px-4 py-2.5 text-sm font-semibold text-foreground hover:text-foreground transition disabled:opacity-50"
        >
          Cancel
        </button>
        <button
          onclick={confirmRestart}
          disabled={restarting}
          class="flex-1 rounded-xl bg-red-600 px-4 py-2.5 text-sm font-semibold text-foreground hover:bg-red-500 transition disabled:opacity-50"
        >
          {restarting ? 'Restarting…' : 'Restart Server'}
        </button>
      </div>
    </div>
  </div>
{/if}
