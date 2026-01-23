<script lang="ts">
  import DiffView from './DiffView.svelte';
  import { createEventDispatcher } from 'svelte';
  import { type ToolCall, getStatusIndicator, getStatusClass, getButtonClass } from './shared';

  export let tool: ToolCall;
  export let compact: boolean = false;

  const dispatch = createEventDispatcher();

  $: isEdit = tool.toolName === 'Edit' && tool.status === 'awaiting_permission';
  $: isWrite = tool.toolName === 'Write';
  $: showInputPreview = (isEdit && (tool.input?.old_string || tool.input?.new_string)) || (isWrite && tool.input?.content);
</script>

<div class="border border-ink-faint">
  <div class="{compact ? 'px-3 py-2 gap-2' : 'px-4 py-3 gap-3'} flex items-center">
    <span class="{getStatusClass(tool.status)} text-xs {tool.status === 'running' ? 'animate-pulse-subtle' : ''}">
      {getStatusIndicator(tool.status)}
    </span>
    <span class="{compact ? 'text-ink-medium text-xs' : 'text-ink text-sm'} truncate">{tool.title || tool.kind}</span>
  </div>

  {#if showInputPreview}
    <DiffView
      filePath={String(tool.input?.file_path || '')}
      oldText={isEdit ? String(tool.input?.old_string || '') : undefined}
      newText={isEdit ? String(tool.input?.new_string || '') : undefined}
      content={isWrite ? String(tool.input?.content || '') : undefined}
      isNew={isWrite}
    />
  {/if}

  {#if tool.diff?.structuredPatch?.length}
    <DiffView filePath={tool.diff.filePath || ''} structuredPatch={tool.diff.structuredPatch} />
  {:else if tool.diff?.content}
    <DiffView filePath={tool.diff.filePath || ''} content={tool.diff.content} isNew />
  {/if}

  {#if tool.diffs?.length}
    {#each tool.diffs.filter(d => d.type === 'diff') as diff}
      <DiffView filePath={diff.path || ''} oldText={diff.oldText} newText={diff.newText} />
    {/each}
  {/if}

  {#if tool.status === 'awaiting_permission' && tool.permissionOptions?.length}
    <div class="{compact ? 'px-3 py-2' : 'px-4 py-3'} border-t border-ink-faint flex flex-wrap gap-2">
      {#each tool.permissionOptions as opt}
        <button
          on:click={() => dispatch('permission', opt.optionId)}
          class="{compact ? 'px-3 py-1 text-xs' : 'px-4 py-1.5 text-sm'} border transition-colors {getButtonClass(opt.kind)}"
        >
          {opt.name}
        </button>
      {/each}
    </div>
  {/if}
</div>
