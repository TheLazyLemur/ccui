<script lang="ts">
  import { onMount } from 'svelte';
  import type { SessionMode } from './shared';
  import { SetMode } from '../../wailsjs/go/main/App';

  export let modes: SessionMode[] = [];
  export let currentModeId = '';

  let open = false;
  let container: HTMLDivElement;

  function selectMode(id: string) {
    SetMode(id);
    open = false;
  }

  function handleClickOutside(e: MouseEvent) {
    if (open && container && !container.contains(e.target as Node)) {
      open = false;
    }
  }

  onMount(() => {
    document.addEventListener('click', handleClickOutside);
    return () => document.removeEventListener('click', handleClickOutside);
  });

  $: currentMode = modes.find(m => m.id === currentModeId);
</script>

{#if modes.length > 0}
  <div bind:this={container} class="relative z-50">
    <button
      on:click|stopPropagation={() => open = !open}
      class="px-2 py-1 text-sm border border-ink-faint text-ink-medium hover:text-ink transition-colors flex items-center gap-1"
    >
      {currentMode?.name || 'Mode'}
      <span class="text-xs">▼</span>
    </button>
    {#if open}
      <div class="absolute top-full left-0 mt-1 bg-paper border border-ink-faint shadow-lg min-w-[150px]">
        {#each modes as mode}
          <button
            on:click|stopPropagation={() => selectMode(mode.id)}
            class="w-full px-3 py-2 text-left text-sm hover:bg-paper-dim transition-colors {mode.id === currentModeId ? 'bg-paper-dim text-ink' : 'text-ink-medium'}"
            title={mode.description}
          >
            {mode.name}
          </button>
        {/each}
      </div>
    {/if}
  </div>
{/if}
