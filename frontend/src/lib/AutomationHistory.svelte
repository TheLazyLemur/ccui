<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import type { AutomationRun, Automation } from './shared';
  import { GetAutomationRuns, ListAutomations } from '../../wailsjs/go/main/App';

  const dispatch = createEventDispatcher<{ viewRun: { run: AutomationRun; automationName: string } }>();

  let automations: Automation[] = [];
  let selectedAutoId = '';
  let runs: AutomationRun[] = [];
  let loading = true;

  onMount(async () => {
    try {
      automations = (await ListAutomations()) || [];
      if (automations.length > 0) {
        selectedAutoId = automations[0].id;
        await loadRuns();
      }
    } catch { /* empty */ }
    loading = false;
  });

  async function loadRuns() {
    if (!selectedAutoId) { runs = []; return; }
    try {
      runs = (await GetAutomationRuns(selectedAutoId)) || [];
    } catch {
      runs = [];
    }
  }

  async function handleSelect(e: Event) {
    selectedAutoId = (e.target as HTMLSelectElement).value;
    await loadRuns();
  }

  function formatTime(iso: string): string {
    if (!iso) return '';
    return new Date(iso).toLocaleString();
  }

  function getAutoName(id: string): string {
    return automations.find(a => a.id === id)?.name || id;
  }
</script>

<div class="h-full overflow-y-auto">
  <div class="p-6">
    <h2 class="text-ink font-medium mb-4">Run History</h2>

    {#if automations.length > 0}
      <div class="mb-4">
        <select on:change={handleSelect} value={selectedAutoId} class="px-3 py-2 bg-paper border border-ink-faint text-ink text-sm focus:outline-none focus:border-ink-muted">
          {#each automations as auto (auto.id)}
            <option value={auto.id}>{auto.name}</option>
          {/each}
        </select>
      </div>
    {/if}

    {#if loading}
      <p class="text-ink-muted text-sm">Loading...</p>
    {:else if runs.length === 0}
      <p class="text-ink-muted text-sm py-8 text-center">No runs yet</p>
    {:else}
      <div class="space-y-2">
        {#each runs as run (run.id)}
          <button
            on:click={() => dispatch('viewRun', { run, automationName: getAutoName(run.automationId) })}
            class="w-full text-left border border-ink-faint p-3 hover:border-ink-muted transition-colors"
          >
            <div class="flex justify-between items-center">
              <div class="flex items-center gap-2">
                <span class="text-xs px-1.5 py-0.5 border border-ink-faint {run.status === 'completed' ? 'text-accent-success' : run.status === 'failed' ? 'text-accent-danger' : 'text-ink-medium'}">
                  {run.status}
                </span>
                <span class="text-sm text-ink">{formatTime(run.startedAt)}</span>
              </div>
              <div class="flex gap-2">
                {#if run.hasFindings}
                  <span class="text-xs text-accent-warning">findings</span>
                {/if}
                {#if !run.read}
                  <span class="w-2 h-2 rounded-full bg-accent-warning"></span>
                {/if}
              </div>
            </div>
            {#if run.output}
              <p class="text-xs text-ink-muted mt-1 truncate">{run.output.slice(0, 120)}</p>
            {/if}
          </button>
        {/each}
      </div>
    {/if}
  </div>
</div>
