<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import type { SessionInfo } from './shared';
  import { CreateSession, SwitchSession, CloseSession } from '../../wailsjs/go/main/App';

  export let sessions: SessionInfo[] = [];
  export let activeSessionId = '';

  let open = false;
  let container: HTMLDivElement;
  let newSessionName = '';
  let showNewInput = false;

  const dispatch = createEventDispatcher<{ sessionChange: string }>();

  async function selectSession(id: string) {
    await SwitchSession(id);
    dispatch('sessionChange', id);
    open = false;
  }

  async function createNewSession() {
    if (!newSessionName.trim()) return;
    const id = await CreateSession(newSessionName.trim());
    dispatch('sessionChange', id);
    newSessionName = '';
    showNewInput = false;
    open = false;
  }

  async function closeSession(id: string) {
    await CloseSession(id);
  }

  function handleClickOutside(e: MouseEvent) {
    if (open && container && !container.contains(e.target as Node)) {
      open = false;
      showNewInput = false;
    }
  }

  onMount(() => {
    document.addEventListener('click', handleClickOutside);
    return () => document.removeEventListener('click', handleClickOutside);
  });

  $: currentSession = sessions.find(s => s.id === activeSessionId);
</script>

<div bind:this={container} class="relative z-50">
  <button
    on:click|stopPropagation={() => open = !open}
    class="px-2 py-1 text-sm border border-ink-faint text-ink-medium hover:text-ink transition-colors flex items-center gap-1"
  >
    {currentSession?.name || 'Session'}
    <span class="text-xs">▼</span>
  </button>
  {#if open}
    <div class="absolute top-full left-0 mt-1 bg-paper border border-ink-faint shadow-lg min-w-[180px]">
      {#each sessions as session}
        <button
          on:click|stopPropagation={() => selectSession(session.id)}
          class="w-full px-3 py-2 text-left text-sm hover:bg-paper-dim transition-colors flex items-center justify-between {session.id === activeSessionId ? 'bg-paper-dim text-ink' : 'text-ink-medium'}"
        >
          <span class="truncate flex-1">{session.name}</span>
          {#if sessions.length > 1}
            <button
              on:click|stopPropagation={() => closeSession(session.id)}
              class="ml-2 text-ink-muted hover:text-accent-danger text-xs"
              title="Close session"
            >×</button>
          {/if}
        </button>
      {/each}
      <div class="border-t border-ink-faint">
        {#if showNewInput}
          <div class="p-2 flex gap-2">
            <input
              bind:value={newSessionName}
              on:keydown={(e) => e.key === 'Enter' && createNewSession()}
              placeholder="Session name"
              class="flex-1 px-2 py-1 text-sm border border-ink-faint bg-paper text-ink focus:outline-none"
              autofocus
            />
            <button
              on:click={createNewSession}
              class="px-2 py-1 text-sm bg-ink text-paper hover:bg-ink-medium"
            >+</button>
          </div>
        {:else}
          <button
            on:click|stopPropagation={() => showNewInput = true}
            class="w-full px-3 py-2 text-left text-sm text-ink-muted hover:text-ink hover:bg-paper-dim transition-colors"
          >+ New Session</button>
        {/if}
      </div>
    </div>
  {/if}
</div>
