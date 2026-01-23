import { describe, it, expect, beforeEach, vi } from 'vitest';

// Mock Wails runtime
type EventCallback = (...args: unknown[]) => void;
const eventListeners = new Map<string, Set<EventCallback>>();

const mockEventsOn = vi.fn((eventName: string, callback: EventCallback) => {
  if (!eventListeners.has(eventName)) {
    eventListeners.set(eventName, new Set());
  }
  eventListeners.get(eventName)!.add(callback);
  return () => {
    eventListeners.get(eventName)?.delete(callback);
  };
});

const mockEventsEmit = vi.fn((eventName: string, ...data: unknown[]) => {
  const listeners = eventListeners.get(eventName);
  if (listeners) {
    listeners.forEach(cb => cb(...data));
  }
});

vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: mockEventsOn,
  EventsEmit: mockEventsEmit,
}));

// Simplified session state management (mirrors App.svelte logic)
interface SessionState {
  messages: string[];
  currentChunk: string;
}

// OLD implementation (has bug)
class BuggySessionManager {
  private sessionStates = new Map<string, SessionState>();
  private unsubscribeFns: (() => void)[] = [];
  private activeSessionId = '';

  currentChunk = '';
  messages: string[] = [];

  getOrCreateState(id: string): SessionState {
    if (!this.sessionStates.has(id)) {
      this.sessionStates.set(id, { messages: [], currentChunk: '' });
    }
    return this.sessionStates.get(id)!;
  }

  saveCurrentState() {
    if (!this.activeSessionId) return;
    const state = this.getOrCreateState(this.activeSessionId);
    state.messages = [...this.messages];
    state.currentChunk = this.currentChunk;
  }

  loadState(id: string) {
    const state = this.getOrCreateState(id);
    this.messages = [...state.messages];
    this.currentChunk = state.currentChunk;
  }

  subscribeToSession(sessionId: string) {
    // BUG: Unsubscribes from all sessions
    this.unsubscribeFns.forEach(fn => fn());
    this.unsubscribeFns = [];

    const on = (event: string, cb: EventCallback) => {
      const unsub = mockEventsOn(`session:${sessionId}:${event}`, cb);
      this.unsubscribeFns.push(unsub);
    };

    on('chat_chunk', (text: string) => {
      this.currentChunk += text;
    });

    on('prompt_complete', () => {
      if (this.currentChunk) {
        this.messages.push(this.currentChunk);
        this.currentChunk = '';
      }
    });
  }

  switchSession(newSessionId: string) {
    if (newSessionId === this.activeSessionId) return;
    this.saveCurrentState();
    this.activeSessionId = newSessionId;
    this.loadState(newSessionId);
    this.subscribeToSession(newSessionId);
  }
}

// FIXED implementation (keeps subscriptions alive) - but has user message bug
class FixedSessionManager {
  private sessionStates = new Map<string, SessionState>();
  private subscribedSessions = new Set<string>();
  private activeSessionId = '';

  currentChunk = '';
  messages: string[] = [];

  getOrCreateState(id: string): SessionState {
    if (!this.sessionStates.has(id)) {
      this.sessionStates.set(id, { messages: [], currentChunk: '' });
    }
    return this.sessionStates.get(id)!;
  }

  loadState(id: string) {
    const state = this.getOrCreateState(id);
    this.messages = [...state.messages];
    this.currentChunk = state.currentChunk;
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

    mockEventsOn(`session:${sessionId}:prompt_complete`, () => {
      if (state.currentChunk) {
        state.messages.push(state.currentChunk);
        state.currentChunk = '';
      }
      syncIfActive();
    });
  }

  switchSession(newSessionId: string) {
    if (newSessionId === this.activeSessionId) return;
    this.activeSessionId = newSessionId;
    this.subscribeToSession(newSessionId);
    this.loadState(newSessionId);
  }

  // BUG: Only updates reactive messages, not sessionStates
  sendMessage(text: string) {
    this.messages.push(`user:${text}`);
  }
}

// FULLY FIXED implementation - user messages also go to sessionStates
class FullyFixedSessionManager {
  private sessionStates = new Map<string, SessionState>();
  private subscribedSessions = new Set<string>();
  private activeSessionId = '';

  currentChunk = '';
  messages: string[] = [];

  getOrCreateState(id: string): SessionState {
    if (!this.sessionStates.has(id)) {
      this.sessionStates.set(id, { messages: [], currentChunk: '' });
    }
    return this.sessionStates.get(id)!;
  }

  loadState(id: string) {
    const state = this.getOrCreateState(id);
    this.messages = [...state.messages];
    this.currentChunk = state.currentChunk;
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

    mockEventsOn(`session:${sessionId}:prompt_complete`, () => {
      if (state.currentChunk) {
        state.messages.push(state.currentChunk);
        state.currentChunk = '';
      }
      syncIfActive();
    });
  }

  switchSession(newSessionId: string) {
    if (newSessionId === this.activeSessionId) return;
    this.activeSessionId = newSessionId;
    this.subscribeToSession(newSessionId);
    this.loadState(newSessionId);
  }

  // FIXED: Updates both reactive messages AND sessionStates
  // Also flushes currentChunk like real App.svelte does
  sendMessage(text: string) {
    const state = this.getOrCreateState(this.activeSessionId);
    if (state.currentChunk) {
      state.messages.push(`bot:${state.currentChunk}`);
      state.currentChunk = '';
    }
    state.messages.push(`user:${text}`);
    this.messages = [...state.messages];
  }
}

describe('Session Switch During Active Message - Buggy Implementation', () => {
  let manager: BuggySessionManager;

  beforeEach(() => {
    eventListeners.clear();
    mockEventsOn.mockClear();
    mockEventsEmit.mockClear();
    manager = new BuggySessionManager();
  });

  it('captures events when subscribed to session', () => {
    manager.switchSession('session-1');
    mockEventsEmit('session:session-1:chat_chunk', 'Hello ');
    mockEventsEmit('session:session-1:chat_chunk', 'World');
    mockEventsEmit('session:session-1:prompt_complete');
    expect(manager.messages).toEqual(['Hello World']);
  });

  it('loses events emitted after switching away from session (BUG)', () => {
    manager.switchSession('session-1');
    mockEventsEmit('session:session-1:chat_chunk', 'Hello ');

    manager.switchSession('session-2');

    // Events LOST because we unsubscribed
    mockEventsEmit('session:session-1:chat_chunk', 'World');
    mockEventsEmit('session:session-1:prompt_complete');

    manager.switchSession('session-1');

    // BUG: "World" was lost
    expect(manager.currentChunk).toBe('Hello ');
    expect(manager.messages).toEqual([]);
  });
});

describe('Session Switch During Active Message - Fixed Implementation', () => {
  let manager: FixedSessionManager;

  beforeEach(() => {
    eventListeners.clear();
    mockEventsOn.mockClear();
    mockEventsEmit.mockClear();
    manager = new FixedSessionManager();
  });

  it('captures events when subscribed to session', () => {
    manager.switchSession('session-1');
    mockEventsEmit('session:session-1:chat_chunk', 'Hello ');
    mockEventsEmit('session:session-1:chat_chunk', 'World');
    mockEventsEmit('session:session-1:prompt_complete');
    expect(manager.messages).toEqual(['Hello World']);
  });

  it('captures events emitted while switched away (FIXED)', () => {
    manager.switchSession('session-1');
    mockEventsEmit('session:session-1:chat_chunk', 'Hello ');

    manager.switchSession('session-2');

    // Events still captured because subscription stays alive
    mockEventsEmit('session:session-1:chat_chunk', 'World');
    mockEventsEmit('session:session-1:prompt_complete');

    manager.switchSession('session-1');

    // FIXED: full message captured
    expect(manager.messages).toEqual(['Hello World']);
    expect(manager.currentChunk).toBe('');
  });

  it('session 2 events work correctly while session 1 is inactive', () => {
    manager.switchSession('session-1');
    manager.switchSession('session-2');

    mockEventsEmit('session:session-2:chat_chunk', 'Session 2 message');
    mockEventsEmit('session:session-2:prompt_complete');

    expect(manager.messages).toEqual(['Session 2 message']);
  });

  it('both sessions accumulate independently', () => {
    manager.switchSession('session-1');
    manager.switchSession('session-2');

    // Events to both sessions
    mockEventsEmit('session:session-1:chat_chunk', 'S1 msg');
    mockEventsEmit('session:session-2:chat_chunk', 'S2 msg');
    mockEventsEmit('session:session-1:prompt_complete');
    mockEventsEmit('session:session-2:prompt_complete');

    // Currently on session 2
    expect(manager.messages).toEqual(['S2 msg']);

    // Switch to session 1, see its message
    manager.switchSession('session-1');
    expect(manager.messages).toEqual(['S1 msg']);
  });

  it('user message lost when agent responds (BUG)', () => {
    manager.switchSession('session-1');

    // User sends a message
    manager.sendMessage('Hello');
    expect(manager.messages).toEqual(['user:Hello']);

    // Agent starts responding - syncIfActive overwrites messages
    mockEventsEmit('session:session-1:chat_chunk', 'Hi there');

    // BUG: User message is gone because loadState overwrote it
    expect(manager.messages).toEqual([]);
  });
});

describe('User Message Preservation - Fully Fixed', () => {
  let manager: FullyFixedSessionManager;

  beforeEach(() => {
    eventListeners.clear();
    mockEventsOn.mockClear();
    mockEventsEmit.mockClear();
    manager = new FullyFixedSessionManager();
  });

  it('user message preserved when agent responds (FIXED)', () => {
    manager.switchSession('session-1');

    // User sends a message
    manager.sendMessage('Hello');
    expect(manager.messages).toEqual(['user:Hello']);

    // Agent starts responding
    mockEventsEmit('session:session-1:chat_chunk', 'Hi there');

    // FIXED: User message is preserved
    expect(manager.messages).toEqual(['user:Hello']);
    expect(manager.currentChunk).toBe('Hi there');

    // Agent completes
    mockEventsEmit('session:session-1:prompt_complete');
    expect(manager.messages).toEqual(['user:Hello', 'Hi there']);
  });

  it('multiple user/agent exchanges work correctly', () => {
    manager.switchSession('session-1');

    manager.sendMessage('First question');
    mockEventsEmit('session:session-1:chat_chunk', 'First answer');
    mockEventsEmit('session:session-1:prompt_complete');

    manager.sendMessage('Second question');
    mockEventsEmit('session:session-1:chat_chunk', 'Second answer');
    mockEventsEmit('session:session-1:prompt_complete');

    expect(manager.messages).toEqual([
      'user:First question',
      'First answer',
      'user:Second question',
      'Second answer'
    ]);
  });
});

describe('Edge Cases', () => {
  let manager: FullyFixedSessionManager;

  beforeEach(() => {
    eventListeners.clear();
    mockEventsOn.mockClear();
    mockEventsEmit.mockClear();
    manager = new FullyFixedSessionManager();
  });

  it('rapid session switching during stream preserves all data', () => {
    manager.switchSession('session-1');
    mockEventsEmit('session:session-1:chat_chunk', 'A');

    manager.switchSession('session-2');
    mockEventsEmit('session:session-2:chat_chunk', 'B');

    manager.switchSession('session-1');
    mockEventsEmit('session:session-1:chat_chunk', 'C');

    manager.switchSession('session-2');
    mockEventsEmit('session:session-2:chat_chunk', 'D');

    // Complete both
    mockEventsEmit('session:session-1:prompt_complete');
    mockEventsEmit('session:session-2:prompt_complete');

    manager.switchSession('session-1');
    expect(manager.messages).toEqual(['AC']);

    manager.switchSession('session-2');
    expect(manager.messages).toEqual(['BD']);
  });

  it('user sends message, switches away, agent responds, switches back', () => {
    manager.switchSession('session-1');
    manager.sendMessage('Question');

    manager.switchSession('session-2');

    // Agent responds to session 1 while we're on session 2
    mockEventsEmit('session:session-1:chat_chunk', 'Answer');
    mockEventsEmit('session:session-1:prompt_complete');

    manager.switchSession('session-1');
    expect(manager.messages).toEqual(['user:Question', 'Answer']);
  });

  it('empty chunks are handled gracefully', () => {
    manager.switchSession('session-1');
    mockEventsEmit('session:session-1:chat_chunk', '');
    mockEventsEmit('session:session-1:chat_chunk', 'Hello');
    mockEventsEmit('session:session-1:chat_chunk', '');
    mockEventsEmit('session:session-1:prompt_complete');

    expect(manager.messages).toEqual(['Hello']);
  });

  it('multiple prompt_complete without chunks does not create empty messages', () => {
    manager.switchSession('session-1');
    mockEventsEmit('session:session-1:prompt_complete');
    mockEventsEmit('session:session-1:prompt_complete');
    mockEventsEmit('session:session-1:prompt_complete');

    expect(manager.messages).toEqual([]);
  });

  it('events for never-switched-to session are captured when finally switched', () => {
    manager.switchSession('session-1');

    // Subscribe to session-2 without switching to it
    manager['subscribeToSession']('session-2');

    // Events arrive for session-2
    mockEventsEmit('session:session-2:chat_chunk', 'Background work');
    mockEventsEmit('session:session-2:prompt_complete');

    // Still on session-1
    expect(manager.messages).toEqual([]);

    // Now switch to session-2
    manager.switchSession('session-2');
    expect(manager.messages).toEqual(['Background work']);
  });

  it('switching to same session is a no-op', () => {
    manager.switchSession('session-1');
    manager.sendMessage('Test');

    manager.switchSession('session-1');
    manager.switchSession('session-1');

    expect(manager.messages).toEqual(['user:Test']);
  });

  it('interleaved chunks from multiple sessions stay separate', () => {
    manager.switchSession('session-1');
    manager.switchSession('session-2');

    // Interleaved chunks
    mockEventsEmit('session:session-1:chat_chunk', '1');
    mockEventsEmit('session:session-2:chat_chunk', 'A');
    mockEventsEmit('session:session-1:chat_chunk', '2');
    mockEventsEmit('session:session-2:chat_chunk', 'B');
    mockEventsEmit('session:session-1:chat_chunk', '3');
    mockEventsEmit('session:session-2:chat_chunk', 'C');

    mockEventsEmit('session:session-1:prompt_complete');
    mockEventsEmit('session:session-2:prompt_complete');

    manager.switchSession('session-1');
    expect(manager.messages).toEqual(['123']);

    manager.switchSession('session-2');
    expect(manager.messages).toEqual(['ABC']);
  });

  it('user message in session 1, switch to 2, send message, both preserved', () => {
    manager.switchSession('session-1');
    manager.sendMessage('S1 msg');

    manager.switchSession('session-2');
    manager.sendMessage('S2 msg');

    manager.switchSession('session-1');
    expect(manager.messages).toEqual(['user:S1 msg']);

    manager.switchSession('session-2');
    expect(manager.messages).toEqual(['user:S2 msg']);
  });

  it('chunk arrives, user sends message, more chunks, then complete', () => {
    manager.switchSession('session-1');

    mockEventsEmit('session:session-1:chat_chunk', 'Partial ');
    manager.sendMessage('Interrupt');
    mockEventsEmit('session:session-1:chat_chunk', 'response');
    mockEventsEmit('session:session-1:prompt_complete');

    // Partial chunk flushed before user message, then new response after
    expect(manager.messages).toEqual(['bot:Partial ', 'user:Interrupt', 'response']);
  });

  it('three sessions with concurrent activity', () => {
    manager.switchSession('session-1');
    manager.switchSession('session-2');
    manager.switchSession('session-3');

    mockEventsEmit('session:session-1:chat_chunk', 'One');
    mockEventsEmit('session:session-2:chat_chunk', 'Two');
    mockEventsEmit('session:session-3:chat_chunk', 'Three');

    mockEventsEmit('session:session-2:prompt_complete');
    mockEventsEmit('session:session-1:prompt_complete');
    mockEventsEmit('session:session-3:prompt_complete');

    expect(manager.messages).toEqual(['Three']);

    manager.switchSession('session-1');
    expect(manager.messages).toEqual(['One']);

    manager.switchSession('session-2');
    expect(manager.messages).toEqual(['Two']);
  });

  it('currentChunk visible while streaming on active session', () => {
    manager.switchSession('session-1');

    mockEventsEmit('session:session-1:chat_chunk', 'Stream');
    expect(manager.currentChunk).toBe('Stream');
    expect(manager.messages).toEqual([]);

    mockEventsEmit('session:session-1:chat_chunk', 'ing...');
    expect(manager.currentChunk).toBe('Streaming...');

    mockEventsEmit('session:session-1:prompt_complete');
    expect(manager.currentChunk).toBe('');
    expect(manager.messages).toEqual(['Streaming...']);
  });

  it('currentChunk not visible for inactive session', () => {
    manager.switchSession('session-1');
    manager.switchSession('session-2');

    // Chunks for session-1 while on session-2
    mockEventsEmit('session:session-1:chat_chunk', 'Hidden');

    // currentChunk should be empty (we're on session-2)
    expect(manager.currentChunk).toBe('');

    // But switching to session-1 shows it
    manager.switchSession('session-1');
    expect(manager.currentChunk).toBe('Hidden');
  });

  it('late subscription still captures events', () => {
    // Events emitted before subscription are lost (expected)
    mockEventsEmit('session:session-1:chat_chunk', 'Lost');

    manager.switchSession('session-1');
    mockEventsEmit('session:session-1:chat_chunk', 'Found');
    mockEventsEmit('session:session-1:prompt_complete');

    expect(manager.messages).toEqual(['Found']);
  });

  it('stress test: many rapid switches and events', () => {
    const sessions = ['s1', 's2', 's3', 's4', 's5'];

    // Subscribe to all
    sessions.forEach(s => manager.switchSession(s));

    // Rapid fire events to all sessions
    for (let i = 0; i < 10; i++) {
      sessions.forEach(s => {
        mockEventsEmit(`session:${s}:chat_chunk`, `${s}-${i}`);
      });
    }

    // Complete all
    sessions.forEach(s => mockEventsEmit(`session:${s}:prompt_complete`));

    // Verify each session got its own messages
    sessions.forEach(s => {
      manager.switchSession(s);
      const expected = Array.from({ length: 10 }, (_, i) => `${s}-${i}`).join('');
      expect(manager.messages).toEqual([expected]);
    });
  });

  it('message order preserved across multiple exchanges', () => {
    manager.switchSession('session-1');

    for (let i = 1; i <= 5; i++) {
      manager.sendMessage(`Q${i}`);
      mockEventsEmit('session:session-1:chat_chunk', `A${i}`);
      mockEventsEmit('session:session-1:prompt_complete');
    }

    expect(manager.messages).toEqual([
      'user:Q1', 'A1',
      'user:Q2', 'A2',
      'user:Q3', 'A3',
      'user:Q4', 'A4',
      'user:Q5', 'A5'
    ]);
  });
});
