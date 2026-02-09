export type PanelSide = 'left' | 'right';
export type PanelType = 'chat' | 'review' | 'terminal' | 'automations' | null;

export interface VimNavCallbacks {
  setFocusedPanel: (panel: PanelSide) => void;
  scrollChat: (direction: 'up' | 'down') => void;
  scrollChatToEdge: (edge: 'top' | 'bottom') => void;
  focusInput: () => void;
}

let lastKey = '';
let lastKeyTime = 0;

export function resetVimState() {
  lastKey = '';
  lastKeyTime = 0;
}

/**
 * Handle vim-style navigation keys. Returns true if key was consumed.
 */
export function handleVimKey(
  e: KeyboardEvent,
  focusedPanel: PanelSide,
  focusedPanelType: PanelType,
  callbacks: VimNavCallbacks
): boolean {
  if (e.metaKey || e.ctrlKey || e.altKey) return false;
  if (focusedPanelType === 'terminal') return false;

  const key = e.key;

  switch (key) {
    case 'h':
      callbacks.setFocusedPanel('left');
      return true;
    case 'l':
      callbacks.setFocusedPanel('right');
      return true;
    case 'j':
      callbacks.scrollChat('down');
      return true;
    case 'k':
      callbacks.scrollChat('up');
      return true;
    case 'G':
      callbacks.scrollChatToEdge('bottom');
      return true;
    case 'g': {
      const now = Date.now();
      if (lastKey === 'g' && now - lastKeyTime < 500) {
        callbacks.scrollChatToEdge('top');
        lastKey = '';
        lastKeyTime = 0;
        return true;
      }
      lastKey = 'g';
      lastKeyTime = now;
      return true;
    }
    case 'i':
    case '/':
      callbacks.focusInput();
      return true;
    default:
      return false;
  }
}
