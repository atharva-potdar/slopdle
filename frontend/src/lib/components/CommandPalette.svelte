<script lang="ts">
  import { api } from '../api';
  import { backdrop } from '../dialog';

  export let selectedCourseId: number | null = null;
  export let activeTab: string = 'dashboard';
  export let enrollments: any[] = [];
  export let onNavigate: (tab: string, courseId?: number, assignmentId?: number) => void = () => {};
  export let onQuestion: (questionId: number, courseId: number) => void = () => {};

  interface Entry {
    label: string;
    hint: string;
    run: () => void;
  }

  let open = false;
  let query = '';
  let index = 0;
  let loaded: Entry[] = [];
  let inputEl: HTMLInputElement | null = null;

  export function launch() {
    query = '';
    index = 0;
    open = true;
    void loadCourseItems();
    setTimeout(() => inputEl?.focus(), 0);
  }

  function close() {
    open = false;
  }

  async function loadCourseItems() {
    // Snapshot course at launch; discard results if user switched mid-fetch.
    const cid = selectedCourseId;
    if (!cid) {
      loaded = [];
      return;
    }
    try {
      const [as, qs, rs] = await Promise.all([
        api.getAssignments(cid).catch(() => []),
        api.getQuestions(cid).catch(() => []),
        api.getResources(cid).catch(() => []),
      ]);
      if (selectedCourseId !== cid) return;
      loaded = [
        ...as.map((a) => ({
          label: a.title,
          hint: 'Assignment',
          run: () => onNavigate('assignments', cid, a.assignmentid),
        })),
        ...qs.map((q: any) => ({
          label: q.title,
          hint: 'Q&A thread',
          run: () => onQuestion(q.questionid, cid),
        })),
        ...rs.map((r: any) => ({
          label: r.title,
          hint: `Resource · ${r.categoryname || ''}`,
          run: () => onNavigate('courses', cid),
        })),
      ];
    } catch (e) {
      loaded = [];
    }
  }

  $: tabs = [
    { label: 'Go to Dashboard', hint: 'Tab', run: () => onNavigate('dashboard') },
    { label: 'Go to Courses & Resources', hint: 'Tab', run: () => onNavigate('courses') },
    { label: 'Go to Assignments', hint: 'Tab', run: () => onNavigate('assignments') },
    { label: 'Go to Attendance', hint: 'Tab', run: () => onNavigate('attendance') },
    { label: 'Go to Q&A', hint: 'Tab', run: () => onNavigate('qa') },
  ] as Entry[];

  $: courses = enrollments.map((e) => ({
    label: `${e.coursecode} — ${e.coursetitle}`,
    hint: `Switch course · ${e.rolename}`,
    run: () => onNavigate(activeTab, e.courseid),
  })) as Entry[];

  $: q = query.trim().toLowerCase();
  $: matched = [...tabs, ...courses, ...loaded]
    .filter((e) => !q || e.label.toLowerCase().includes(q));
  $: results = matched.slice(0, 12);
  $: hiddenCount = matched.length - results.length;
  // Terminal grammar: group rows under mono labels.
  $: groups = (() => {
    const order = ['Tab', 'Course', 'Assignment', 'Q&A thread', 'Resource'];
    const buckets = new Map<string, Entry[]>();
    results.forEach((r) => {
      const k = r.hint.startsWith('Switch course') ? 'Course' : r.hint.startsWith('Resource') ? 'Resource' : r.hint;
      if (!buckets.has(k)) buckets.set(k, []);
      buckets.get(k)!.push(r);
    });
    return order.filter((k) => buckets.has(k)).map((k) => ({ label: k, rows: buckets.get(k)! }));
  })();
  $: flatIndex = new Map(results.map((r, i) => [r, i]));
  $: if (index >= results.length) index = 0;

  function pick(e: Entry) {
    close();
    e.run();
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      index = (index + 1) % Math.max(results.length, 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      index = (index - 1 + results.length) % Math.max(results.length, 1);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (results[index]) pick(results[index]);
    }
  }
</script>

{#if open}
  <div class="fixed inset-0 z-[70] bg-black/80 flex items-start justify-center pt-24 px-4" use:backdrop={close}>
    <div class="bg-zinc-950 border border-zinc-800 rounded-lg w-full max-w-md shadow-xl overflow-hidden">
      <div class="flex items-center border-b border-zinc-800">
        <span class="pl-4 font-mono text-sm text-zinc-500 select-none" aria-hidden="true">›</span>
        <input
          bind:this={inputEl}
          bind:value={query}
          on:keydown={onKey}
          placeholder="course, assignment, question…"
          aria-label="Command palette"
          class="w-full bg-transparent px-2 py-3 text-sm text-zinc-100 placeholder:text-zinc-600 outline-none"
        />
      </div>
      <div class="max-h-72 overflow-y-auto p-1">
        {#each groups as g}
          <div class="px-3 pt-2 pb-0.5 font-mono text-[10px] uppercase tracking-widest text-zinc-600">{g.label}</div>
          {#each g.rows as r}
            {@const i = flatIndex.get(r) ?? 0}
            <button
              on:click={() => pick(r)}
              on:mouseenter={() => (index = i)}
              class="w-full text-left px-3 py-2 rounded text-xs flex items-center justify-between gap-2 {i === index ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-400'}"
            >
              <span class="truncate">{r.label}</span>
              <span class="font-mono text-[11px] text-zinc-500 shrink-0">{r.hint}</span>
            </button>
          {/each}
        {/each}
        {#if results.length === 0}
          <div class="px-3 py-4 text-center text-xs text-zinc-500">Nothing matches. Keep typing.</div>
        {:else if hiddenCount > 0}
          <div class="px-3 py-1.5 text-center font-mono text-[11px] text-zinc-600">+{hiddenCount} more — keep typing</div>
        {/if}
      </div>
      <div class="px-4 py-2 border-t border-zinc-800 font-mono text-[11px] text-zinc-600">
        ↑↓ navigate · Enter open · Esc close
      </div>
    </div>
  </div>
{/if}
