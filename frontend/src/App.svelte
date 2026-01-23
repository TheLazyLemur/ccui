<script lang="ts">
  import { onMount, afterUpdate } from 'svelte';
  import { EventsOn, EventsEmit } from '../wailsjs/runtime/runtime';
  import { marked } from 'marked';
  import ToolCard from './lib/ToolCard.svelte';
  import { type Message, type ToolCall, type UserQuestion, getStatusIndicator, getStatusClass } from './lib/shared';

  let messages: Message[] = [];
  let inputText = '';
  let messagesContainer: HTMLDivElement;
  let textarea: HTMLTextAreaElement;
  let messageId = 0;
  let isLoading = false;
  let currentChunk = '';
  let currentThought = '';
  let userQuestion: UserQuestion | null = null;
  let userAnswerInput = '';
  let expandedSubagents: Set<string> = new Set();

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
    if (!(e.metaKey || e.ctrlKey)) return;
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

  onMount(() => {
    loadSettings();
    window.addEventListener('keydown', handleGlobalKeydown);

    EventsOn('chat_chunk', (t: string) => currentChunk += t);
    EventsOn('chat_thought', (t: string) => currentThought += t);

    EventsOn('tool_state', (state: ToolCall) => {
      const idx = messages.findIndex(m => m.toolState?.id === state.id);
      if (idx >= 0) { messages[idx].toolState = state; messages = messages; }
      else messages = [...messages, { id: ++messageId, text: '', sender: 'tool', toolState: state }];
    });

    EventsOn('prompt_complete', () => {
      if (currentChunk) { messages = [...messages, { id: ++messageId, text: currentChunk, sender: 'bot' }]; currentChunk = ''; }
      currentThought = '';
      isLoading = false;
    });

    EventsOn('error', (err: string) => {
      messages = [...messages, { id: ++messageId, text: `Error: ${err}`, sender: 'bot' }];
      isLoading = false;
    });

    EventsOn('user_question', (q: UserQuestion) => { userQuestion = q; userAnswerInput = ''; });
  });

  afterUpdate(() => { if (messagesContainer) messagesContainer.scrollTop = messagesContainer.scrollHeight; });

  function sendMessage() {
    const text = inputText.trim();
    if (!text || isLoading) return;
    if (currentChunk) { messages = [...messages, { id: ++messageId, text: currentChunk, sender: 'bot' }]; currentChunk = ''; }
    messages = [...messages, { id: ++messageId, text, sender: 'user' }];
    EventsEmit('send_message', text);
    inputText = '';
    isLoading = true;
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

</script>

<div class="h-full bg-paper paper-texture" style="zoom: {fontScale}">
  <div class="h-full flex flex-col max-w-4xl mx-auto">
    <!-- Header -->
    <header class="px-6 py-4 flex justify-between items-center border-b border-ink-faint relative z-10">
      <h1 class="text-lg font-medium tracking-tight text-ink">ccui</h1>
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

    <!-- Messages -->
    <div bind:this={messagesContainer} class="flex-1 overflow-y-auto px-6 py-6 space-y-5 relative z-10">
      {#each messages as msg, i (msg.id)}
        {#if msg.sender === 'tool' && msg.toolState && !msg.toolState.parentId}
          {@const tool = msg.toolState}
          {@const isSubagent = tool.toolName === 'Task'}
          {@const children = isSubagent ? getChildTools(tool.id) : []}
          {@const isExpanded = expandedSubagents.has(tool.id)}

          <div class="animate-slide-up" style="animation-delay: {i * 30}ms">
            {#if isSubagent}
              <div class="border border-ink-faint">
                <button on:click={() => toggleSubagent(tool.id)} class="w-full px-4 py-3 flex items-center gap-3 text-left hover:bg-paper-dim transition-colors">
                  <span class="text-ink-muted text-xs transition-transform {isExpanded ? 'rotate-90' : ''}">▸</span>
                  <span class="{getStatusClass(tool.status)} text-xs {tool.status === 'running' ? 'animate-pulse-subtle' : ''}">{getStatusIndicator(tool.status)}</span>
                  <span class="text-ink text-sm flex-1 truncate">{tool.title || 'Task'}</span>
                  <span class="text-ink-muted text-xs font-mono">{children.length}</span>
                </button>
                {#if isExpanded}
                  <div class="border-t border-ink-faint bg-paper-dim/50 px-4 py-3 space-y-2">
                    {#each children as childTool}
                      <ToolCard tool={childTool} compact on:permission={respondPermission} />
                    {/each}
                  </div>
                {/if}
              </div>
            {:else}
              <ToolCard {tool} on:permission={respondPermission} />
            {/if}
          </div>
        {:else if msg.sender !== 'tool'}
          <div class="animate-slide-up" style="animation-delay: {i * 30}ms">
            {#if msg.sender === 'user'}
              <div class="flex justify-end">
                <div class="max-w-[80%]">
                  <div class="text-right mb-1"><span class="text-xs text-ink-muted uppercase tracking-wide">You</span></div>
                  <div class="bg-paper-dim border border-ink-faint px-5 py-4">
                    <p class="text-[15px] leading-relaxed text-ink whitespace-pre-wrap break-words">{msg.text}</p>
                  </div>
                </div>
              </div>
            {:else}
              <div class="max-w-[85%]">
                <div class="mb-1"><span class="text-xs text-ink-muted uppercase tracking-wide">Assistant</span></div>
                <div class="border-l-2 border-ink-faint pl-5 py-1 prose prose-sm prose-eink">{@html marked(msg.text)}</div>
              </div>
            {/if}
          </div>
        {/if}
      {/each}

      {#if currentChunk}
        <div class="animate-fade-in max-w-[85%]">
          <div class="mb-1"><span class="text-xs text-ink-muted uppercase tracking-wide">Assistant</span></div>
          <div class="border-l-2 border-ink-medium pl-5 py-1 prose prose-sm prose-eink">{@html marked(currentChunk)}<span class="inline-block w-px h-4 bg-ink ml-0.5 animate-pulse-subtle align-baseline"></span></div>
        </div>
      {/if}

      {#if currentThought}
        <div class="animate-fade-in max-w-[70%] px-5 py-4 border border-dashed border-ink-faint">
          <div class="flex items-center gap-2 text-xs text-ink-muted mb-2">
            <span class="uppercase tracking-wide">Thinking</span><span class="animate-pulse-subtle">...</span>
          </div>
          <p class="text-sm text-ink-light font-mono leading-relaxed">{currentThought.slice(-200)}{currentThought.length > 200 ? '...' : ''}</p>
        </div>
      {/if}

      {#if messages.length === 0 && !currentChunk}
        <div class="absolute inset-0 flex flex-col items-center justify-center px-8">
          <div class="text-6xl text-ink-faint mb-6">※</div>
          <p class="text-ink-light text-center max-w-sm leading-relaxed">Begin a conversation below</p>
        </div>
      {/if}
    </div>

    <!-- User Question Modal -->
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

    <!-- Input -->
    <div class="px-6 py-4 border-t border-ink-faint relative z-10">
      <div class="flex gap-3 items-end">
        <textarea bind:this={textarea} bind:value={inputText} on:keydown={handleKeydown} on:input={autoResize} placeholder="Write a message..." rows="1" disabled={isLoading} class="flex-1 px-4 py-3 bg-paper border border-ink-faint text-ink text-[15px] placeholder-ink-muted resize-none leading-normal focus:outline-none focus:border-ink-muted transition-colors disabled:opacity-50"></textarea>
        <button on:click={sendMessage} disabled={isLoading} class="px-5 py-3 bg-ink text-paper text-sm hover:bg-ink-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
          {#if isLoading}<span class="inline-block animate-spin-slow">◎</span>{:else}Send{/if}
        </button>
      </div>
    </div>
  </div>
</div>
