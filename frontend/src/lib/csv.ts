function cell(v: string | number | null | undefined): string {
  let s = v == null ? '' : String(v);
  // Formula-injection guard: prefix risky leading chars (CSV convention).
  if (/^[=+\-@\t\r]/.test(s)) s = `'${s}`;
  s = s.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
}

/** Download rows (first row = header) as filename.csv. Client-side blob. */
export function downloadCSV(filename: string, rows: (string | number | null | undefined)[][]): void {
  const text = '﻿' + rows.map((r) => r.map(cell).join(',')).join('\r\n');
  const url = URL.createObjectURL(new Blob([text], { type: 'text/csv;charset=utf-8' }));
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
