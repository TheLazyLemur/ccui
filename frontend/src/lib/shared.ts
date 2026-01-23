export interface PatchHunk {
  oldStart: number;
  oldLines: number;
  newStart: number;
  newLines: number;
  lines: string[];
}

export interface ToolCall {
  id: string;
  title: string;
  kind: string;
  status: string;
  toolName?: string;
  parentId?: string;
  input?: Record<string, unknown>;
  output?: unknown[];
  diffs?: { type: string; path?: string; oldText?: string; newText?: string }[];
  diff?: { filePath?: string; structuredPatch?: PatchHunk[]; content?: string };
  permissionOptions?: { optionId: string; name: string; kind: string }[];
}

export interface Message {
  id: number;
  text: string;
  sender: 'user' | 'bot' | 'tool';
  toolState?: ToolCall;
}

export interface UserQuestion {
  requestId: string;
  question: string;
  options?: { label: string; description?: string }[];
}

const STATUS_INDICATORS: Record<string, string> = {
  pending: '○', awaiting_permission: '◇', running: '◎', completed: '●', error: '✕'
};

const STATUS_CLASSES: Record<string, string> = {
  pending: 'text-ink-muted', awaiting_permission: 'text-accent-warning',
  running: 'text-ink-medium', completed: 'text-accent-success', error: 'text-accent-danger'
};

export const getStatusIndicator = (status: string): string => STATUS_INDICATORS[status] || '○';
export const getStatusClass = (status: string): string => STATUS_CLASSES[status] || 'text-ink-muted';

export function getButtonClass(kind: string): string {
  if (kind.startsWith('allow')) return 'border-accent-success text-accent-success hover:bg-accent-success/10';
  if (kind.startsWith('reject')) return 'border-accent-danger text-accent-danger hover:bg-accent-danger/10';
  return 'border-ink-faint text-ink-medium hover:bg-paper-dim';
}
