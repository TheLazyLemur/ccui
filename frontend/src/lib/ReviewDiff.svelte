<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { FileChange, ReviewComment } from './shared';

  export let fileChange: FileChange;
  export let comments: ReviewComment[] = [];
  export let collapsed = false;

  const dispatch = createEventDispatcher<{
    addComment: { type: 'line' | 'hunk'; lineNumber?: number; hunkIndex?: number };
    removeComment: string;
  }>();

  function getLineComments(lineNum: number): ReviewComment[] {
    return comments.filter(c => c.type === 'line' && c.filePath === fileChange.filePath && c.lineNumber === lineNum);
  }

  function getHunkComments(hunkIdx: number): ReviewComment[] {
    return comments.filter(c => c.type === 'hunk' && c.filePath === fileChange.filePath && c.hunkIndex === hunkIdx);
  }

  function getLineClass(line: string): string {
    if (line.startsWith('+')) return 'bg-accent-success/10 text-accent-success';
    if (line.startsWith('-')) return 'bg-accent-danger/10 text-accent-danger';
    return 'text-ink-medium';
  }
</script>

<div class="border border-ink-faint">
  <button
    on:click={() => collapsed = !collapsed}
    class="w-full px-4 py-2 flex items-center gap-2 text-left hover:bg-paper-dim transition-colors"
  >
    <span class="text-ink-muted text-xs transition-transform {collapsed ? '' : 'rotate-90'}">▸</span>
    <span class="text-ink text-sm font-mono flex-1 truncate">{fileChange.filePath}</span>
    <span class="text-ink-muted text-xs">{fileChange.hunks.length} hunk{fileChange.hunks.length !== 1 ? 's' : ''}</span>
  </button>

  {#if !collapsed}
    <div class="border-t border-ink-faint">
      {#each fileChange.hunks as hunk, hunkIdx}
        <div class="border-b border-ink-faint last:border-b-0">
          <!-- Hunk header -->
          <div class="px-4 py-1 bg-paper-dim flex items-center justify-between">
            <span class="text-xs text-ink-muted font-mono">
              @@ -{hunk.oldStart},{hunk.oldLines} +{hunk.newStart},{hunk.newLines} @@
            </span>
            <button
              on:click={() => dispatch('addComment', { type: 'hunk', hunkIndex: hunkIdx })}
              class="text-ink-muted hover:text-ink text-xs px-1"
              title="Comment on hunk"
            >+</button>
          </div>

          <!-- Hunk comments -->
          {#each getHunkComments(hunkIdx) as comment}
            <div class="px-4 py-2 bg-accent-warning/5 border-l-2 border-accent-warning flex items-start gap-2">
              <span class="text-ink-medium text-sm flex-1">{comment.text}</span>
              <button
                on:click={() => dispatch('removeComment', comment.id)}
                class="text-ink-muted hover:text-accent-danger text-xs"
              >x</button>
            </div>
          {/each}

          <!-- Lines -->
          {#each hunk.lines as line, lineIdx}
            {@const lineNum = hunk.newStart + lineIdx}
            <div class="group flex">
              <span class="w-12 px-2 text-right text-xs text-ink-muted font-mono select-none border-r border-ink-faint bg-paper-dim">
                {lineNum}
              </span>
              <pre class="flex-1 px-3 py-0.5 text-sm font-mono whitespace-pre-wrap break-all {getLineClass(line)}">{line}</pre>
              <button
                on:click={() => dispatch('addComment', { type: 'line', lineNumber: lineNum })}
                class="opacity-0 group-hover:opacity-100 px-2 text-ink-muted hover:text-ink text-xs transition-opacity"
                title="Comment on line"
              >+</button>
            </div>

            <!-- Line comments -->
            {#each getLineComments(lineNum) as comment}
              <div class="ml-12 px-4 py-2 bg-accent-warning/5 border-l-2 border-accent-warning flex items-start gap-2">
                <span class="text-ink-medium text-sm flex-1">{comment.text}</span>
                <button
                  on:click={() => dispatch('removeComment', comment.id)}
                  class="text-ink-muted hover:text-accent-danger text-xs"
                >x</button>
              </div>
            {/each}
          {/each}
        </div>
      {/each}
    </div>
  {/if}
</div>
