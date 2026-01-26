<script lang="ts">
  export let type: 'chat' | 'review' | 'terminal' | null = null;
  export let panelId: 'left' | 'right';

  const typeLabels: Record<string, string> = {
    chat: 'Chat',
    review: 'Review',
    terminal: 'Terminal'
  };
</script>

<div data-panel-id={panelId} class="h-full flex flex-col bg-paper">
  {#if type}
    <div
      data-testid="panel-header"
      class="px-4 py-2 border-b border-ink-faint flex items-center justify-between"
    >
      <span class="text-sm text-ink">{typeLabels[type]}</span>
      <span class="text-xs text-ink-muted">Cmd+K</span>
    </div>
    <div
      data-testid="panel-content-{type}"
      class="flex-1 overflow-auto p-4"
    >
      <div class="text-ink-muted text-sm">
        {typeLabels[type]} content placeholder
      </div>
    </div>
  {:else}
    <div
      data-testid="panel-empty"
      class="h-full flex items-center justify-center text-ink-muted text-sm"
    >
      Press Cmd+K to open panel
    </div>
  {/if}
</div>
