<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import type { AutomationRun } from './shared';
  import { GetTriageItems, MarkRunRead, ListAutomations } from '../../wailsjs/go/main/App';
  import type { Automation } from './shared';

  const dispatch = createEventDispatcher<{ viewRun: { run: AutomationRun; automationName: string } }>();

  let items: AutomationRun[] = [];
  let automations: Automation[] = [];
  let loading = true;

  export async function refresh() {
    loading = true;
    try {
      [items, automations] = await Promise.all([
        GetTriageItems().then(r => r || []),
        ListAutomations().then(r => r || [])
      ]);
    } catch {
      items = [];
    }
    loading = false;
  }

  onMount(refresh);

  async function markRead(run: AutomationRun) {
    try {
      await MarkRunRead(run.automationId, run.id);
      items = items.filter(i => i.id !== run.id);
    } catch (e) {
      console.error('mark read failed', e);
    }
  }

  function getAutoName(id: string): string {
    return automations.find(a => a.id === id)?.name || id;
  }

  function formatTime(iso: string): string {
    if (!iso) return '';
    return new Date(iso).toLocaleString();
  }
</script>

<div class="h-full overflow-y-auto">
  <div class="p-6">
    <h2 class="text-ink font-medium mb-4">Triage ({items.length})</h2>

    {#if loading}
      <p class="text-ink-muted text-sm">Loading...</p>
    {:else if items.length === 0}
      <div class="py-12 text-center">
        <p class="text-ink-muted text-sm">No unread findings</p>
      </div>
    {:else}
      <div class="space-y-2">
        {#each items as item (item.id)}
          <div class="border border-ink-faint p-4 hover:border-ink-muted transition-colors">
            <div class="flex justify-between items-start">
              <button
                on:click={() => dispatch('viewRun', { run: item, automationName: getAutoName(item.automationId) })}
                class="flex-1 text-left min-w-0"
              >
                <div class="flex items-center gap-2">
                  <span class="w-2 h-2 rounded-full bg-accent-warning flex-shrink-0"></span>
                  <span class="text-sm text-ink font-medium truncate">{getAutoName(item.automationId)}</span>
                </div>
                <p class="text-xs text-ink-muted mt-1">{formatTime(item.startedAt)}</p>
                {#if item.output}
                  <p class="text-xs text-ink-muted mt-1 truncate">{item.output.slice(0, 120)}</p>
                {/if}
              </button>
              <button
                on:click={() => markRead(item)}
                class="px-2 py-1 text-xs border border-ink-faint text-ink-muted hover:border-ink-muted hover:text-ink transition-colors ml-2 flex-shrink-0"
              >
                Dismiss
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
