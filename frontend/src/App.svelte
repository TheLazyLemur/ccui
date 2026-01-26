<script lang="ts">
  import { onMount } from 'svelte';
  import { EventsOn, EventsEmit } from '../wailsjs/runtime/runtime';
  import ReviewPanel from './lib/ReviewPanel.svelte';
  import ModeSelector from './lib/ModeSelector.svelte';
  import SessionSelector from './lib/SessionSelector.svelte';
  import Terminal from './lib/Terminal.svelte';
  import SplitPane from './lib/SplitPane.svelte';
  import CommandPalette from './lib/CommandPalette.svelte';
  import ChatContent from './lib/ChatContent.svelte';
  import { type Message, type ToolCall, type UserQuestion, type FileChange, type ReviewComment, type SessionMode, type PlanEntry, type SessionInfo, type SessionState } from './lib/shared';
  import { GetSessions, GetActiveSession, CreateSession } from '../wailsjs/go/main/App';

  // Multi-session state
  let sessions: SessionInfo[] = [];
  let activeSessionId = '';
  let sessionStates = new Map<string, SessionState>();
  let subscribedSessions = new Set<string>();
  let unsubscribeFns: (() => void)[] = [];

  // Current session's reactive state (bound to active session)
  let messages: Message[] = [];
  let inputText = '';
  let textarea: HTMLTextAreaElement;
  let isLoading = false;
  let currentChunk = '';
  let currentThought = '';
  let userQuestion: UserQuestion | null = null;
  let userAnswerInput = '';
  let expandedSubagents: Set<string> = new Set();

  // Panel state (replaces tab state)
  type PanelType = 'chat' | 'review' | 'terminal' | null;
  let leftPanel: PanelType = 'chat';
  let rightPanel: PanelType = 'terminal';
  let splitSize = 50;
  let showPalette = false;
  let paletteTarget: 'left' | 'right' = 'left';
  let focusedPanel: 'left' | 'right' = 'left';

  let fileChanges: FileChange[] = [];
  let reviewComments: ReviewComment[] = [];
  let reviewAgentOutput = '';
  let reviewAgentRunning = false;

  // Session modes & plan
  let availableModes: SessionMode[] = [];
  let currentModeId = '';
  let planEntries: PlanEntry[] = [];

  function getOrCreateSessionState(id: string): SessionState {
    if (!sessionStates.has(id)) {
      sessionStates.set(id, {
        messages: [],
        fileChanges: [],
        reviewComments: [],
        planEntries: [],
        currentModeId: '',
        currentChunk: '',
        currentThought: '',
        availableModes: [],
        isLoading: false,
        reviewAgentOutput: '',
        reviewAgentRunning: false
      });
    }
    return sessionStates.get(id)!;
  }

  function saveCurrentSessionState() {
    if (!activeSessionId) return;
    const state = getOrCreateSessionState(activeSessionId);
    Object.assign(state, { messages, fileChanges, reviewComments, planEntries, currentModeId, currentChunk, currentThought, availableModes, isLoading, reviewAgentOutput, reviewAgentRunning });
  }

  function loadSessionState(id: string) {
    const state = getOrCreateSessionState(id);
    ({ messages, fileChanges, reviewComments, planEntries, currentModeId, currentChunk, currentThought, availableModes, isLoading, reviewAgentOutput, reviewAgentRunning } = state);
  }

  function subscribeToSession(sessionId: string) {
    if (subscribedSessions.has(sessionId)) return;
    subscribedSessions.add(sessionId);

    const state = getOrCreateSessionState(sessionId);
    const on = (e: string, cb: (...args: any[]) => void) => unsubscribeFns.push(EventsOn(`session:${sessionId}:${e}`, cb));

    // Helper to sync reactive vars if this is active session
    const syncIfActive = () => {
      if (sessionId === activeSessionId) loadSessionState(sessionId);
    };

    on('chat_chunk', (t: string) => {
      state.currentChunk += t;
      syncIfActive();
    });
    on('chat_thought', (t: string) => {
      state.currentThought += t;
      syncIfActive();
    });
    on('tool_state', (toolState: ToolCall) => {
      const idx = state.messages.findIndex(m => m.toolState?.id === toolState.id);
      if (idx >= 0) { state.messages[idx].toolState = toolState; }
      else {
        const newId = state.messages.length > 0 ? Math.max(...state.messages.map(m => m.id)) + 1 : 1;
        state.messages.push({ id: newId, text: '', sender: 'tool', toolState });
      }
      syncIfActive();
    });
    on('prompt_complete', () => {
      if (state.currentChunk) {
        const newId = state.messages.length > 0 ? Math.max(...state.messages.map(m => m.id)) + 1 : 1;
        state.messages.push({ id: newId, text: state.currentChunk, sender: 'bot' });
        state.currentChunk = '';
      }
      state.currentThought = '';
      state.isLoading = false;
      syncIfActive();
    });
    on('error', (err: string) => {
      const newId = state.messages.length > 0 ? Math.max(...state.messages.map(m => m.id)) + 1 : 1;
      state.messages.push({ id: newId, text: `Error: ${err}`, sender: 'bot' });
      state.isLoading = false;
      syncIfActive();
    });
    on('file_changes_updated', (changes: FileChange[]) => { state.fileChanges = changes; syncIfActive(); });
    on('review_agent_chunk', (t: string) => { state.reviewAgentOutput += t; syncIfActive(); });
    on('review_agent_running', () => { state.reviewAgentRunning = true; state.reviewAgentOutput = ''; syncIfActive(); });
    on('review_agent_complete', () => { state.reviewAgentRunning = false; state.reviewComments = []; syncIfActive(); });
    on('modes_available', (modes: SessionMode[]) => { state.availableModes = modes; syncIfActive(); });
    on('mode_changed', (modeId: string) => { state.currentModeId = modeId; syncIfActive(); });
    on('plan_update', (entries: PlanEntry[]) => { state.planEntries = entries; syncIfActive(); });
  }

  function handleSessionChange(newSessionId: string) {
    if (newSessionId === activeSessionId) return;
    saveCurrentSessionState();
    activeSessionId = newSessionId;
    loadSessionState(newSessionId);
    subscribeToSession(newSessionId);
  }

  // Theme & font
  let isDark = false;
  let fontScale = 1;
  const THEME_KEY = 'ccui-theme';
  const FONT_SCALE_KEY = 'ccui-font-scale';

  function loadSettings() {
    if (localStorage.getItem(THEME_KEY) === 'dark') {
      isDark = true;
      document.documentElement.classList.add('dark');
    }
    const fs = localStorage.getItem(FONT_SCALE_KEY);
    if (fs) fontScale = parseFloat(fs);
  }

  function toggleTheme() {
    isDark = !isDark;
    document.documentElement.classList.toggle('dark', isDark);
    localStorage.setItem(THEME_KEY, isDark ? 'dark' : 'light');
  }

  function handleGlobalKeydown(e: KeyboardEvent) {
    // Escape closes palette
    if (e.key === 'Escape' && showPalette) {
      e.preventDefault();
      showPalette = false;
      return;
    }

    if (!(e.metaKey || e.ctrlKey)) return;

    // Cmd+K: Open command palette for focused panel
    if (e.key === 'k') {
      e.preventDefault();
      paletteTarget = focusedPanel;
      showPalette = true;
      return;
    }

    // Cmd+\: Toggle split view
    if (e.key === '\\') {
      e.preventDefault();
      rightPanel = rightPanel === null ? 'terminal' : null;
      return;
    }

    // Cmd+1: Focus left panel
    if (e.key === '1') {
      e.preventDefault();
      focusedPanel = 'left';
      return;
    }

    // Cmd+2: Focus right panel
    if (e.key === '2') {
      e.preventDefault();
      focusedPanel = 'right';
      return;
    }

    // Font scaling
    if (e.key === '=' || e.key === '+') { e.preventDefault(); fontScale = Math.min(fontScale + 0.1, 2); }
    else if (e.key === '-') { e.preventDefault(); fontScale = Math.max(fontScale - 0.1, 0.5); }
    else if (e.key === '0') { e.preventDefault(); fontScale = 1; }
    else return;
    localStorage.setItem(FONT_SCALE_KEY, fontScale.toString());
  }

  function getChildTools(parentId: string): ToolCall[] {
    return messages.filter(m => m.toolState?.parentId === parentId).map(m => m.toolState!);
  }

  function toggleSubagent(id: string) {
    expandedSubagents.has(id) ? expandedSubagents.delete(id) : expandedSubagents.add(id);
    expandedSubagents = expandedSubagents;
  }

  function autoResize() {
    if (!textarea) return;
    textarea.style.height = 'auto';
    textarea.style.height = Math.min(textarea.scrollHeight, 200) + 'px';
  }

  onMount(async () => {
    loadSettings();
    window.addEventListener('keydown', handleGlobalKeydown);

    // Global session events (not prefixed)
    EventsOn('sessions_updated', (s: SessionInfo[]) => { sessions = s; });
    EventsOn('active_session_changed', (id: string) => {
      if (id && id !== activeSessionId) {
        handleSessionChange(id);
      }
    });

    // User question is global (handled by MCP server)
    EventsOn('user_question', (q: UserQuestion) => { userQuestion = q; userAnswerInput = ''; });

    // Initialize: fetch existing sessions or create first one
    const existingSessions = await GetSessions();
    if (existingSessions.length > 0) {
      sessions = existingSessions;
      const active = await GetActiveSession();
      if (active) handleSessionChange(active);
      else handleSessionChange(existingSessions[0].id);
    } else {
      await CreateSession('Session 1');
    }

    return () => {
      unsubscribeFns.forEach(fn => fn());
    };
  });

  function sendMessage() {
    const text = inputText.trim();
    if (!text || isLoading || !activeSessionId) return;
    const state = getOrCreateSessionState(activeSessionId);
    if (state.currentChunk) {
      const newId = state.messages.length > 0 ? Math.max(...state.messages.map(m => m.id)) + 1 : 1;
      state.messages.push({ id: newId, text: state.currentChunk, sender: 'bot' });
      state.currentChunk = '';
    }
    const msgId = state.messages.length > 0 ? Math.max(...state.messages.map(m => m.id)) + 1 : 1;
    state.messages.push({ id: msgId, text, sender: 'user' });
    state.isLoading = true;
    loadSessionState(activeSessionId);
    EventsEmit('send_message', text);
    inputText = '';
    if (textarea) textarea.style.height = 'auto';
  }

  function handleKeydown(e: KeyboardEvent) { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage(); } }
  function respondPermission(e: CustomEvent<string>) { EventsEmit('permission_response', e.detail); }
  function cancelRequest() { EventsEmit('cancel'); isLoading = false; }

  function submitUserAnswer(answer?: string) {
    if (!userQuestion) return;
    const a = answer || userAnswerInput.trim();
    if (!a) return;
    EventsEmit('user_answer', { requestId: userQuestion.requestId, answer: a });
    userQuestion = null;
    userAnswerInput = '';
  }

  const handleAddComment = (e: CustomEvent<ReviewComment>) => reviewComments = [...reviewComments, e.detail];
  const handleRemoveComment = (e: CustomEvent<string>) => reviewComments = reviewComments.filter(c => c.id !== e.detail);
  const handleSubmitReview = () => EventsEmit('submit_review', reviewComments);

  function handlePaletteSelect(e: CustomEvent<PanelType>) {
    if (paletteTarget === 'left') {
      leftPanel = e.detail;
    } else {
      rightPanel = e.detail;
    }
    showPalette = false;
  }

  function handlePaletteClose() {
    showPalette = false;
  }

  function openPaletteForPanel(panel: 'left' | 'right') {
    paletteTarget = panel;
    focusedPanel = panel;
    showPalette = true;
  }

  function handleSplitResize(e: CustomEvent<number>) {
    splitSize = e.detail;
  }

  // Check if any panel has chat (for showing input)
  $: hasChatPanel = leftPanel === 'chat' || rightPanel === 'chat';
</script>

<div class="h-full bg-paper paper-texture" style="zoom: {fontScale}">
  <div class="h-full flex flex-col">
    <!-- Header -->
    <header class="px-6 py-4 flex justify-between items-center border-b border-ink-faint relative z-20">
      <div class="flex items-center gap-4">
        <h1 class="text-lg font-medium tracking-tight text-ink">ccui</h1>
        <SessionSelector {sessions} {activeSessionId} on:sessionChange={(e) => handleSessionChange(e.detail)} />
        <ModeSelector modes={availableModes} {currentModeId} />
      </div>
      <div class="flex items-center gap-4">
        {#if isLoading}
          <button on:click={cancelRequest} class="px-3 py-1.5 text-sm border border-ink-faint text-ink-medium hover:border-ink-muted hover:text-ink transition-colors">Cancel</button>
        {/if}
        <button type="button" on:click={toggleTheme} class="w-8 h-8 flex items-center justify-center border border-ink-faint text-ink-medium hover:border-ink-muted hover:text-ink active:scale-95 cursor-pointer" title={isDark ? 'Light mode' : 'Dark mode'}>
          {#if isDark}<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><circle cx="12" cy="12" r="5" stroke-width="1.5"/><path stroke-width="1.5" d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
          {:else}<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-width="1.5" d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>{/if}
        </button>
      </div>
    </header>

    <!-- Split Pane Content Area -->
    <div class="flex-1 overflow-hidden relative">
      <SplitPane leftSize={splitSize} on:resize={handleSplitResize}>
        <div slot="left" class="h-full flex flex-col {focusedPanel === 'left' ? 'ring-1 ring-ink-faint ring-inset' : ''}">
          <!-- Left Panel Header -->
          <button
            on:click={() => openPaletteForPanel('left')}
            class="px-4 py-2 text-xs text-ink-muted uppercase tracking-wide border-b border-ink-faint hover:bg-paper-dim transition-colors flex items-center justify-between"
          >
            <span>{leftPanel || 'Empty'}</span>
            <span class="text-ink-faint">Cmd+K</span>
          </button>
          <!-- Left Panel Content -->
          <div class="flex-1 overflow-hidden">
            {#if leftPanel === 'chat'}
              <ChatContent
                {messages}
                {currentChunk}
                {currentThought}
                {planEntries}
                {expandedSubagents}
                {getChildTools}
                {toggleSubagent}
                on:permission={respondPermission}
              />
            {:else if leftPanel === 'review'}
              <ReviewPanel
                {fileChanges}
                comments={reviewComments}
                agentOutput={reviewAgentOutput}
                agentRunning={reviewAgentRunning}
                on:addComment={handleAddComment}
                on:removeComment={handleRemoveComment}
                on:submitReview={handleSubmitReview}
              />
            {:else if leftPanel === 'terminal'}
              <Terminal terminalId={activeSessionId || 'default'} />
            {:else}
              <div class="h-full flex flex-col items-center justify-center text-ink-muted">
                <p class="text-sm">Press <kbd class="px-1.5 py-0.5 border border-ink-faint text-xs">Cmd+K</kbd> to open panel</p>
              </div>
            {/if}
          </div>
        </div>

        <div slot="right" class="h-full flex flex-col {focusedPanel === 'right' ? 'ring-1 ring-ink-faint ring-inset' : ''}">
          {#if rightPanel !== null}
            <!-- Right Panel Header -->
            <button
              on:click={() => openPaletteForPanel('right')}
              class="px-4 py-2 text-xs text-ink-muted uppercase tracking-wide border-b border-ink-faint hover:bg-paper-dim transition-colors flex items-center justify-between"
            >
              <span>{rightPanel || 'Empty'}</span>
              <span class="text-ink-faint">Cmd+K</span>
            </button>
            <!-- Right Panel Content -->
            <div class="flex-1 overflow-hidden">
              {#if rightPanel === 'chat'}
                <ChatContent
                  {messages}
                  {currentChunk}
                  {currentThought}
                  {planEntries}
                  {expandedSubagents}
                  {getChildTools}
                  {toggleSubagent}
                  on:permission={respondPermission}
                />
              {:else if rightPanel === 'review'}
                <ReviewPanel
                  {fileChanges}
                  comments={reviewComments}
                  agentOutput={reviewAgentOutput}
                  agentRunning={reviewAgentRunning}
                  on:addComment={handleAddComment}
                  on:removeComment={handleRemoveComment}
                  on:submitReview={handleSubmitReview}
                />
              {:else if rightPanel === 'terminal'}
                <Terminal terminalId={activeSessionId || 'default'} />
              {:else}
                <div class="h-full flex flex-col items-center justify-center text-ink-muted">
                  <p class="text-sm">Press <kbd class="px-1.5 py-0.5 border border-ink-faint text-xs">Cmd+K</kbd> to open panel</p>
                </div>
              {/if}
            </div>
          {/if}
        </div>
      </SplitPane>
    </div>

    <!-- User Question Modal (overlay) -->
    {#if userQuestion}
      <div class="absolute inset-0 bg-paper/90 flex items-center justify-center z-50">
        <div class="bg-paper border border-ink-faint max-w-lg w-full mx-6 animate-slide-up">
          <div class="px-6 py-4 border-b border-ink-faint"><h2 class="text-ink font-medium">Question</h2></div>
          <div class="px-6 py-5">
            <p class="text-ink-medium text-[15px] leading-relaxed mb-5">{userQuestion.question}</p>
            {#if userQuestion.options?.length}
              <div class="space-y-2 mb-5">
                {#each userQuestion.options as opt}
                  <button on:click={() => submitUserAnswer(opt.label)} class="w-full px-4 py-3 text-left border border-ink-faint hover:border-ink-muted hover:bg-paper-dim transition-colors">
                    <span class="text-ink text-sm">{opt.label}</span>
                    {#if opt.description}<p class="text-ink-muted text-xs mt-1">{opt.description}</p>{/if}
                  </button>
                {/each}
              </div>
            {/if}
            <textarea bind:value={userAnswerInput} placeholder="Type your response..." rows="3" class="w-full px-4 py-3 bg-paper border border-ink-faint text-ink text-[15px] placeholder-ink-muted resize-none focus:outline-none focus:border-ink-muted transition-colors"></textarea>
          </div>
          <div class="px-6 py-4 border-t border-ink-faint flex justify-end">
            <button on:click={() => submitUserAnswer()} disabled={!userAnswerInput.trim()} class="px-5 py-2 bg-ink text-paper text-sm hover:bg-ink-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed">Submit</button>
          </div>
        </div>
      </div>
    {/if}

    <!-- Command Palette (overlay) -->
    {#if showPalette}
      <CommandPalette on:select={handlePaletteSelect} on:close={handlePaletteClose} />
    {/if}

    <!-- Input (shared, visible when chat panel exists) -->
    {#if hasChatPanel}
      <div class="px-6 py-4 border-t border-ink-faint relative z-10">
        <div class="flex gap-3 items-end">
          <textarea bind:this={textarea} bind:value={inputText} on:keydown={handleKeydown} on:input={autoResize} placeholder="Write a message..." rows="1" disabled={isLoading} class="flex-1 px-4 py-3 bg-paper border border-ink-faint text-ink text-[15px] placeholder-ink-muted resize-none leading-normal focus:outline-none focus:border-ink-muted transition-colors disabled:opacity-50"></textarea>
          <button on:click={sendMessage} disabled={isLoading} class="px-5 py-3 bg-ink text-paper text-sm hover:bg-ink-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
            {#if isLoading}<span class="inline-block animate-spin-slow">◎</span>{:else}Send{/if}
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>
