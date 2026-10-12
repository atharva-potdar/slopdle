import { writable } from 'svelte/store';

export interface Toast {
  id: number;
  msg: string;
  actionLabel?: string;
  onAction?: () => void;
  ms?: number;
}

let nextId = 1;
export const toasts = writable<Toast[]>([]);

function dismiss(id: number) {
  toasts.update((ts) => ts.filter((t) => t.id !== id));
}

/** Fire-and-forget notice. Auto-dismisses after ms (default 4s). */
export function toast(msg: string, ms = 4000): number {
  const id = nextId++;
  toasts.update((ts) => [...ts, { id, msg, ms }]);
  setTimeout(() => dismiss(id), ms);
  return id;
}

/**
 * Delayed delete with Undo. Single-flight per label: scheduling a second
 * delete with the same label supersedes the first (its timer is cancelled),
 * so rapid deletes can't stack timers and race refetches. Args are captured
 * at schedule time — execute must not read live view state.
 */
export const UNDO_MS = 8000;

const pendingTimers = new Map<string, ReturnType<typeof setTimeout>>();

export function undoableDelete(label: string, execute: () => Promise<void>): void {
  const prior = pendingTimers.get(label);
  if (prior) {
    clearTimeout(prior);
    pendingTimers.delete(label);
    toasts.update((ts) => ts.filter((t) => t.msg !== `${label} will be deleted`));
  }
  const id = nextId++;
  let cancelled = false;
  const timer = setTimeout(async () => {
    pendingTimers.delete(label);
    dismiss(id);
    if (cancelled) return;
    try {
      await execute();
    } catch (err: any) {
      toast(err?.message || 'Delete failed', 6000);
    }
  }, UNDO_MS);
  pendingTimers.set(label, timer);
  toasts.update((ts) => [
    ...ts,
    {
      id,
      msg: `${label} will be deleted`,
      actionLabel: 'Undo',
      onAction: () => {
        cancelled = true;
        clearTimeout(timer);
        dismiss(id);
      },
    },
  ]);
}
