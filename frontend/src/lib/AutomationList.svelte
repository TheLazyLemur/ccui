<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import type { Automation } from './shared';
  import { ListAutomations, UpdateAutomation, DeleteAutomation, RunAutomationNow } from '../../wailsjs/go/main/App';

  const dispatch = createEventDispatcher<{ edit: Automation; create: void }>();

  let automations: Automation[] = [];
  let loading = true;

  export async function refresh() {
    loading = true;
    try {
      automations = (await ListAutomations()) || [];
    } catch {
      automations = [];
    }
    loading = false;
  }

  onMount(refresh);

  async function toggleEnabled(auto: Automation) {
    try {
      await UpdateAutomation({ ...auto, enabled: !auto.enabled } as any);
      await refresh();
    } catch (e) {
      console.error('toggle failed', e);
    }
  }

  async function runNow(id: string) {
    try {
      await RunAutomationNow(id);
    } catch (e) {
      console.error('run now failed', e);
    }
  }

  async function remove(id: string) {
    try {
      await DeleteAutomation(id);
      await refresh();
    } catch (e) {
      console.error('delete failed', e);
    }
  }

  function formatSchedule(s: string): string {
    if (!s) return 'Manual only';
    return s;
  }
</script>

<div class="h-full overflow-y-auto">
  <div class="p-6">
    <div class="flex justify-between items-center mb-4">
      <h2 class="text-ink font-medium">Automations</h2>
      <button on:click={() => dispatch('create')} class="px-4 py-1.5 bg-ink text-paper text-sm hover:bg-ink-medium transition-colors">New</button>
    </div>

    {#if loading}
      <p class="text-ink-muted text-sm">Loading...</p>
    {:else if automations.length === 0}
      <div class="py-12 text-center">
        <p class="text-ink-muted text-sm mb-4">No automations yet</p>
        <button on:click={() => dispatch('create')} class="px-4 py-2 border border-ink-faint text-ink-medium text-sm hover:border-ink-muted hover:text-ink transition-colors">Create your first automation</button>
      </div>
    {:else}
      <div class="space-y-2">
        {#each automations as auto (auto.id)}
          <div class="border border-ink-faint p-4 hover:border-ink-muted transition-colors">
            <div class="flex justify-between items-start">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="text-ink text-sm font-medium truncate">{auto.name}</span>
                  <span class="text-xs px-1.5 py-0.5 border border-ink-faint {auto.enabled ? 'text-accent-success' : 'text-ink-muted'}">
                    {auto.enabled ? 'ON' : 'OFF'}
                  </span>
                </div>
                <p class="text-ink-muted text-xs mt-1 truncate">{auto.prompt}</p>
                <div class="flex gap-3 mt-2 text-xs text-ink-muted">
                  <span class="font-mono">{formatSchedule(auto.schedule)}</span>
                  <span>{auto.permissionLevel.replace('_', ' ')}</span>
                </div>
              </div>
              <div class="flex gap-1 ml-3">
                <button on:click={() => runNow(auto.id)} class="px-2 py-1 text-xs border border-ink-faint text-ink-muted hover:border-ink-muted hover:text-ink transition-colors" title="Run immediately">Run</button>
                <button on:click={() => toggleEnabled(auto)} class="px-2 py-1 text-xs border border-ink-faint text-ink-muted hover:border-ink-muted hover:text-ink transition-colors" title={auto.enabled ? 'Disable' : 'Enable'}>
                  {auto.enabled ? 'Disable' : 'Enable'}
                </button>
                <button on:click={() => dispatch('edit', auto)} class="px-2 py-1 text-xs border border-ink-faint text-ink-muted hover:border-ink-muted hover:text-ink transition-colors">Edit</button>
                <button on:click={() => remove(auto.id)} class="px-2 py-1 text-xs border border-ink-faint text-accent-danger hover:border-accent-danger transition-colors">Del</button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
