<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EventsOn, EventsEmit } from '../../wailsjs/runtime/runtime';
  import { Terminal } from 'xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import 'xterm/css/xterm.css';

  export let terminalId = 'default';

  let container: HTMLDivElement;
  let term: Terminal;
  let fitAddon: FitAddon;
  let unsubscribes: (() => void)[] = [];
  let currentPtyId: string | null = null;
  let resizeObserver: ResizeObserver | null = null;

  function startPty(id: string) {
    if (currentPtyId === id) return;

    // Stop previous PTY if any
    if (currentPtyId) {
      EventsEmit('terminal:stop', { id: currentPtyId });
    }

    // Clear previous output subscription
    unsubscribes.forEach(fn => fn());
    unsubscribes = [];

    currentPtyId = id;

    // Subscribe to output for new ID
    unsubscribes.push(
      EventsOn(`terminal:${id}:output`, (data: string) => {
        term?.write(data);
      })
    );

    // Clear terminal and start new PTY
    term?.clear();
    EventsEmit('terminal:start', {
      id,
      cols: term?.cols || 80,
      rows: term?.rows || 24,
    });
  }

  onMount(() => {
    term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace',
      theme: {
        background: '#1a1a1a',
        foreground: '#e0e0e0',
        cursor: '#e0e0e0',
        cursorAccent: '#1a1a1a',
        selectionBackground: '#444',
      },
      allowProposedApi: true,
    });

    fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(container);
    fitAddon.fit();
    term.focus();

    // Send input to Go backend
    term.onData((data) => {
      if (currentPtyId) {
        EventsEmit('terminal:input', { id: currentPtyId, data });
      }
    });

    // Handle resize
    let lastCols = 0, lastRows = 0;
    resizeObserver = new ResizeObserver(() => {
      if (container.clientHeight === 0) return; // Hidden
      fitAddon.fit();
      if (currentPtyId && (term.cols !== lastCols || term.rows !== lastRows)) {
        lastCols = term.cols;
        lastRows = term.rows;
        EventsEmit('terminal:resize', {
          id: currentPtyId,
          cols: term.cols,
          rows: term.rows,
        });
      }
    });
    resizeObserver.observe(container);

    // Start PTY
    startPty(terminalId);
  });

  // React to terminalId changes
  $: if (term && terminalId) {
    startPty(terminalId);
  }

  onDestroy(() => {
    unsubscribes.forEach((fn) => fn());
    if (currentPtyId) {
      EventsEmit('terminal:stop', { id: currentPtyId });
    }
    resizeObserver?.disconnect();
    term?.dispose();
  });
</script>

<div bind:this={container} class="terminal-container" on:click={() => term?.focus()}></div>

<style>
  .terminal-container {
    width: 100%;
    height: 100%;
    background: #1a1a1a;
  }
  .terminal-container :global(.xterm) {
    padding: 8px;
    height: 100%;
  }
</style>
