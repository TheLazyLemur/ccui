<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import SplitPane from './SplitPane.svelte';

  export let leftSize = 50;
  export let minLeft = 20;
  export let minRight = 20;

  const dispatch = createEventDispatcher();

  function handleResize(e: CustomEvent<number>) {
    dispatch('resize', e.detail);
  }
</script>

<SplitPane {leftSize} {minLeft} {minRight} on:resize={handleResize}>
  <div slot="left">Left content</div>
  <div slot="right">Right content</div>
</SplitPane>
