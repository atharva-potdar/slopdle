/**
 * Single home for all timestamp handling.
 *
 * Backend contract: MariaDB DATETIME holds local wall time, Go reads it with
 * loc=Local, JSON carries real instants (RFC3339 with offset). So `new Date`
 * on any API timestamp is already correct — the historical bugs came from
 * `toISOString()` (UTC wall) for display/keys. Never use toISOString here.
 */

type MaybeTime = string | number | Date | { Time?: string; Valid?: boolean } | null | undefined;

export function parseDate(v: MaybeTime): Date | null {
  if (v == null) return null;
  if (v instanceof Date) return isNaN(v.getTime()) ? null : v;
  if (typeof v === 'number') return new Date(v);
  if (typeof v === 'object') {
    if (!v.Valid || !v.Time) return null;
    return parseDate(v.Time);
  }
  const d = new Date(v);
  return isNaN(d.getTime()) ? null : d;
}

const pad = (n: number) => String(n).padStart(2, '0');

/** Local calendar day key. Only day-bucketing function — UTC keys banned. */
export function dayKey(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** Local YYYY-MM-DD. */
export function fmtDate(v: MaybeTime): string {
  const d = parseDate(v);
  return d ? dayKey(d) : 'N/A';
}

/** Local `Tue Oct 14, 21:33`. Human timestamps, never ISO. */
export function fmtWhen(v: MaybeTime): string {
  const d = parseDate(v);
  if (!d) return '—';
  return `${d.toLocaleDateString(undefined, { weekday: 'short', month: 'short', day: 'numeric' })}, ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** `2h ago` / `Tue 14:03` / full date. For feeds, threads, notifications. */
export function timeAgo(v: MaybeTime): string {
  const d = parseDate(v);
  if (!d) return '—';
  const s = Math.floor((Date.now() - d.getTime()) / 1000);
  if (s < 60) return 'just now';
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  if (s < 7 * 86400) return `${Math.floor(s / 86400)}d ago`;
  return fmtWhen(v);
}

/** `YYYY-MM-DDTHH:MM` local wall for datetime-local inputs. */
export function fmtInput(v: MaybeTime): string {
  const d = parseDate(v);
  return d ? `${dayKey(d)}T${pad(d.getHours())}:${pad(d.getMinutes())}` : '';
}
