<script lang="ts">
  import type { PlanEntry } from './shared';
  import { getPlanStatusIndicator } from './shared';

  export let entries: PlanEntry[] = [];

  function getStatusClass(status: string): string {
    switch (status) {
      case 'completed': return 'text-accent-success';
      case 'in_progress': return 'text-ink animate-pulse';
      default: return 'text-ink-muted';
    }
  }

  function getPriorityClass(priority: string): string {
    switch (priority) {
      case 'high': return 'text-ink';
      case 'low': return 'text-ink-faint';
      default: return 'text-ink-medium';
    }
  }
</script>

{#if entries.length > 0}
  <div class="border border-ink-faint p-3 mb-3 bg-paper-dim">
    <div class="text-xs text-ink-muted mb-2 uppercase tracking-wide">Plan</div>
    <ul class="space-y-1">
      {#each entries as entry}
        <li class="flex items-start gap-2 text-sm">
          <span class={getStatusClass(entry.status)}>{getPlanStatusIndicator(entry.status)}</span>
          <span class={getPriorityClass(entry.priority)}>{entry.content}</span>
        </li>
      {/each}
    </ul>
  </div>
{/if}
