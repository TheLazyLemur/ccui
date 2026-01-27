<script lang="ts">
  import { createEventDispatcher, afterUpdate } from 'svelte';
  import { marked } from 'marked';
  import ToolCard from './ToolCard.svelte';
  import PlanPanel from './PlanPanel.svelte';
  import { type Message, type ToolCall, type PlanEntry, getStatusIndicator, getStatusClass } from './shared';

  export let messages: Message[] = [];
  export let currentChunk = '';
  export let currentThought = '';
  export let planEntries: PlanEntry[] = [];
  export let expandedSubagents: Set<string> = new Set();
  export let getChildTools: (parentId: string) => ToolCall[];
  export let toggleSubagent: (id: string) => void;

  const dispatch = createEventDispatcher<{ permission: { toolId: string; optionId: string } }>();

  function respondPermission(e: CustomEvent<{ toolId: string; optionId: string }>) {
    dispatch('permission', e.detail);
  }

  let container: HTMLDivElement;

  afterUpdate(() => {
    if (container) container.scrollTop = container.scrollHeight;
  });
</script>

<div bind:this={container} class="h-full overflow-y-auto px-6 py-6 space-y-5">
  <PlanPanel entries={planEntries} />
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
    <div class="h-full flex flex-col items-center justify-center px-8">
      <div class="text-6xl text-ink-faint mb-6">※</div>
      <p class="text-ink-light text-center max-w-sm leading-relaxed">Begin a conversation below</p>
    </div>
  {/if}
</div>
