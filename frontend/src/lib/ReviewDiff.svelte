<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { diffLines } from 'diff';
  import type { FileChange, ReviewComment, PatchHunk } from './shared';

  export let fileChange: FileChange;
  export let comments: ReviewComment[] = [];
  export let collapsed = false;

  const dispatch = createEventDispatcher<{
    addComment: { type: 'line' | 'hunk'; lineNumber?: number; hunkIndex?: number };
    removeComment: string;
  }>();

  // Compute diff from original vs current (session-start → now)
  $: computedHunks = computeHunks(fileChange.originalContent, fileChange.currentContent);

  // Reactive comment maps - Svelte tracks these dependencies properly
  $: lineCommentsByLine = comments
    .filter(c => c.type === 'line' && c.filePath === fileChange.filePath)
    .reduce((acc, c) => {
      (acc[c.lineNumber!] ||= []).push(c);
      return acc;
    }, {} as Record<number, ReviewComment[]>);

  $: hunkCommentsByIdx = comments
    .filter(c => c.type === 'hunk' && c.filePath === fileChange.filePath)
    .reduce((acc, c) => {
      (acc[c.hunkIndex!] ||= []).push(c);
      return acc;
    }, {} as Record<number, ReviewComment[]>);

  function computeHunks(original: string, current: string): PatchHunk[] {
    const changes = diffLines(original, current);
    const hunks: PatchHunk[] = [];
    let oldLine = 1, newLine = 1;
    let currentHunk: PatchHunk | null = null;
    const CONTEXT = 3;

    for (const change of changes) {
      const lines = change.value.replace(/\n$/, '').split('\n');
      const isAdd = change.added;
      const isRemove = change.removed;

      if (!isAdd && !isRemove) {
        // Context lines - close hunk if gap > CONTEXT*2
        if (currentHunk && lines.length > CONTEXT * 2) {
          // Add trailing context to current hunk
          for (let i = 0; i < CONTEXT && i < lines.length; i++) {
            currentHunk.lines.push(' ' + lines[i]);
            currentHunk.oldLines++;
            currentHunk.newLines++;
          }
          hunks.push(currentHunk);
          currentHunk = null;
        }
        oldLine += lines.length;
        newLine += lines.length;
      } else {
        // Start new hunk if needed
        if (!currentHunk) {
          const leadingContext = Math.min(CONTEXT, oldLine - 1);
          currentHunk = {
            oldStart: oldLine - leadingContext,
            oldLines: 0,
            newStart: newLine - leadingContext,
            newLines: 0,
            lines: []
          };
        }

        for (const line of lines) {
          if (isAdd) {
            currentHunk.lines.push('+' + line);
            currentHunk.newLines++;
            newLine++;
          } else {
            currentHunk.lines.push('-' + line);
            currentHunk.oldLines++;
            oldLine++;
          }
        }
      }
    }

    if (currentHunk) hunks.push(currentHunk);
    return hunks;
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
    <span class="text-ink-muted text-xs">{computedHunks.length} hunk{computedHunks.length !== 1 ? 's' : ''}</span>
  </button>

  {#if !collapsed}
    <div class="border-t border-ink-faint">
      {#each computedHunks as hunk, hunkIdx}
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
          {#each hunkCommentsByIdx[hunkIdx] || [] as comment}
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
            {#each lineCommentsByLine[lineNum] || [] as comment}
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
