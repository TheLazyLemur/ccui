<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { marked } from 'marked';
  import type { AutomationRun } from './shared';

  export let run: AutomationRun;
  export let automationName = '';

  const dispatch = createEventDispatcher<{ back: void }>();

  function formatTime(iso: string): string {
    if (!iso) return '';
    return new Date(iso).toLocaleString();
  }

  $: renderedOutput = run.output ? marked(run.output) : '';
</script>

<div class="h-full overflow-y-auto">
  <div class="p-6 max-w-3xl">
    <button on:click={() => dispatch('back')} class="text-sm text-ink-muted hover:text-ink transition-colors mb-4">
      &larr; Back
    </button>

    <div class="mb-4">
      <h2 class="text-ink font-medium">{automationName || 'Run Detail'}</h2>
      <div class="flex gap-3 mt-1 text-xs text-ink-muted">
        <span class="px-1.5 py-0.5 border border-ink-faint {run.status === 'completed' ? 'text-accent-success' : run.status === 'failed' ? 'text-accent-danger' : 'text-ink-medium'}">
          {run.status}
        </span>
        <span>{formatTime(run.startedAt)}</span>
        {#if run.completedAt}
          <span>&rarr; {formatTime(run.completedAt)}</span>
        {/if}
        {#if run.hasFindings}
          <span class="text-accent-warning">Has findings</span>
        {/if}
      </div>
    </div>

    {#if run.error}
      <div class="mb-4 px-4 py-3 border border-accent-danger text-accent-danger text-sm">
        {run.error}
      </div>
    {/if}

    {#if run.output}
      <div class="prose prose-sm prose-eink border border-ink-faint p-4">
        {@html renderedOutput}
      </div>
    {:else}
      <p class="text-ink-muted text-sm">No output</p>
    {/if}
  </div>
</div>
