<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let leftSize = 50;
  export let minLeft = 20;
  export let minRight = 20;

  const dispatch = createEventDispatcher<{ resize: number }>();

  let containerEl: HTMLElement;
  let isDragging = false;
  let currentSize = clamp(leftSize);

  // Check if right slot has content using $$slots
  $: hasRightContent = $$slots.right;

  function clamp(size: number): number {
    return Math.min(Math.max(size, minLeft), 100 - minRight);
  }

  // Update when props change (only when not dragging)
  $: if (!isDragging) currentSize = clamp(leftSize);

  function handleMouseDown(e: MouseEvent) {
    e.preventDefault();
    isDragging = true;
  }

  function handleMouseMove(e: MouseEvent) {
    if (!isDragging || !containerEl) return;
    const rect = containerEl.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const percent = (x / rect.width) * 100;
    currentSize = clamp(percent);
  }

  function handleMouseUp() {
    if (isDragging) {
      isDragging = false;
      dispatch('resize', currentSize);
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    const step = 5;
    if (e.key === 'ArrowLeft') {
      e.preventDefault();
      currentSize = clamp(currentSize - step);
      dispatch('resize', currentSize);
    } else if (e.key === 'ArrowRight') {
      e.preventDefault();
      currentSize = clamp(currentSize + step);
      dispatch('resize', currentSize);
    }
  }
</script>

<svelte:window on:mousemove={handleMouseMove} on:mouseup={handleMouseUp} />

<div
  bind:this={containerEl}
  data-testid="split-pane"
  class="flex h-full w-full"
>
  <div
    data-testid="left-panel"
    class="h-full overflow-hidden"
    style="width: {hasRightContent ? currentSize + '%' : '100%'}"
  >
    <slot name="left" />
  </div>

  {#if hasRightContent}
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div
      data-testid="drag-handle"
      class="handle flex-shrink-0 h-full"
      class:dragging={isDragging}
      on:mousedown={handleMouseDown}
      on:keydown={handleKeyDown}
      role="separator"
      aria-label="Resize panels"
      aria-orientation="vertical"
      aria-valuenow={currentSize}
      aria-valuemin={minLeft}
      aria-valuemax={100 - minRight}
      tabindex="0"
    />
  {/if}

  <div
    data-testid="right-panel"
    class="h-full overflow-hidden"
    class:flex-1={hasRightContent}
    class:hidden={!hasRightContent}
  >
    <slot name="right" />
  </div>
</div>

<style>
  .handle {
    width: 4px;
    cursor: col-resize;
    background: var(--color-ink-faint);
    transition: background 0.15s ease;
  }

  .handle:hover,
  .handle.dragging {
    background: var(--color-ink-muted);
  }

  .handle:focus {
    outline: 2px solid var(--color-ink-muted);
    outline-offset: -2px;
  }
</style>
