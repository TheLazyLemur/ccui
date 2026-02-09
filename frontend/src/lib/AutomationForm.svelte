<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { Automation, PermissionLevel } from './shared';
  import { CreateAutomation, UpdateAutomation, BrowseDirectory } from '../../wailsjs/go/main/App';

  export let automation: Automation | null = null;

  const dispatch = createEventDispatcher<{ saved: Automation; cancel: void }>();

  let name = automation?.name ?? '';
  let prompt = automation?.prompt ?? '';
  let schedule = automation?.schedule ?? '';
  let projectDir = automation?.projectDir ?? '';
  let backendType = automation?.backendType ?? 'anthropic';
  let permissionLevel: PermissionLevel = automation?.permissionLevel ?? 'read_only';
  let enabled = automation?.enabled ?? true;
  let useWorktree = automation?.useWorktree ?? false;
  let saving = false;
  let error = '';

  async function browseDir() {
    try {
      const dir = await BrowseDirectory();
      if (dir) projectDir = dir;
    } catch (e) {
      error = String(e);
    }
  }

  async function save() {
    if (!name.trim() || !prompt.trim()) {
      error = 'Name and prompt are required';
      return;
    }
    saving = true;
    error = '';
    try {
      const data = { name, prompt, schedule, projectDir, backendType, permissionLevel, enabled, useWorktree } as any;
      let result: Automation;
      if (automation) {
        data.id = automation.id;
        result = await UpdateAutomation(data);
      } else {
        result = await CreateAutomation(data);
      }
      dispatch('saved', result);
    } catch (e) {
      error = String(e);
    } finally {
      saving = false;
    }
  }
</script>

<div class="h-full overflow-y-auto">
  <div class="p-6 space-y-5 max-w-xl">
    <h2 class="text-ink font-medium">{automation ? 'Edit' : 'New'} Automation</h2>

    {#if error}
      <div class="px-4 py-2 border border-accent-danger text-accent-danger text-sm">{error}</div>
    {/if}

    <div class="space-y-1">
      <label for="auto-name" class="text-xs text-ink-muted uppercase tracking-wide">Name</label>
      <input id="auto-name" bind:value={name} placeholder="Daily code review" class="w-full px-3 py-2 bg-paper border border-ink-faint text-ink text-sm focus:outline-none focus:border-ink-muted" />
    </div>

    <div class="space-y-1">
      <label for="auto-prompt" class="text-xs text-ink-muted uppercase tracking-wide">Prompt</label>
      <textarea id="auto-prompt" bind:value={prompt} rows="4" placeholder="Review recent changes for bugs and security issues..." class="w-full px-3 py-2 bg-paper border border-ink-faint text-ink text-sm resize-none focus:outline-none focus:border-ink-muted"></textarea>
    </div>

    <div class="space-y-1">
      <label for="auto-schedule" class="text-xs text-ink-muted uppercase tracking-wide">Schedule (cron)</label>
      <input id="auto-schedule" bind:value={schedule} placeholder="0 9 * * * (daily at 9am)" class="w-full px-3 py-2 bg-paper border border-ink-faint text-ink text-sm font-mono focus:outline-none focus:border-ink-muted" />
      <p class="text-xs text-ink-muted">Leave empty for manual-only execution</p>
    </div>

    <div class="space-y-1">
      <label for="auto-dir" class="text-xs text-ink-muted uppercase tracking-wide">Project Directory</label>
      <div class="flex gap-2">
        <input id="auto-dir" bind:value={projectDir} placeholder="/path/to/project" class="flex-1 px-3 py-2 bg-paper border border-ink-faint text-ink text-sm font-mono focus:outline-none focus:border-ink-muted" />
        <button on:click={browseDir} class="px-3 py-2 border border-ink-faint text-ink-medium text-sm hover:border-ink-muted hover:text-ink transition-colors">Browse</button>
      </div>
    </div>

    <div class="space-y-1">
      <label for="auto-backend" class="text-xs text-ink-muted uppercase tracking-wide">Backend</label>
      <select id="auto-backend" bind:value={backendType} class="w-full px-3 py-2 bg-paper border border-ink-faint text-ink text-sm focus:outline-none focus:border-ink-muted">
        <option value="anthropic">Anthropic API</option>
      </select>
    </div>

    <div class="space-y-1">
      <label for="auto-perm" class="text-xs text-ink-muted uppercase tracking-wide">Permission Level</label>
      <select id="auto-perm" bind:value={permissionLevel} class="w-full px-3 py-2 bg-paper border border-ink-faint text-ink text-sm focus:outline-none focus:border-ink-muted">
        <option value="read_only">Read Only</option>
        <option value="workspace_write">Workspace Write</option>
        <option value="full_access">Full Access</option>
      </select>
      <p class="text-xs text-ink-muted">
        {#if permissionLevel === 'read_only'}Read, search, and browse only
        {:else if permissionLevel === 'workspace_write'}Can also write and edit files
        {:else}All tools auto-approved{/if}
      </p>
    </div>

    <label class="flex items-center gap-2 cursor-pointer">
      <input type="checkbox" bind:checked={enabled} class="check-ink" />
      <span class="text-sm text-ink">Enabled</span>
    </label>

    <label class="flex items-center gap-2 cursor-pointer">
      <input type="checkbox" bind:checked={useWorktree} class="check-ink" />
      <span class="text-sm text-ink">Use git worktree</span>
    </label>
    {#if useWorktree}
      <p class="text-xs text-ink-muted -mt-3 ml-6">Runs in an isolated worktree copy, won't affect your working directory</p>
    {/if}

    <div class="flex gap-3 pt-2">
      <button on:click={save} disabled={saving} class="px-5 py-2 bg-ink text-paper text-sm hover:bg-ink-medium transition-colors disabled:opacity-40">
        {saving ? 'Saving...' : automation ? 'Update' : 'Create'}
      </button>
      <button on:click={() => dispatch('cancel')} class="px-5 py-2 border border-ink-faint text-ink-medium text-sm hover:border-ink-muted hover:text-ink transition-colors">Cancel</button>
    </div>
  </div>
</div>
