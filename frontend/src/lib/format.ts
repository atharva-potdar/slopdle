/** Short human form of a URL: decoded last path segment, middle-truncated.
 *  `https://cdn/x/syllabus-2024-final.pdf` → `syllabus…24.pdf`. */
export function shortUrl(url: string | null | undefined, max = 32): string {
  if (!url) return '—';
  let tail = url.trim();
  try {
    const u = new URL(tail);
    const segs = u.pathname.split('/').filter(Boolean);
    if (segs.length) tail = segs[segs.length - 1];
  } catch (e) {
    const parts = tail.split('/').filter(Boolean);
    tail = parts.length ? parts[parts.length - 1] : tail;
  }
  try {
    tail = decodeURIComponent(tail);
  } catch (e) {
    /* keep raw */
  }
  if (tail.length <= max) return tail;
  const dot = tail.lastIndexOf('.');
  const ext = dot > 0 && tail.length - dot <= 6 ? tail.slice(dot) : '';
  const stem = ext ? tail.slice(0, -ext.length) : tail;
  const keep = Math.max(6, max - ext.length - 1);
  const head = Math.ceil(keep / 2);
  const last = Math.floor(keep / 2);
  return `${stem.slice(0, head)}…${stem.slice(-last)}${ext}`;
}
