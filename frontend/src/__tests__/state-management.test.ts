import { describe, it, expect, beforeEach, vi } from 'vitest';

// Types matching shared.ts
interface ToolCall {
  id: string;
  title: string;
  kind: string;
  status: string;
  toolName?: string;
  parentId?: string;
}

interface Message {
  id: number;
  text: string;
  sender: 'user' | 'bot' | 'tool';
  toolState?: ToolCall;
}

interface FileChange {
  filePath: string;
  originalContent: string;
  currentContent: string;
}

interface PlanEntry {
  content: string;
  priority: 'high' | 'medium' | 'low';
  status: 'pending' | 'in_progress' | 'completed';
}

interface SessionMode {
  id: string;
  name: string;
}

interface SessionState {
  messages: Message[];
  fileChanges: FileChange[];
  planEntries: PlanEntry[];
  currentModeId: string;
  currentChunk: string;
  currentThought: string;
  availableModes: SessionMode[];
  isLoading: boolean;
}

// Mock event system
type EventCallback = (...args: unknown[]) => void;
const eventListeners = new Map<string, Set<EventCallback>>();

const mockEventsOn = vi.fn((eventName: string, callback: EventCallback) => {
  if (!eventListeners.has(eventName)) {
    eventListeners.set(eventName, new Set());
  }
  eventListeners.get(eventName)!.add(callback);
  return () => eventListeners.get(eventName)?.delete(callback);
});

const mockEventsEmit = vi.fn((eventName: string, ...data: unknown[]) => {
  eventListeners.get(eventName)?.forEach(cb => cb(...data));
});

// Session manager with full state (mirrors App.svelte more closely)
class FullSessionManager {
  private sessionStates = new Map<string, SessionState>();
  private subscribedSessions = new Set<string>();
  private activeSessionId = '';

  // Reactive state
  messages: Message[] = [];
  currentChunk = '';
  currentThought = '';
  isLoading = false;
  fileChanges: FileChange[] = [];
  planEntries: PlanEntry[] = [];
  availableModes: SessionMode[] = [];
  currentModeId = '';

  private getOrCreateState(id: string): SessionState {
    if (!this.sessionStates.has(id)) {
      this.sessionStates.set(id, {
        messages: [],
        fileChanges: [],
        planEntries: [],
        currentModeId: '',
        currentChunk: '',
        currentThought: '',
        availableModes: [],
        isLoading: false
      });
    }
    return this.sessionStates.get(id)!;
  }

  private loadState(id: string) {
    const state = this.getOrCreateState(id);
    this.messages = [...state.messages];
    this.currentChunk = state.currentChunk;
    this.currentThought = state.currentThought;
    this.isLoading = state.isLoading;
    this.fileChanges = [...state.fileChanges];
    this.planEntries = [...state.planEntries];
    this.availableModes = [...state.availableModes];
    this.currentModeId = state.currentModeId;
  }

  subscribeToSession(sessionId: string) {
    if (this.subscribedSessions.has(sessionId)) return;
    this.subscribedSessions.add(sessionId);

    const state = this.getOrCreateState(sessionId);
    const syncIfActive = () => {
      if (sessionId === this.activeSessionId) this.loadState(sessionId);
    };

    mockEventsOn(`session:${sessionId}:chat_chunk`, (text: string) => {
      state.currentChunk += text;
      syncIfActive();
    });

    mockEventsOn(`session:${sessionId}:chat_thought`, (text: string) => {
      state.currentThought += text;
      syncIfActive();
    });

    mockEventsOn(`session:${sessionId}:tool_state`, (toolState: ToolCall) => {
      const idx = state.messages.findIndex(m => m.toolState?.id === toolState.id);
      if (idx >= 0) {
        state.messages[idx].toolState = toolState;
      } else {
        const newId = state.messages.length > 0 ? Math.max(...state.messages.map(m => m.id)) + 1 : 1;
        state.messages.push({ id: newId, text: '', sender: 'tool', toolState });
      }
      syncIfActive();
    });

    mockEventsOn(`session:${sessionId}:prompt_complete`, () => {
      if (state.currentChunk) {
        const newId = state.messages.length > 0 ? Math.max(...state.messages.map(m => m.id)) + 1 : 1;
        state.messages.push({ id: newId, text: state.currentChunk, sender: 'bot' });
        state.currentChunk = '';
      }
      state.currentThought = '';
      state.isLoading = false;
      syncIfActive();
    });

    mockEventsOn(`session:${sessionId}:error`, (err: string) => {
      const newId = state.messages.length > 0 ? Math.max(...state.messages.map(m => m.id)) + 1 : 1;
      state.messages.push({ id: newId, text: `Error: ${err}`, sender: 'bot' });
      state.isLoading = false;
      syncIfActive();
    });

    mockEventsOn(`session:${sessionId}:file_changes_updated`, (changes: FileChange[]) => {
      state.fileChanges = changes;
      syncIfActive();
    });

    mockEventsOn(`session:${sessionId}:modes_available`, (modes: SessionMode[]) => {
      state.availableModes = modes;
      syncIfActive();
    });

    mockEventsOn(`session:${sessionId}:mode_changed`, (modeId: string) => {
      state.currentModeId = modeId;
      syncIfActive();
    });

    mockEventsOn(`session:${sessionId}:plan_update`, (entries: PlanEntry[]) => {
      state.planEntries = entries;
      syncIfActive();
    });
  }

  switchSession(newSessionId: string) {
    if (newSessionId === this.activeSessionId) return;
    this.activeSessionId = newSessionId;
    this.subscribeToSession(newSessionId);
    this.loadState(newSessionId);
  }

  sendMessage(text: string): boolean {
    const trimmed = text.trim();
    if (!trimmed || this.isLoading || !this.activeSessionId) return false;
    const state = this.getOrCreateState(this.activeSessionId);
    if (state.currentChunk) {
      const newId = state.messages.length > 0 ? Math.max(...state.messages.map(m => m.id)) + 1 : 1;
      state.messages.push({ id: newId, text: state.currentChunk, sender: 'bot' });
      state.currentChunk = '';
    }
    const msgId = state.messages.length > 0 ? Math.max(...state.messages.map(m => m.id)) + 1 : 1;
    state.messages.push({ id: msgId, text: trimmed, sender: 'user' });
    state.isLoading = true;
    this.loadState(this.activeSessionId);
    return true;
  }

  cancelRequest() {
    if (!this.activeSessionId) return;
    const state = this.getOrCreateState(this.activeSessionId);
    state.isLoading = false;
    this.loadState(this.activeSessionId);
  }

  getToolMessages(): Message[] {
    return this.messages.filter(m => m.sender === 'tool');
  }

  getChildTools(parentId: string): ToolCall[] {
    return this.messages
      .filter(m => m.toolState?.parentId === parentId)
      .map(m => m.toolState!);
  }
}

describe('Tool State Management', () => {
  let manager: FullSessionManager;

  beforeEach(() => {
    eventListeners.clear();
    mockEventsOn.mockClear();
    mockEventsEmit.mockClear();
    manager = new FullSessionManager();
    manager.switchSession('session-1');
  });

  it('adds new tool state as message', () => {
    const tool: ToolCall = { id: 't1', title: 'Read file', kind: 'read', status: 'running' };
    mockEventsEmit('session:session-1:tool_state', tool);

    expect(manager.getToolMessages()).toHaveLength(1);
    expect(manager.getToolMessages()[0].toolState).toEqual(tool);
  });

  it('updates existing tool state by id', () => {
    const tool1: ToolCall = { id: 't1', title: 'Read file', kind: 'read', status: 'running' };
    const tool2: ToolCall = { id: 't1', title: 'Read file', kind: 'read', status: 'completed' };

    mockEventsEmit('session:session-1:tool_state', tool1);
    mockEventsEmit('session:session-1:tool_state', tool2);

    expect(manager.getToolMessages()).toHaveLength(1);
    expect(manager.getToolMessages()[0].toolState?.status).toBe('completed');
  });

  it('handles multiple tools with different ids', () => {
    const tools: ToolCall[] = [
      { id: 't1', title: 'Read', kind: 'read', status: 'completed' },
      { id: 't2', title: 'Write', kind: 'write', status: 'running' },
      { id: 't3', title: 'Bash', kind: 'bash', status: 'pending' }
    ];

    tools.forEach(t => mockEventsEmit('session:session-1:tool_state', t));

    expect(manager.getToolMessages()).toHaveLength(3);
  });

  it('handles subagent tools with parentId', () => {
    const parent: ToolCall = { id: 'p1', title: 'Task', kind: 'task', status: 'running', toolName: 'Task' };
    const child1: ToolCall = { id: 'c1', title: 'Read', kind: 'read', status: 'completed', parentId: 'p1' };
    const child2: ToolCall = { id: 'c2', title: 'Write', kind: 'write', status: 'running', parentId: 'p1' };

    mockEventsEmit('session:session-1:tool_state', parent);
    mockEventsEmit('session:session-1:tool_state', child1);
    mockEventsEmit('session:session-1:tool_state', child2);

    const children = manager.getChildTools('p1');
    expect(children).toHaveLength(2);
    expect(children.map(c => c.id)).toEqual(['c1', 'c2']);
  });

  it('tool status transitions: pending -> running -> completed', () => {
    const statuses = ['pending', 'running', 'completed'];

    statuses.forEach(status => {
      mockEventsEmit('session:session-1:tool_state', { id: 't1', title: 'Test', kind: 'test', status });
    });

    expect(manager.getToolMessages()[0].toolState?.status).toBe('completed');
  });

  it('tools interleaved with chat messages maintain order', () => {
    // Note: tool_state creates message immediately, chat_chunk waits for prompt_complete
    mockEventsEmit('session:session-1:chat_chunk', 'Hello');
    mockEventsEmit('session:session-1:tool_state', { id: 't1', title: 'Read', kind: 'read', status: 'running' });
    mockEventsEmit('session:session-1:prompt_complete');
    mockEventsEmit('session:session-1:tool_state', { id: 't1', title: 'Read', kind: 'read', status: 'completed' });
    mockEventsEmit('session:session-1:chat_chunk', 'Done');
    mockEventsEmit('session:session-1:prompt_complete');

    // Tool appears first (created on tool_state), then Hello (flushed on prompt_complete)
    expect(manager.messages).toHaveLength(3);
    expect(manager.messages[0].sender).toBe('tool');
    expect(manager.messages[1].text).toBe('Hello');
    expect(manager.messages[2].text).toBe('Done');
  });
});

describe('Message Input Validation', () => {
  let manager: FullSessionManager;

  beforeEach(() => {
    eventListeners.clear();
    manager = new FullSessionManager();
    manager.switchSession('session-1');
  });

  it('rejects empty message', () => {
    expect(manager.sendMessage('')).toBe(false);
    expect(manager.messages).toHaveLength(0);
  });

  it('rejects whitespace-only message', () => {
    expect(manager.sendMessage('   ')).toBe(false);
    expect(manager.sendMessage('\t\n')).toBe(false);
    expect(manager.messages).toHaveLength(0);
  });

  it('rejects message while loading', () => {
    manager.sendMessage('First');
    expect(manager.isLoading).toBe(true);

    expect(manager.sendMessage('Second')).toBe(false);
    expect(manager.messages.filter(m => m.sender === 'user')).toHaveLength(1);
  });

  it('accepts message after loading completes', () => {
    manager.sendMessage('First');
    mockEventsEmit('session:session-1:chat_chunk', 'Response');
    mockEventsEmit('session:session-1:prompt_complete');

    expect(manager.sendMessage('Second')).toBe(true);
    expect(manager.messages.filter(m => m.sender === 'user')).toHaveLength(2);
  });

  it('trims message but preserves internal whitespace', () => {
    manager.sendMessage('  hello   world  ');
    expect(manager.messages[0].text).toBe('hello   world');
  });

  it('handles very long message', () => {
    const longMsg = 'x'.repeat(10000);
    expect(manager.sendMessage(longMsg)).toBe(true);
    expect(manager.messages[0].text).toBe(longMsg);
  });

  it('handles message with newlines', () => {
    const multiline = 'line1\nline2\nline3';
    manager.sendMessage(multiline);
    expect(manager.messages[0].text).toBe(multiline);
  });

  it('handles unicode and emoji', () => {
    const unicode = '你好 🎉 مرحبا';
    manager.sendMessage(unicode);
    expect(manager.messages[0].text).toBe(unicode);
  });
});

describe('Loading State', () => {
  let manager: FullSessionManager;

  beforeEach(() => {
    eventListeners.clear();
    manager = new FullSessionManager();
    manager.switchSession('session-1');
  });

  it('sets loading true on send', () => {
    expect(manager.isLoading).toBe(false);
    manager.sendMessage('Test');
    expect(manager.isLoading).toBe(true);
  });

  it('clears loading on prompt_complete', () => {
    manager.sendMessage('Test');
    mockEventsEmit('session:session-1:prompt_complete');
    expect(manager.isLoading).toBe(false);
  });

  it('clears loading on error', () => {
    manager.sendMessage('Test');
    mockEventsEmit('session:session-1:error', 'Something went wrong');
    expect(manager.isLoading).toBe(false);
  });

  it('cancel clears loading', () => {
    manager.sendMessage('Test');
    manager.cancelRequest();
    expect(manager.isLoading).toBe(false);
  });

  it('loading state is per-session', () => {
    manager.sendMessage('Test');
    expect(manager.isLoading).toBe(true);

    manager.switchSession('session-2');
    expect(manager.isLoading).toBe(false);

    manager.switchSession('session-1');
    expect(manager.isLoading).toBe(true);
  });
});

describe('Error Handling', () => {
  let manager: FullSessionManager;

  beforeEach(() => {
    eventListeners.clear();
    manager = new FullSessionManager();
    manager.switchSession('session-1');
  });

  it('error adds message with Error prefix', () => {
    mockEventsEmit('session:session-1:error', 'Connection failed');
    expect(manager.messages[0].text).toBe('Error: Connection failed');
    expect(manager.messages[0].sender).toBe('bot');
  });

  it('error clears loading state', () => {
    manager.sendMessage('Test');
    mockEventsEmit('session:session-1:error', 'Failed');
    expect(manager.isLoading).toBe(false);
  });

  it('multiple errors create multiple messages', () => {
    mockEventsEmit('session:session-1:error', 'Error 1');
    mockEventsEmit('session:session-1:error', 'Error 2');
    expect(manager.messages).toHaveLength(2);
  });

  it('error during streaming preserves partial chunk', () => {
    mockEventsEmit('session:session-1:chat_chunk', 'Partial');
    mockEventsEmit('session:session-1:error', 'Interrupted');

    // Chunk not finalized, but error added
    expect(manager.currentChunk).toBe('Partial');
    expect(manager.messages[0].text).toBe('Error: Interrupted');
  });
});

describe('Thought/Thinking State', () => {
  let manager: FullSessionManager;

  beforeEach(() => {
    eventListeners.clear();
    manager = new FullSessionManager();
    manager.switchSession('session-1');
  });

  it('accumulates thought chunks', () => {
    mockEventsEmit('session:session-1:chat_thought', 'Thinking');
    mockEventsEmit('session:session-1:chat_thought', ' about');
    mockEventsEmit('session:session-1:chat_thought', ' this');
    expect(manager.currentThought).toBe('Thinking about this');
  });

  it('clears thought on prompt_complete', () => {
    mockEventsEmit('session:session-1:chat_thought', 'Deep thought');
    mockEventsEmit('session:session-1:prompt_complete');
    expect(manager.currentThought).toBe('');
  });

  it('thought and chunk can accumulate simultaneously', () => {
    mockEventsEmit('session:session-1:chat_thought', 'Thinking...');
    mockEventsEmit('session:session-1:chat_chunk', 'Response');
    expect(manager.currentThought).toBe('Thinking...');
    expect(manager.currentChunk).toBe('Response');
  });

  it('thought is per-session', () => {
    mockEventsEmit('session:session-1:chat_thought', 'S1 thought');
    manager.switchSession('session-2');
    mockEventsEmit('session:session-2:chat_thought', 'S2 thought');

    expect(manager.currentThought).toBe('S2 thought');
    manager.switchSession('session-1');
    expect(manager.currentThought).toBe('S1 thought');
  });
});

describe('File Changes', () => {
  let manager: FullSessionManager;

  beforeEach(() => {
    eventListeners.clear();
    manager = new FullSessionManager();
    manager.switchSession('session-1');
  });

  it('updates file changes', () => {
    const changes: FileChange[] = [
      { filePath: '/a.ts', originalContent: 'old', currentContent: 'new' }
    ];
    mockEventsEmit('session:session-1:file_changes_updated', changes);
    expect(manager.fileChanges).toEqual(changes);
  });

  it('replaces file changes entirely', () => {
    mockEventsEmit('session:session-1:file_changes_updated', [{ filePath: '/a.ts', originalContent: '', currentContent: '' }]);
    mockEventsEmit('session:session-1:file_changes_updated', [{ filePath: '/b.ts', originalContent: '', currentContent: '' }]);
    expect(manager.fileChanges).toHaveLength(1);
    expect(manager.fileChanges[0].filePath).toBe('/b.ts');
  });

  it('empty array clears file changes', () => {
    mockEventsEmit('session:session-1:file_changes_updated', [{ filePath: '/a.ts', originalContent: '', currentContent: '' }]);
    mockEventsEmit('session:session-1:file_changes_updated', []);
    expect(manager.fileChanges).toHaveLength(0);
  });

  it('file changes are per-session', () => {
    mockEventsEmit('session:session-1:file_changes_updated', [{ filePath: '/s1.ts', originalContent: '', currentContent: '' }]);
    manager.switchSession('session-2');
    mockEventsEmit('session:session-2:file_changes_updated', [{ filePath: '/s2.ts', originalContent: '', currentContent: '' }]);

    expect(manager.fileChanges[0].filePath).toBe('/s2.ts');
    manager.switchSession('session-1');
    expect(manager.fileChanges[0].filePath).toBe('/s1.ts');
  });
});

describe('Modes', () => {
  let manager: FullSessionManager;

  beforeEach(() => {
    eventListeners.clear();
    manager = new FullSessionManager();
    manager.switchSession('session-1');
  });

  it('sets available modes', () => {
    const modes: SessionMode[] = [
      { id: 'code', name: 'Code' },
      { id: 'plan', name: 'Plan' }
    ];
    mockEventsEmit('session:session-1:modes_available', modes);
    expect(manager.availableModes).toEqual(modes);
  });

  it('sets current mode', () => {
    mockEventsEmit('session:session-1:mode_changed', 'plan');
    expect(manager.currentModeId).toBe('plan');
  });

  it('mode changes are per-session', () => {
    mockEventsEmit('session:session-1:mode_changed', 'code');
    manager.switchSession('session-2');
    mockEventsEmit('session:session-2:mode_changed', 'plan');

    expect(manager.currentModeId).toBe('plan');
    manager.switchSession('session-1');
    expect(manager.currentModeId).toBe('code');
  });
});

describe('Plan Entries', () => {
  let manager: FullSessionManager;

  beforeEach(() => {
    eventListeners.clear();
    manager = new FullSessionManager();
    manager.switchSession('session-1');
  });

  it('sets plan entries', () => {
    const entries: PlanEntry[] = [
      { content: 'Step 1', priority: 'high', status: 'pending' },
      { content: 'Step 2', priority: 'medium', status: 'pending' }
    ];
    mockEventsEmit('session:session-1:plan_update', entries);
    expect(manager.planEntries).toEqual(entries);
  });

  it('replaces plan entries', () => {
    mockEventsEmit('session:session-1:plan_update', [{ content: 'Old', priority: 'low', status: 'pending' }]);
    mockEventsEmit('session:session-1:plan_update', [{ content: 'New', priority: 'high', status: 'in_progress' }]);
    expect(manager.planEntries).toHaveLength(1);
    expect(manager.planEntries[0].content).toBe('New');
  });

  it('plan entries are per-session', () => {
    mockEventsEmit('session:session-1:plan_update', [{ content: 'S1 plan', priority: 'high', status: 'pending' }]);
    manager.switchSession('session-2');
    mockEventsEmit('session:session-2:plan_update', [{ content: 'S2 plan', priority: 'low', status: 'completed' }]);

    expect(manager.planEntries[0].content).toBe('S2 plan');
    manager.switchSession('session-1');
    expect(manager.planEntries[0].content).toBe('S1 plan');
  });
});

describe('Message ID Generation', () => {
  let manager: FullSessionManager;

  beforeEach(() => {
    eventListeners.clear();
    manager = new FullSessionManager();
    manager.switchSession('session-1');
  });

  it('generates unique sequential IDs', () => {
    manager.sendMessage('First');
    mockEventsEmit('session:session-1:chat_chunk', 'Response');
    mockEventsEmit('session:session-1:prompt_complete');
    manager.sendMessage('Second');

    const ids = manager.messages.map(m => m.id);
    expect(ids).toEqual([1, 2, 3]);
  });

  it('IDs are unique across message types', () => {
    manager.sendMessage('User msg');
    mockEventsEmit('session:session-1:tool_state', { id: 't1', title: 'Tool', kind: 'test', status: 'running' });
    mockEventsEmit('session:session-1:chat_chunk', 'Bot msg');
    mockEventsEmit('session:session-1:prompt_complete');

    const ids = manager.messages.map(m => m.id);
    const uniqueIds = new Set(ids);
    expect(uniqueIds.size).toBe(ids.length);
  });

  it('IDs continue incrementing after session switch', () => {
    manager.sendMessage('S1 msg');
    manager.switchSession('session-2');
    manager.sendMessage('S2 msg');
    manager.switchSession('session-1');
    mockEventsEmit('session:session-1:chat_chunk', 'Response');
    mockEventsEmit('session:session-1:prompt_complete');

    // Session 1 should have IDs 1, 2
    expect(manager.messages.map(m => m.id)).toEqual([1, 2]);
  });
});
