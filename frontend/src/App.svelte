<script lang="ts">
  import { onMount, afterUpdate } from 'svelte';
  import { EventsOn, EventsEmit } from '../wailsjs/runtime/runtime';
  import { marked } from 'marked';

  interface Message {
    id: number;
    text: string;
    sender: 'user' | 'bot' | 'tool';
    diff?: DiffInfo;
    toolTitle?: string;
    toolState?: ToolCall;
  }

  interface DiffBlock {
    type: string;
    path?: string;
    oldText?: string;
    newText?: string;
  }

  interface PatchHunk {
    oldStart: number;
    oldLines: number;
    newStart: number;
    newLines: number;
    lines: string[];
  }

  interface DiffInfo {
    filePath?: string;
    oldString?: string;
    newString?: string;
    originalFile?: string;
    structuredPatch?: PatchHunk[];
    type?: string;
    content?: string;
  }

  interface ToolCall {
    id: string;
    title: string;
    kind: string;
    status: string; // pending, awaiting_permission, running, completed, error
    toolName?: string;
    parentId?: string;
    input?: Record<string, unknown>;
    output?: unknown[];
    diffs?: DiffBlock[];
    diff?: DiffInfo;
    permissionOptions?: PermissionOption[];
  }

  interface PermissionOption {
    optionId: string;
    name: string;
    kind: string;
  }

  let messages: Message[] = [];
  let inputText = '';
  let messagesContainer: HTMLDivElement;
  let textarea: HTMLTextAreaElement;
  let messageId = 0;
  let isLoading = false;

  // Streaming state
  let currentChunk = '';
  let currentThought = '';

  // User question modal state
  interface UserQuestion {
    requestId: string;
    question: string;
    options?: { label: string; description?: string }[];
  }
  let userQuestion: UserQuestion | null = null;
  let userAnswerInput = '';

  // Subagent collapse state
  let expandedSubagents: Set<string> = new Set();

  // Font scaling (Cmd/Ctrl +/-)
  let fontScale = 1;
  const FONT_SCALE_KEY = 'ccui-font-scale';

  function loadFontScale() {
    const stored = localStorage.getItem(FONT_SCALE_KEY);
    if (stored) fontScale = parseFloat(stored);
  }

  function saveFontScale() {
    localStorage.setItem(FONT_SCALE_KEY, fontScale.toString());
  }

  function handleGlobalKeydown(e: KeyboardEvent) {
    if (!(e.metaKey || e.ctrlKey)) return;
    if (e.key === '=' || e.key === '+') {
      e.preventDefault();
      fontScale = Math.min(fontScale + 0.1, 2);
      saveFontScale();
      console.log('Zoom in:', fontScale);
    } else if (e.key === '-') {
      e.preventDefault();
      fontScale = Math.max(fontScale - 0.1, 0.5);
      saveFontScale();
      console.log('Zoom out:', fontScale);
    } else if (e.key === '0') {
      e.preventDefault();
      fontScale = 1;
      saveFontScale();
      console.log('Zoom reset:', fontScale);
    }
  }

  function toggleSubagent(id: string) {
    if (expandedSubagents.has(id)) {
      expandedSubagents.delete(id);
    } else {
      expandedSubagents.add(id);
    }
    expandedSubagents = expandedSubagents; // trigger reactivity
  }

  function getChildTools(parentId: string): ToolCall[] {
    return messages
      .filter(m => m.toolState?.parentId === parentId)
      .map(m => m.toolState!)
      .filter(Boolean);
  }

  function isSubagent(tool: ToolCall): boolean {
    return tool.toolName === 'Task';
  }

  function hasParent(tool: ToolCall): boolean {
    return !!tool.parentId;
  }


  function autoResize() {
    if (!textarea) return;
    textarea.style.height = 'auto';
    textarea.style.height = Math.min(textarea.scrollHeight, 200) + 'px';
  }

  onMount(() => {
    loadFontScale();
    window.addEventListener('keydown', handleGlobalKeydown);

    EventsOn('chat_chunk', (text: string) => {
      currentChunk += text;
    });

    EventsOn('chat_thought', (text: string) => {
      currentThought += text;
    });

    EventsOn('tool_state', (state: ToolCall) => {
      // Find existing tool message or create new one
      const idx = messages.findIndex(m => m.toolState?.id === state.id);
      if (idx >= 0) {
        messages[idx].toolState = state;
        messages = messages;
      } else {
        messages = [...messages, {
          id: ++messageId,
          text: '',
          sender: 'tool',
          toolState: state
        }];
      }
    });

    EventsOn('prompt_complete', (stopReason: string) => {
      console.log('Prompt complete:', stopReason);
      if (currentChunk) {
        messages = [...messages, { id: ++messageId, text: currentChunk, sender: 'bot' }];
        currentChunk = '';
      }
      currentThought = '';
      isLoading = false;
    });

    EventsOn('error', (err: string) => {
      console.error('Error:', err);
      messages = [...messages, { id: ++messageId, text: `Error: ${err}`, sender: 'bot' }];
      isLoading = false;
    });

    EventsOn('user_question', (q: UserQuestion) => {
      console.log('User question:', q);
      userQuestion = q;
      userAnswerInput = '';
    });

    console.log('Event listeners registered');
  });

  afterUpdate(() => {
    if (messagesContainer) {
      messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }
  });

  function sendMessage() {
    const text = inputText.trim();
    if (!text || isLoading) return;

    if (currentChunk) {
      messages = [...messages, { id: ++messageId, text: currentChunk, sender: 'bot' }];
      currentChunk = '';
    }

    messages = [...messages, { id: ++messageId, text, sender: 'user' }];
    EventsEmit('send_message', text);
    inputText = '';
    isLoading = true;
    if (textarea) textarea.style.height = 'auto';
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  }

  function respondPermission(optionId: string) {
    EventsEmit('permission_response', optionId);
  }

  function cancelRequest() {
    EventsEmit('cancel');
    isLoading = false;
  }

  function submitUserAnswer(answer?: string) {
    if (!userQuestion) return;
    const finalAnswer = answer || userAnswerInput.trim();
    if (!finalAnswer) return;
    EventsEmit('user_answer', { requestId: userQuestion.requestId, answer: finalAnswer });
    userQuestion = null;
    userAnswerInput = '';
  }

  function getToolStatusIcon(status: string): string {
    switch (status) {
      case 'pending': return '○';
      case 'awaiting_permission': return '⊘';
      case 'running': return '◐';
      case 'completed': return '●';
      case 'error': return '✕';
      default: return '○';
    }
  }

  function getToolStatusColor(status: string): string {
    switch (status) {
      case 'pending': return 'text-ink-muted';
      case 'awaiting_permission': return 'text-accent-wine';
      case 'running': return 'text-accent-copper';
      case 'completed': return 'text-accent-sage';
      case 'error': return 'text-accent-wine';
      default: return 'text-ink-muted';
    }
  }

  function getPermissionButtonClass(kind: string): string {
    if (kind.startsWith('allow')) {
      return 'bg-accent-sage/10 border-accent-sage/40 text-accent-sage hover:bg-accent-sage/20';
    } else if (kind.startsWith('reject')) {
      return 'bg-accent-wine/10 border-accent-wine/40 text-accent-wine hover:bg-accent-wine/20';
    }
    return 'bg-parchment-dark/50 border-ink-muted/30 text-ink-medium hover:bg-parchment-dark';
  }

  function formatDiffLine(line: string): { class: string; text: string } {
    if (line.startsWith('+')) {
      return { class: 'bg-diff-add-bg text-diff-add-text border-l-2 border-diff-add-border', text: line };
    } else if (line.startsWith('-')) {
      return { class: 'bg-diff-remove-bg text-diff-remove-text border-l-2 border-diff-remove-border', text: line };
    }
    return { class: 'text-ink-light border-l-2 border-transparent', text: line };
  }

  function getShortPath(path: string): string {
    const parts = path.split('/');
    return parts.slice(-2).join('/');
  }
</script>

<div class="h-full p-5 bg-ink-dark" style="zoom: {fontScale}">
<div class="h-full flex flex-col bg-parchment-light rounded-2xl overflow-hidden shadow-xl">
  <!-- Header -->
  <header class="px-8 py-5 border-b border-ink-muted/15 bg-parchment-glow/80 flex justify-between items-center">
    <div class="flex items-center gap-5">
      <div class="flex items-center gap-3">
        <span class="font-editorial text-3xl font-medium text-ink-deep tracking-tight italic">Correspondence</span>
      </div>
      <span class="ornament text-2xl opacity-30">❧</span>
    </div>
    {#if isLoading}
      <button
        on:click={cancelRequest}
        class="px-5 py-2 text-sm font-medium text-accent-wine border border-accent-wine/30 rounded-full
               hover:bg-accent-wine/10 transition-all duration-300"
      >
        Cease
      </button>
    {/if}
  </header>

  <!-- Messages -->
  <div
    bind:this={messagesContainer}
    class="flex-1 overflow-y-auto px-8 py-8 space-y-6 relative"
  >
    {#each messages as msg, i (msg.id)}
      {#if msg.sender === 'tool' && msg.toolState}
        <!-- Skip child tools - they render inside their parent -->
        {#if !hasParent(msg.toolState)}
          {@const tool = msg.toolState}
          {#if isSubagent(tool)}
            <!-- Subagent (Task) - collapsible container -->
            {@const children = getChildTools(tool.id)}
            {@const isExpanded = expandedSubagents.has(tool.id)}
            <div class="animate-paper-unfold" style="animation-delay: {i * 0.05}s">
              <div class="rounded-lg bg-parchment-glow border border-accent-copper/30 overflow-hidden shadow-sm">
                <!-- Collapsible header -->
                <button
                  on:click={() => toggleSubagent(tool.id)}
                  class="w-full px-6 py-4 flex items-center gap-4 hover:bg-parchment-dark/20 transition-colors"
                >
                  <span class="text-ink-muted text-sm transition-transform {isExpanded ? 'rotate-90' : ''}">▶</span>
                  <span class="{getToolStatusColor(tool.status)} {tool.status === 'running' ? 'animate-spin' : ''} text-sm">
                    {getToolStatusIcon(tool.status)}
                  </span>
                  <span class="text-ink-dark text-sm font-medium">{tool.title || 'Subagent'}</span>
                  <span class="text-ink-muted text-xs font-mono-refined">[{children.length} tools]</span>
                  <span class="text-ink-muted text-xs ml-auto font-mono-refined">{tool.status}</span>
                </button>

                <!-- Expanded content: child tools -->
                {#if isExpanded}
                  <div class="border-t border-ink-muted/15 bg-parchment-base/30 px-4 py-3 space-y-3">
                    {#each children as childTool}
                      <div class="rounded-lg bg-parchment-glow border border-ink-muted/15 overflow-hidden">
                        <div class="px-5 py-3 flex items-center gap-3">
                          <span class="{getToolStatusColor(childTool.status)} {childTool.status === 'running' ? 'animate-spin' : ''} text-xs">
                            {getToolStatusIcon(childTool.status)}
                          </span>
                          <span class="text-ink-dark text-xs font-medium truncate">{childTool.title || childTool.kind}</span>
                          <span class="text-ink-muted text-[10px] ml-auto font-mono-refined">{childTool.status}</span>
                        </div>

                        {#if childTool.diff?.structuredPatch?.length}
                          <div class="border-t border-ink-muted/10">
                            <div class="px-5 py-2 bg-parchment-dark/30 text-xs text-ink-medium font-mono-refined">
                              {getShortPath(childTool.diff.filePath || '')}
                            </div>
                            <div class="p-3 font-mono-refined text-xs overflow-x-auto bg-parchment-glow">
                              {#each childTool.diff.structuredPatch as hunk}
                                <div class="text-ink-muted/50 mb-1 text-[9px]">@@ -{hunk.oldStart},{hunk.oldLines} +{hunk.newStart},{hunk.newLines} @@</div>
                                {#each hunk.lines as line}
                                  {@const formatted = formatDiffLine(line)}
                                  <div class="{formatted.class} px-2 text-[11px]">{formatted.text}</div>
                                {/each}
                              {/each}
                            </div>
                          </div>
                        {:else if childTool.diff?.content}
                          <div class="border-t border-ink-muted/10">
                            <div class="px-5 py-2 bg-parchment-dark/30 text-xs text-ink-medium font-mono-refined flex items-center gap-3">
                              {getShortPath(childTool.diff.filePath || '')}
                              <span class="text-accent-sage text-[10px] ml-auto font-medium">New</span>
                            </div>
                            <div class="p-4 font-mono-refined text-xs overflow-x-auto bg-parchment-glow text-accent-sage/90">
                              <pre class="whitespace-pre-wrap leading-relaxed">{childTool.diff.content}</pre>
                            </div>
                          </div>
                        {/if}

                        {#if childTool.status === 'awaiting_permission' && childTool.permissionOptions?.length}
                          <div class="px-5 py-3 border-t border-ink-muted/10 bg-parchment-base/40 flex flex-wrap gap-2">
                            {#each childTool.permissionOptions as opt}
                              <button
                                on:click={() => respondPermission(opt.optionId)}
                                class="px-4 py-1.5 rounded-full text-xs font-medium border transition-all duration-300 {getPermissionButtonClass(opt.kind)}"
                              >
                                {opt.name}
                              </button>
                            {/each}
                          </div>
                        {/if}
                      </div>
                    {/each}
                  </div>
                {/if}
              </div>
            </div>
          {:else}
            <!-- Regular tool call message -->
            <div class="animate-paper-unfold" style="animation-delay: {i * 0.05}s">
              <div class="rounded-lg bg-parchment-glow border border-ink-muted/20 overflow-hidden shadow-sm">
                <div class="px-6 py-4 flex items-center gap-4">
                  <span class="{getToolStatusColor(tool.status)} {tool.status === 'running' ? 'animate-spin' : ''} text-sm">
                    {getToolStatusIcon(tool.status)}
                  </span>
                  <span class="text-ink-dark text-sm font-medium">{tool.title || tool.kind}</span>
                  <span class="text-ink-muted text-xs ml-auto font-mono-refined">{tool.status}</span>
                </div>

                {#if tool.diff?.structuredPatch?.length}
                  <div class="border-t border-ink-muted/15">
                    <div class="px-6 py-3 bg-parchment-dark/40 text-sm text-ink-medium font-mono-refined">
                      {getShortPath(tool.diff.filePath || '')}
                    </div>
                    <div class="p-4 font-mono-refined text-sm overflow-x-auto bg-parchment-glow">
                      {#each tool.diff.structuredPatch as hunk}
                        <div class="text-ink-muted/50 mb-1 text-[10px]">@@ -{hunk.oldStart},{hunk.oldLines} +{hunk.newStart},{hunk.newLines} @@</div>
                        {#each hunk.lines as line}
                          {@const formatted = formatDiffLine(line)}
                          <div class="{formatted.class} px-2 text-[13px]">{formatted.text}</div>
                        {/each}
                      {/each}
                    </div>
                  </div>
                {:else if tool.diff?.content}
                  <div class="border-t border-ink-muted/15">
                    <div class="px-6 py-3 bg-parchment-dark/40 text-sm text-ink-medium font-mono-refined flex items-center gap-4">
                      {getShortPath(tool.diff.filePath || '')}
                      <span class="text-accent-sage text-xs ml-auto font-medium tracking-wide">New</span>
                    </div>
                    <div class="p-5 font-mono-refined text-sm overflow-x-auto bg-parchment-glow text-accent-sage/90">
                      <pre class="whitespace-pre-wrap leading-relaxed">{tool.diff.content}</pre>
                    </div>
                  </div>
                {:else if tool.diffs?.length}
                  {#each tool.diffs as diff}
                    {#if diff.type === 'diff'}
                      <div class="border-t border-ink-muted/15">
                        <div class="px-6 py-3 bg-parchment-dark/40 text-sm text-ink-medium font-mono-refined">
                          {getShortPath(diff.path || '')}
                        </div>
                        <div class="p-4 font-mono-refined text-sm space-y-1">
                          {#if diff.oldText}
                            <div class="bg-diff-remove-bg text-diff-remove-text border-l-2 border-diff-remove-border px-4 py-1 text-[13px]">- {diff.oldText}</div>
                          {/if}
                          {#if diff.newText}
                            <div class="bg-diff-add-bg text-diff-add-text border-l-2 border-diff-add-border px-4 py-1 text-[13px]">+ {diff.newText}</div>
                          {/if}
                        </div>
                      </div>
                    {/if}
                  {/each}
                {/if}

                {#if tool.status === 'awaiting_permission' && tool.permissionOptions?.length}
                  <div class="px-6 py-4 border-t border-ink-muted/15 bg-parchment-base/40 flex flex-wrap gap-3">
                    {#each tool.permissionOptions as opt}
                      <button
                        on:click={() => respondPermission(opt.optionId)}
                        class="px-5 py-2 rounded-full text-sm font-medium border transition-all duration-300 {getPermissionButtonClass(opt.kind)}"
                      >
                        {opt.name}
                      </button>
                    {/each}
                  </div>
                {/if}
              </div>
            </div>
          {/if}
        {/if}
      {:else if msg.sender !== 'tool'}
        <!-- Regular message -->
        <div class="flex {msg.sender === 'user' ? 'justify-end' : 'justify-start'} animate-fade-rise" style="animation-delay: {i * 0.05}s">
          {#if msg.sender === 'user'}
            <div class="max-w-[75%] relative">
              <div class="absolute -left-6 top-3 text-ink-muted/30 font-editorial text-lg italic">You</div>
              <div class="bg-parchment-glow border border-accent-copper/20 rounded-2xl rounded-br-sm px-7 py-5 shadow-sm">
                <p class="text-[15px] leading-relaxed whitespace-pre-wrap break-words text-ink-deep">{msg.text}</p>
              </div>
            </div>
          {:else}
            <div class="max-w-[75%] relative">
              <div class="absolute -left-8 top-3 ornament text-xl opacity-40">✦</div>
              <div class="bg-parchment-base/70 border border-ink-muted/15 rounded-2xl rounded-bl-sm px-7 py-5 prose prose-sm prose-ink">
                {@html marked(msg.text)}
              </div>
            </div>
          {/if}
        </div>
      {/if}
    {/each}

    <!-- Streaming chunk -->
    {#if currentChunk}
      <div class="flex justify-start animate-fade-rise">
        <div class="max-w-[75%] relative">
          <div class="absolute -left-8 top-3 ornament text-xl opacity-40 animate-quill">✦</div>
          <div class="bg-parchment-base/70 border border-accent-copper/25 rounded-2xl rounded-bl-sm px-7 py-5 prose prose-sm prose-ink">
            {@html marked(currentChunk)}<span class="inline-block w-0.5 h-5 bg-accent-copper ml-1 animate-gentle-pulse align-middle"></span>
          </div>
        </div>
      </div>
    {/if}

    <!-- Thinking indicator -->
    {#if currentThought}
      <div class="flex justify-start animate-fade-rise">
        <div class="max-w-[65%] px-6 py-5 rounded-xl bg-parchment-dark/30 border border-ink-muted/10">
          <div class="flex items-center gap-3 text-sm text-ink-muted mb-3">
            <span class="font-editorial italic">Contemplating</span>
            <span class="animate-gentle-pulse">...</span>
          </div>
          <p class="text-sm leading-relaxed text-ink-light/70 font-mono-refined">{currentThought.slice(-200)}{currentThought.length > 200 ? '...' : ''}</p>
        </div>
      </div>
    {/if}


    {#if messages.length === 0 && !currentChunk}
      <div class="absolute inset-0 flex flex-col items-center justify-center text-center px-12">
        <div class="ornament text-6xl mb-8 text-ink-muted/30">❦</div>
        <p class="font-editorial text-2xl text-ink-light/60 italic mb-3">Begin your correspondence</p>
        <p class="text-ink-muted text-sm">Compose a message below to start</p>
      </div>
    {/if}
  </div>

  <!-- User Question Modal -->
  {#if userQuestion}
    <div class="absolute inset-0 bg-ink-dark/50 flex items-center justify-center z-50">
      <div class="bg-parchment-glow border border-ink-muted/30 rounded-2xl shadow-2xl max-w-lg w-full mx-6 overflow-hidden animate-fade-rise">
        <div class="px-8 py-6 border-b border-ink-muted/15">
          <h2 class="font-editorial text-xl text-ink-deep">Question from Assistant</h2>
        </div>
        <div class="px-8 py-6">
          <p class="text-ink-dark text-[15px] leading-relaxed mb-6">{userQuestion.question}</p>

          {#if userQuestion.options?.length}
            <div class="space-y-2 mb-6">
              {#each userQuestion.options as opt}
                <button
                  on:click={() => submitUserAnswer(opt.label)}
                  class="w-full px-5 py-3 text-left rounded-lg border border-ink-muted/20
                         bg-parchment-base/60 hover:bg-parchment-dark/40 transition-colors"
                >
                  <span class="text-ink-dark font-medium text-sm">{opt.label}</span>
                  {#if opt.description}
                    <p class="text-ink-muted text-xs mt-1">{opt.description}</p>
                  {/if}
                </button>
              {/each}
            </div>
          {/if}

          <textarea
            bind:value={userAnswerInput}
            placeholder="Type your response..."
            rows="3"
            class="w-full px-4 py-3 bg-parchment-base/60 border border-ink-muted/20 rounded-lg
                   text-ink-deep text-[15px] placeholder-ink-muted/50 resize-none leading-normal
                   focus:outline-none focus:border-accent-copper/40 focus:bg-parchment-glow
                   transition-all duration-300"
          ></textarea>
        </div>
        <div class="px-8 py-4 border-t border-ink-muted/15 bg-parchment-base/40 flex justify-end gap-3">
          <button
            on:click={() => submitUserAnswer()}
            disabled={!userAnswerInput.trim()}
            class="px-6 py-2 bg-accent-copper text-parchment-glow font-medium rounded-lg text-sm
                   hover:bg-accent-rust transition-all duration-200
                   disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Submit
          </button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Input -->
  <div class="px-8 py-4 border-t border-ink-muted/15 bg-parchment-glow/80">
    <div class="flex gap-3 items-end">
      <textarea
        bind:this={textarea}
        bind:value={inputText}
        on:keydown={handleKeydown}
        on:input={autoResize}
        placeholder="Write your message..."
        rows="1"
        disabled={isLoading}
        class="flex-1 px-4 py-3 bg-parchment-base/60 border border-ink-muted/20 rounded-lg
               text-ink-deep text-[15px] placeholder-ink-muted/50 resize-none leading-normal
               focus:outline-none focus:border-accent-copper/40 focus:bg-parchment-glow
               transition-all duration-300 disabled:opacity-50"
      ></textarea>
      <button
        on:click={sendMessage}
        disabled={isLoading}
        class="px-6 py-3 bg-accent-copper text-parchment-glow font-medium rounded-lg text-sm
               hover:bg-accent-rust transition-all duration-200
               disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {#if isLoading}
          <span class="inline-block animate-spin">◐</span>
        {:else}
          Send
        {/if}
      </button>
    </div>
  </div>
</div>
</div>
