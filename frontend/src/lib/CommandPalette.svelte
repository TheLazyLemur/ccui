<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';

  type OptionType = 'chat' | 'review' | 'terminal' | null;

  interface Option {
    id: string;
    label: string;
    value: OptionType;
  }

  const options: Option[] = [
    { id: 'chat', label: 'Chat', value: 'chat' },
    { id: 'review', label: 'Review', value: 'review' },
    { id: 'terminal', label: 'Terminal', value: 'terminal' },
    { id: 'close', label: 'Close Panel', value: null },
  ];

  const dispatch = createEventDispatcher<{ select: OptionType; close: void }>();

  let query = '';
  let selectedIndex = 0;
  let inputEl: HTMLInputElement;

  $: filteredOptions = options.filter(opt =>
    opt.label.toLowerCase().includes(query.toLowerCase())
  );

  $: if (filteredOptions.length > 0 && selectedIndex >= filteredOptions.length) {
    selectedIndex = 0;
  }

  // Reset selection when query changes
  $: query, selectedIndex = 0;

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      selectedIndex = (selectedIndex + 1) % filteredOptions.length;
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      selectedIndex = (selectedIndex - 1 + filteredOptions.length) % filteredOptions.length;
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (filteredOptions.length > 0) {
        dispatch('select', filteredOptions[selectedIndex].value);
      }
    } else if (e.key === 'Escape') {
      e.preventDefault();
      dispatch('close');
    }
  }

  function handleOverlayClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      dispatch('close');
    }
  }

  function handleOptionClick(option: Option) {
    dispatch('select', option.value);
  }

  function handleOptionHover(index: number) {
    selectedIndex = index;
  }

  onMount(() => {
    inputEl?.focus();
  });
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div
  data-testid="palette-overlay"
  class="absolute inset-0 bg-paper/90 z-50 flex items-center justify-center"
  on:click={handleOverlayClick}
>
  <div
    data-testid="palette-modal"
    class="bg-paper border border-ink-faint w-full max-w-md mx-6 animate-slide-up"
  >
    <div class="px-4 py-3 border-b border-ink-faint">
      <!-- svelte-ignore a11y-autofocus -->
      <input
        bind:this={inputEl}
        bind:value={query}
        on:keydown={handleKeyDown}
        data-testid="palette-input"
        type="text"
        placeholder="Search..."
        autofocus
        class="w-full bg-transparent text-ink text-[15px] placeholder-ink-muted focus:outline-none"
      />
    </div>
    <div class="py-2">
      {#each filteredOptions as option, i (option.id)}
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div
          data-testid="palette-option-{option.id}"
          class="px-4 py-2 cursor-pointer transition-colors {i === selectedIndex ? 'selected bg-paper-dim' : 'hover:bg-paper-dim/50'}"
          class:selected={i === selectedIndex}
          on:click={() => handleOptionClick(option)}
          on:mouseenter={() => handleOptionHover(i)}
        >
          <span class="text-ink text-sm">{option.label}</span>
        </div>
      {/each}
    </div>
  </div>
</div>
