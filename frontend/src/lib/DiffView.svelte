<script lang="ts">
  import type { PatchHunk } from './shared';

  export let filePath: string = '';
  export let structuredPatch: PatchHunk[] | undefined = undefined;
  export let content: string | undefined = undefined;
  export let oldText: string | undefined = undefined;
  export let newText: string | undefined = undefined;
  export let isNew: boolean = false;

  function getShortPath(path: string): string {
    return path.split('/').slice(-2).join('/');
  }

  function formatLine(line: string): { class: string; text: string } {
    if (line.startsWith('+')) {
      return { class: 'bg-diff-add-bg text-diff-add-text border-l-2 border-diff-add-border pl-2', text: line };
    } else if (line.startsWith('-')) {
      return { class: 'bg-diff-remove-bg text-diff-remove-text border-l-2 border-diff-remove-border pl-2', text: line };
    }
    return { class: 'text-ink-light pl-3', text: line };
  }
</script>

<div class="border-t border-ink-faint">
  <div class="px-3 py-1.5 bg-paper-dim text-xs text-ink-muted font-mono flex items-center justify-between">
    <span>{getShortPath(filePath)}</span>
    {#if isNew || content}
      <span class="text-accent-success text-[10px]">new</span>
    {/if}
  </div>

  {#if structuredPatch?.length}
    <div class="px-2 py-2 font-mono text-[11px] overflow-x-auto">
      {#each structuredPatch as hunk}
        <div class="text-ink-muted text-[10px] mb-1">@@ -{hunk.oldStart},{hunk.oldLines} +{hunk.newStart},{hunk.newLines} @@</div>
        {#each hunk.lines as line}
          {@const f = formatLine(line)}
          <div class={f.class}>{f.text}</div>
        {/each}
      {/each}
    </div>
  {:else if content}
    <div class="p-3 font-mono text-[11px] overflow-x-auto text-accent-success">
      <pre class="whitespace-pre-wrap">{content}</pre>
    </div>
  {:else if oldText || newText}
    <div class="px-3 py-3 font-mono text-xs space-y-1 max-h-64 overflow-auto">
      {#if oldText}
        <div class="bg-diff-remove-bg text-diff-remove-text border-l-2 border-diff-remove-border pl-2 whitespace-pre-wrap">- {oldText}</div>
      {/if}
      {#if newText}
        <div class="bg-diff-add-bg text-diff-add-text border-l-2 border-diff-add-border pl-2 whitespace-pre-wrap">+ {newText}</div>
      {/if}
    </div>
  {/if}
</div>
