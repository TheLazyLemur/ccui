import { describe, it, expect, vi, beforeEach } from 'vitest';
import { handleVimKey, resetVimState, type VimNavCallbacks } from './vimNav';

function makeEvent(key: string, mods: Partial<KeyboardEvent> = {}): KeyboardEvent {
  return { key, metaKey: false, ctrlKey: false, altKey: false, ...mods } as KeyboardEvent;
}

function makeCallbacks(): VimNavCallbacks & { [K in keyof VimNavCallbacks]: ReturnType<typeof vi.fn> } {
  return {
    setFocusedPanel: vi.fn(),
    scrollChat: vi.fn(),
    scrollChatToEdge: vi.fn(),
    focusInput: vi.fn(),
  };
}

describe('handleVimKey', () => {
  beforeEach(() => resetVimState());

  describe('panel focus', () => {
    it('h sets focused panel to left', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('h'), 'right', 'chat', cb);
      expect(handled).toBe(true);
      expect(cb.setFocusedPanel).toHaveBeenCalledWith('left');
    });

    it('l sets focused panel to right', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('l'), 'left', 'chat', cb);
      expect(handled).toBe(true);
      expect(cb.setFocusedPanel).toHaveBeenCalledWith('right');
    });
  });

  describe('scrolling', () => {
    it('j scrolls chat down', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('j'), 'left', 'chat', cb);
      expect(handled).toBe(true);
      expect(cb.scrollChat).toHaveBeenCalledWith('down');
    });

    it('k scrolls chat up', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('k'), 'left', 'chat', cb);
      expect(handled).toBe(true);
      expect(cb.scrollChat).toHaveBeenCalledWith('up');
    });

    it('G scrolls to bottom', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('G'), 'left', 'chat', cb);
      expect(handled).toBe(true);
      expect(cb.scrollChatToEdge).toHaveBeenCalledWith('bottom');
    });

    it('gg scrolls to top', () => {
      const cb = makeCallbacks();
      handleVimKey(makeEvent('g'), 'left', 'chat', cb);
      const handled = handleVimKey(makeEvent('g'), 'left', 'chat', cb);
      expect(handled).toBe(true);
      expect(cb.scrollChatToEdge).toHaveBeenCalledWith('top');
    });

    it('single g does not scroll to top', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('g'), 'left', 'chat', cb);
      expect(handled).toBe(true);
      expect(cb.scrollChatToEdge).not.toHaveBeenCalled();
    });
  });

  describe('input focus', () => {
    it('i focuses input', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('i'), 'left', 'chat', cb);
      expect(handled).toBe(true);
      expect(cb.focusInput).toHaveBeenCalled();
    });

    it('/ focuses input', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('/'), 'left', 'chat', cb);
      expect(handled).toBe(true);
      expect(cb.focusInput).toHaveBeenCalled();
    });
  });

  describe('guards', () => {
    it('ignores when metaKey held', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('h', { metaKey: true }), 'left', 'chat', cb);
      expect(handled).toBe(false);
      expect(cb.setFocusedPanel).not.toHaveBeenCalled();
    });

    it('ignores when ctrlKey held', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('h', { ctrlKey: true }), 'left', 'chat', cb);
      expect(handled).toBe(false);
    });

    it('ignores when altKey held', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('h', { altKey: true }), 'left', 'chat', cb);
      expect(handled).toBe(false);
    });

    it('ignores when focused panel is terminal', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('h'), 'left', 'terminal', cb);
      expect(handled).toBe(false);
      expect(cb.setFocusedPanel).not.toHaveBeenCalled();
    });

    it('returns false for unrecognised key', () => {
      const cb = makeCallbacks();
      const handled = handleVimKey(makeEvent('x'), 'left', 'chat', cb);
      expect(handled).toBe(false);
    });
  });
});
