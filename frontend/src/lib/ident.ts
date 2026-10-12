/**
 * Generative course mark: deterministic 5x5 block pattern hashed from the
 * course code. Zinc fills only — zero palette cost, zero assets, every
 * course instantly recognizable. Replaces decorative generic icons.
 */
function hash(str: string): number {
  let h = 2166136261;
  for (let i = 0; i < str.length; i++) {
    h ^= str.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

/** 25 booleans, mirrored horizontally for symmetry (like identicons). */
export function identCells(code: string): boolean[] {
  const cells: boolean[] = new Array(25).fill(false);
  let h = hash(code || '?');
  for (let row = 0; row < 5; row++) {
    for (let col = 0; col < 3; col++) {
      h = (Math.imul(h, 1103515245) + 12345) & 0x7fffffff;
      const on = h % 2 === 0;
      cells[row * 5 + col] = on;
      cells[row * 5 + (4 - col)] = on;
    }
  }
  return cells;
}
