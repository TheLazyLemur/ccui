<script lang="ts">
  import type { Automation, AutomationRun } from './shared';
  import AutomationList from './AutomationList.svelte';
  import AutomationForm from './AutomationForm.svelte';
  import AutomationTriage from './AutomationTriage.svelte';
  import AutomationHistory from './AutomationHistory.svelte';
  import AutomationRunDetail from './AutomationRunDetail.svelte';

  export let triageCount = 0;

  type Tab = 'list' | 'triage' | 'history';
  type View = 'tab' | 'create' | 'edit' | 'detail';

  let activeTab: Tab = 'list';
  let view: View = 'tab';
  let editTarget: Automation | null = null;
  let detailRun: AutomationRun | null = null;
  let detailAutoName = '';
  let listRef: AutomationList;
  let triageRef: AutomationTriage;

  function handleCreate() {
    editTarget = null;
    view = 'create';
  }

  function handleEdit(e: CustomEvent<Automation>) {
    editTarget = e.detail;
    view = 'edit';
  }

  function handleSaved() {
    view = 'tab';
    editTarget = null;
    listRef?.refresh();
  }

  function handleCancel() {
    view = 'tab';
    editTarget = null;
  }

  function handleViewRun(e: CustomEvent<{ run: AutomationRun; automationName: string }>) {
    detailRun = e.detail.run;
    detailAutoName = e.detail.automationName;
    view = 'detail';
  }

  function handleDetailBack() {
    view = 'tab';
    detailRun = null;
    triageRef?.refresh();
  }

  function switchTab(tab: Tab) {
    activeTab = tab;
    view = 'tab';
  }
</script>

<div class="h-full flex flex-col">
  {#if view === 'tab'}
    <!-- Tab bar -->
    <div class="flex border-b border-ink-faint">
      <button
        on:click={() => switchTab('list')}
        class="px-4 py-2 text-xs uppercase tracking-wide transition-colors {activeTab === 'list' ? 'text-ink border-b-2 border-ink' : 'text-ink-muted hover:text-ink'}"
      >
        Automations
      </button>
      <button
        on:click={() => switchTab('triage')}
        class="px-4 py-2 text-xs uppercase tracking-wide transition-colors relative {activeTab === 'triage' ? 'text-ink border-b-2 border-ink' : 'text-ink-muted hover:text-ink'}"
      >
        Triage
        {#if triageCount > 0}
          <span class="absolute -top-0.5 -right-0.5 w-4 h-4 bg-accent-warning text-paper text-[10px] flex items-center justify-center rounded-full">{triageCount}</span>
        {/if}
      </button>
      <button
        on:click={() => switchTab('history')}
        class="px-4 py-2 text-xs uppercase tracking-wide transition-colors {activeTab === 'history' ? 'text-ink border-b-2 border-ink' : 'text-ink-muted hover:text-ink'}"
      >
        History
      </button>
    </div>

    <!-- Tab content -->
    <div class="flex-1 overflow-hidden">
      {#if activeTab === 'list'}
        <AutomationList bind:this={listRef} on:create={handleCreate} on:edit={handleEdit} />
      {:else if activeTab === 'triage'}
        <AutomationTriage bind:this={triageRef} on:viewRun={handleViewRun} />
      {:else if activeTab === 'history'}
        <AutomationHistory on:viewRun={handleViewRun} />
      {/if}
    </div>
  {:else if view === 'create'}
    <AutomationForm automation={null} on:saved={handleSaved} on:cancel={handleCancel} />
  {:else if view === 'edit'}
    <AutomationForm automation={editTarget} on:saved={handleSaved} on:cancel={handleCancel} />
  {:else if view === 'detail' && detailRun}
    <AutomationRunDetail run={detailRun} automationName={detailAutoName} on:back={handleDetailBack} />
  {/if}
</div>
