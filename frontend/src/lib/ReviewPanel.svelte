<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { marked } from 'marked';
  import ReviewDiff from './ReviewDiff.svelte';
  import type { FileChange, ReviewComment } from './shared';

  export let fileChanges: FileChange[] = [];
  export let comments: ReviewComment[] = [];
  export let agentOutput = '';
  export let agentRunning = false;

  let generalComment = '';
  let commentInput = '';
  let pendingComment: { type: 'line' | 'hunk'; filePath: string; lineNumber?: number; hunkIndex?: number } | null = null;

  const dispatch = createEventDispatcher<{
    addComment: ReviewComment;
    removeComment: string;
    submitReview: void;
  }>();

  function handleAddComment(e: CustomEvent<{ type: 'line' | 'hunk'; lineNumber?: number; hunkIndex?: number }>, filePath: string) {
    pendingComment = { ...e.detail, filePath };
    commentInput = '';
  }

  function submitComment() {
    if (!pendingComment || !commentInput.trim()) return;
    const comment: ReviewComment = {
      id: crypto.randomUUID(),
      type: pendingComment.type,
      filePath: pendingComment.filePath,
      lineNumber: pendingComment.lineNumber,
      hunkIndex: pendingComment.hunkIndex,
      text: commentInput.trim()
    };
    dispatch('addComment', comment);
    pendingComment = null;
    commentInput = '';
  }

  function cancelComment() {
    pendingComment = null;
    commentInput = '';
  }

  function submitReview() {
    if (generalComment.trim()) {
      dispatch('addComment', {
        id: crypto.randomUUID(),
        type: 'general',
        text: generalComment.trim()
      });
    }
    dispatch('submitReview');
    generalComment = '';
  }

  // Reactive comment maps for proper Svelte reactivity
  $: commentsByFile = comments.reduce((acc, c) => {
    if (c.filePath) (acc[c.filePath] ||= []).push(c);
    return acc;
  }, {} as Record<string, ReviewComment[]>);

  $: generalComments = comments.filter(c => c.type === 'general');
</script>

<div class="h-full flex flex-col overflow-hidden">
  <!-- File changes -->
  <div class="flex-1 overflow-y-auto px-6 py-6 space-y-4">
    {#if fileChanges.length === 0}
      <div class="text-center text-ink-muted py-12">
        <p>No file changes yet</p>
        <p class="text-sm mt-2">Changes will appear here as the agent edits files</p>
      </div>
    {:else}
      {#each fileChanges as fc}
        <ReviewDiff
          fileChange={fc}
          comments={commentsByFile[fc.filePath] || []}
          on:addComment={(e) => handleAddComment(e, fc.filePath)}
          on:removeComment={(e) => dispatch('removeComment', e.detail)}
        />
      {/each}
    {/if}

    <!-- General comments -->
    {#each generalComments as comment}
      <div class="px-4 py-3 border border-ink-faint bg-paper-dim flex items-start gap-2">
        <span class="text-xs text-ink-muted">General:</span>
        <span class="text-ink-medium text-sm flex-1">{comment.text}</span>
        <button
          on:click={() => dispatch('removeComment', comment.id)}
          class="text-ink-muted hover:text-accent-danger text-xs"
        >x</button>
      </div>
    {/each}

    <!-- Agent output -->
    {#if agentOutput || agentRunning}
      <div class="border border-ink-faint">
        <div class="px-4 py-2 bg-paper-dim border-b border-ink-faint flex items-center gap-2">
          <span class="text-xs text-ink-muted uppercase tracking-wide">Review Agent</span>
          {#if agentRunning}
            <span class="text-xs text-ink-medium animate-pulse-subtle">...</span>
          {/if}
        </div>
        <div class="px-4 py-3 prose prose-sm prose-eink max-h-64 overflow-y-auto">
          {#if agentOutput}
            {@html marked(agentOutput)}
          {:else}
            <span class="text-ink-muted">Processing...</span>
          {/if}
        </div>
      </div>
    {/if}
  </div>

  <!-- Comment input modal -->
  {#if pendingComment}
    <div class="absolute inset-0 bg-paper/90 flex items-center justify-center z-50">
      <div class="bg-paper border border-ink-faint max-w-md w-full mx-6">
        <div class="px-4 py-3 border-b border-ink-faint">
          <span class="text-sm text-ink">Add {pendingComment.type} comment</span>
          <span class="text-xs text-ink-muted ml-2">
            {pendingComment.filePath}{pendingComment.lineNumber ? `:${pendingComment.lineNumber}` : ''}
          </span>
        </div>
        <div class="p-4">
          <textarea
            bind:value={commentInput}
            placeholder="Enter your comment..."
            rows="3"
            class="w-full px-3 py-2 bg-paper border border-ink-faint text-ink text-sm placeholder-ink-muted resize-none focus:outline-none focus:border-ink-muted"
          ></textarea>
        </div>
        <div class="px-4 py-3 border-t border-ink-faint flex justify-end gap-2">
          <button on:click={cancelComment} class="px-3 py-1.5 text-sm text-ink-medium hover:text-ink">Cancel</button>
          <button on:click={submitComment} disabled={!commentInput.trim()} class="px-4 py-1.5 bg-ink text-paper text-sm disabled:opacity-40">Add</button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Bottom: general comment + submit -->
  <div class="px-6 py-4 border-t border-ink-faint space-y-3">
    <textarea
      bind:value={generalComment}
      placeholder="General comment (optional)..."
      rows="2"
      disabled={agentRunning}
      class="w-full px-4 py-3 bg-paper border border-ink-faint text-ink text-[15px] placeholder-ink-muted resize-none focus:outline-none focus:border-ink-muted disabled:opacity-50"
    ></textarea>
    <div class="flex justify-end">
      <button
        on:click={submitReview}
        disabled={agentRunning || (fileChanges.length === 0 && comments.length === 0 && !generalComment.trim())}
        class="px-5 py-2 bg-ink text-paper text-sm hover:bg-ink-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
      >
        {#if agentRunning}
          <span class="inline-block animate-spin-slow">*</span> Reviewing...
        {:else}
          Submit Review
        {/if}
      </button>
    </div>
  </div>
</div>
