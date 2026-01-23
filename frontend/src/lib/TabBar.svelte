<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let activeTab: 'chat' | 'review' = 'chat';
  export let changeCount = 0;

  const dispatch = createEventDispatcher<{ tabChange: 'chat' | 'review' }>();
</script>

<div class="flex gap-1">
  <button
    on:click={() => dispatch('tabChange', 'chat')}
    class="px-3 py-1.5 text-sm transition-colors {activeTab === 'chat' ? 'bg-ink text-paper' : 'text-ink-medium hover:text-ink border border-ink-faint'}"
  >Chat</button>
  <button
    on:click={() => dispatch('tabChange', 'review')}
    class="px-3 py-1.5 text-sm transition-colors {activeTab === 'review' ? 'bg-ink text-paper' : 'text-ink-medium hover:text-ink border border-ink-faint'}"
  >Review{#if changeCount > 0}<span class="ml-1.5 text-xs">({changeCount})</span>{/if}</button>
</div>
