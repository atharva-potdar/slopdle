<script lang="ts">
  import { ArrowUpRight } from 'lucide-svelte';
  import { api, type User, type AttendanceSummary, type StudentGradeItem, type PendingStudent, type UserSubmission } from '../api';
  import { parseDate, dayKey, fmtDate } from '../time';

  export let currentUser: User;
  export let selectedCourseId: number;
  export let currentRole: string;
  export let enrollments: any[] = [];
  export let onNavigate: (tab: string, courseId?: number, assignmentId?: number) => void;
  export let refreshKey: number = 0;

  let attendance: AttendanceSummary | null = null;
  let grades: StudentGradeItem[] = [];
  let pendingAssignments: any[] = [];
  let myCompleted: UserSubmission[] = [];
  let pendingStudents: PendingStudent[] = [];
  let pendingGradingCount = 0;
  let enrolledStudents: any[] = [];
  let batchCount = 0;
  let announcementMsg = '';
  let announcementStatus = '';
  let loading = true;

  $: isStudent = currentRole === 'Student';
  $: isTeacherOrTA = currentRole === 'Teacher' || currentRole === 'Teaching Assistant';
  $: activeCourse = enrollments.find((e: any) => e.courseid === selectedCourseId);
  $: termWeek = (() => {
    // Academic-week flavor for the masthead meta line (best-effort).
    const now = Date.now();
    const ds = agendaSource.map((x: any) => parseDate(x.date)?.getTime()).filter((t): t is number => t !== undefined && !isNaN(t));
    if (!ds.length) return null;
    const spanWeeks = Math.max(1, Math.ceil((Math.max(...ds) - Math.min(...ds)) / (7 * 86400000)) + 1);
    const elapsed = Math.min(spanWeeks, Math.max(1, Math.ceil((now - Math.min(...ds)) / (7 * 86400000)) + 1));
    return { elapsed, spanWeeks };
  })();
  $: staffCourseIds = new Set(
    enrollments
      .filter(e => e.rolename === 'Teacher' || e.rolename === 'Teaching Assistant')
      .map(e => e.courseid)
  );

  $: if (currentUser && selectedCourseId && refreshKey >= 0) {
    loadDashboardData();
  }

  async function loadDashboardData() {
    loading = true;
    try {
      if (isStudent) {
        attendance = await api.getStudentAttendance(selectedCourseId, currentUser.userId).catch(() => null);
        grades = await api.getStudentGrades(selectedCourseId, currentUser.userId).catch(() => []);
        pendingAssignments = await api.getPendingAssignmentsForStudent(currentUser.userId).catch(() => []);
        myCompleted = await api.getUserSubmissions(currentUser.userId).catch(() => []);
      } else if (isTeacherOrTA) {
        pendingStudents = await api.getPendingStudentsForCourse(selectedCourseId).catch(() => []);
        const report = await api.getSubmissionReport(selectedCourseId).catch(() => []);
        pendingGradingCount = report.filter(r => !r.score).length;
        const enr = await api.getEnrollments(selectedCourseId).catch(() => []);
        enrolledStudents = enr.filter((e: any) => e.rolename === 'Student');
        batchCount = await api.getBatches(selectedCourseId).then((b) => b.length).catch(() => 0);
      }
    } catch (err) {
      console.error('Failed to load dashboard data', err);
    } finally {
      loading = false;
    }
  }

  async function sendAnnouncement() {
    const message = announcementMsg.trim();
    if (!message || enrolledStudents.length === 0) return;
    try {
      await Promise.all(
        enrolledStudents.map(s => api.createNotification(s.userid, message, 'Announcement', 'announcement', selectedCourseId))
      );
      announcementStatus = `Announcement sent to ${enrolledStudents.length} students.`;
      announcementMsg = '';
      setTimeout(() => announcementStatus = '', 3000);
    } catch (err: any) {
      alert(err.message);
    }
  }

  function isOverdue(deadline: string | null | undefined): boolean {
    if (!deadline) return false;
    const d = parseDate(deadline);
    return d ? d.getTime() < Date.now() : false;
  }

  // Deadline agenda + calendar (client-side over already-fetched lists).
  // Day keys are local (lib/time) — UTC keys shifted dots a day off in IST.

  // Teacher rows aggregate per assignment: "ER Diagram — 2 missing".
  // Raw per-student rows duplicated every assignment N times (noise).
  $: teacherGroups = (() => {
    const map = new Map<string, { label: string; date: string; students: string[] }>();
    pendingStudents.forEach((s: any) => {
      const k = `${s.assignmentname}|${s.deadline}`;
      let g = map.get(k);
      if (!g) {
        g = { label: s.assignmentname, date: s.deadline, students: [] };
        map.set(k, g);
      }
      g.students.push(`${s.firstname} ${s.lastname}`);
    });
    return [...map.values()];
  })();

  $: agendaSource = isStudent
    ? pendingAssignments.map((p: any) => ({ date: p.deadline, label: p.assignmentname, sub: p.coursecode, courseid: p.courseid, assignmentid: p.assignmentid, students: [] as string[] }))
    : teacherGroups.map((g) => ({ date: g.date, label: g.label, sub: `${g.students.length} missing`, courseid: null, assignmentid: null, students: g.students }));

  $: agendaWeek = (() => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const days: { date: Date; items: any[]; overdue: any[] }[] = [];
    for (let i = 0; i < 7; i++) {
      const d = new Date(today);
      d.setDate(d.getDate() + i);
      const k = dayKey(d);
      days.push({
        date: d,
        items: agendaSource.filter((x: any) => {
          const pd = parseDate(x.date);
          return pd !== null && dayKey(pd) === k;
        }),
        overdue: [],
      });
    }
    const overdue = agendaSource.filter((x: any) => {
      const pd = parseDate(x.date);
      return pd !== null && pd.getTime() < today.getTime();
    });
    if (overdue.length) days.unshift({ date: today, items: [], overdue });
    return days;
  })();

  $: calCursor = new Date();
  $: calCells = (() => {
    const y = calCursor.getFullYear();
    const m = calCursor.getMonth();
    const first = new Date(y, m, 1).getDay();
    const count = new Date(y, m + 1, 0).getDate();
    const cells: (Date | null)[] = [];
    for (let i = 0; i < first; i++) cells.push(null);
    for (let d = 1; d <= count; d++) cells.push(new Date(y, m, d));
    return cells;
  })();
  $: calMarks = (() => {
    const map = new Map<string, number>();
    agendaSource.forEach((x: any) => {
      const pd = parseDate(x.date);
      if (!pd) return;
      const k = dayKey(pd);
      map.set(k, (map.get(k) || 0) + 1);
    });
    return map;
  })();
  $: todayKey = dayKey(new Date());

  function shiftMonth(dir: 1 | -1) {
    calCursor = new Date(calCursor.getFullYear(), calCursor.getMonth() + dir, 1);
  }

  // Active work first (deadline order from API), recent completed greyed
  // at the end. Old completed work (>14 days) is hidden to cut noise.
  const RECENT_DAYS = 14;
  function isRecentCompleted(s: any): boolean {
    const pd = parseDate(s.submissiontime?.Time);
    if (!pd) return true;
    return Date.now() - pd.getTime() < RECENT_DAYS * 24 * 3600 * 1000;
  }
  $: recentCompleted = myCompleted.filter(isRecentCompleted);
  $: hiddenCompletedCount = myCompleted.length - recentCompleted.length;
  $: dashboardRows = [
    ...pendingAssignments.map((p: any) => ({
      coursecode: p.coursecode,
      courseid: p.courseid,
      assignmentid: p.assignmentid,
      assignmentname: p.assignmentname,
      format: p.formatrequired?.String || 'ANY',
      deadline: p.deadline,
      done: false,
      late: isOverdue(p.deadline),
      score: null as string | null,
      submittedOn: null as string | null,
    })),
    ...recentCompleted.map((s) => ({
      coursecode: s.coursecode,
      courseid: null as number | null,
      assignmentid: s.assignmentid,
      assignmentname: s.assignmentname,
      format: '—',
      deadline: s.deadline,
      done: true,
      late: (() => {
        const st = parseDate(s.submissiontime?.Time);
        const dl = parseDate(s.deadline);
        return !!st && !!dl && st.getTime() > dl.getTime();
      })(),
      score: s.effectivescore ?? (s.score?.String || null),
      submittedOn: s.submissiontime?.Time ? fmtDate(s.submissiontime.Time) : null,
    })),
  ];
</script>

<div class="space-y-8">

  <!-- Course masthead: the hero names the course, not the user.
       Person identity lives in the navbar account menu (no duplication). -->
  <div class="border-b border-line pb-5">
    <div class="font-mono text-[11px] uppercase tracking-widest text-faint">
      {activeCourse?.coursecode || 'Course'}
    </div>
    <h1 class="font-display text-3xl font-semibold tracking-tight text-ink mt-1">
      {activeCourse?.coursetitle || 'Dashboard'}
    </h1>
    <p class="font-mono text-[11px] text-faint mt-1.5">
      {#if isStudent}
        Student · {pendingAssignments.length} pending{termWeek ? ` · week ${termWeek.elapsed}/${termWeek.spanWeeks}` : ''}
      {:else}
        {currentRole}{enrolledStudents.length ? ` · ${enrolledStudents.length} enrolled` : ''}{batchCount ? ` · ${batchCount} batches` : ''}{termWeek ? ` · week ${termWeek.elapsed}/${termWeek.spanWeeks}` : ''}
      {/if}
    </p>
  </div>

    <!-- Week agenda + month calendar -->
    <div class="grid grid-cols-1 lg:grid-cols-5 gap-4">
      <div class="lg:col-span-3 rounded-lg border border-line bg-panel p-4">
        <h2 class="text-sm font-semibold text-ink">Next 7 days</h2>
        <p class="text-xs text-faint">{isStudent ? 'Your upcoming deadlines' : 'Missing-work deadlines this week'}</p>
        <div class="mt-3 space-y-1.5">
          {#each agendaWeek as day}
            {#if day.overdue.length > 0}
              <div class="flex items-start gap-3 rounded-md border border-red-900/60 bg-red-950/20 px-3 py-2">
                <span class="font-mono text-[11px] text-red-400 w-16 shrink-0 pt-0.5">OVERDUE</span>
                <div class="text-xs text-ink">
                  {day.overdue.length} past due: {day.overdue.slice(0, 3).map((x) => x.label).join(', ')}{day.overdue.length > 3 ? ` +${day.overdue.length - 3}` : ''}
                </div>
              </div>
            {:else}
              <div class="flex items-start gap-3 px-1 py-1">
                <span class="font-mono text-[11px] text-faint w-16 shrink-0 pt-0.5">
                  {day.date.toLocaleDateString(undefined, { weekday: 'short' })} {day.date.getDate()}
                </span>
                <div class="text-xs">
                  {#if day.items.length === 0}
                    <span class="text-faint">—</span>
                  {:else}
                    {#each day.items.slice(0, 4) as it}
                      {#if it.courseid && it.assignmentid}
                        <button on:click={() => onNavigate('assignments', it.courseid, it.assignmentid)} class="block text-left text-ink hover:underline underline-offset-2">
                          {it.label} <span class="font-mono text-[11px] text-faint">· {it.sub}</span>
                        </button>
                      {:else}
                        <span class="block text-ink" title={it.students.length ? it.students.join(', ') : ''}>
                          {it.label} <span class="font-mono text-[11px] text-faint">· {it.sub}</span>
                        </span>
                      {/if}
                    {/each}
                    {#if day.items.length > 4}
                      <span class="font-mono text-[11px] text-faint">+{day.items.length - 4} more</span>
                    {/if}
                  {/if}
                </div>
              </div>
            {/if}
          {/each}
        </div>
      </div>

      <div class="lg:col-span-2 rounded-lg border border-line bg-panel p-4">
        <div class="flex items-center justify-between">
          <h2 class="text-sm font-semibold text-ink">
            {calCursor.toLocaleDateString(undefined, { month: 'long', year: 'numeric' })}
          </h2>
          <div class="flex gap-1">
            <button on:click={() => shiftMonth(-1)} aria-label="Previous month" class="w-6 h-6 rounded border border-line text-muted hover:text-ink">‹</button>
            <button on:click={() => shiftMonth(1)} aria-label="Next month" class="w-6 h-6 rounded border border-line text-muted hover:text-ink">›</button>
          </div>
        </div>
        <div class="mt-3 grid grid-cols-7 gap-1 text-center font-mono text-[11px] text-faint">
          {#each ['S', 'M', 'T', 'W', 'T', 'F', 'S'] as d}
            <span>{d}</span>
          {/each}
          {#each calCells as c}
            {#if c === null}
              <span></span>
            {:else}
              {@const k = dayKey(c)}
              {@const n = calMarks.get(k) || 0}
              <span
                title={n ? `${n} deadline${n > 1 ? 's' : ''}` : ''}
                class="aspect-square flex flex-col items-center justify-center rounded {k === todayKey ? 'border border-line text-ink' : 'text-muted'}"
              >
                <span>{c.getDate()}</span>
                {#if n > 0}
                  <span class="flex gap-0.5 mt-0.5">
                    {#each Array(Math.min(n, 3)) as _}
                      <span class="w-1 h-1 rounded-full {k < todayKey ? 'bg-red-400' : 'bg-ink'}"></span>
                    {/each}
                  </span>
                {/if}
              </span>
            {/if}
          {/each}
        </div>
      </div>
    </div>

  {#if isStudent}
    <!-- STUDENT METRICS -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">

      <!-- Attendance Card -->
      <div class="rounded-lg border border-line bg-panel p-4">
        <div class="text-xs font-medium text-muted">Attendance </div>
        <div class="mt-2 text-2xl font-semibold tracking-tight text-ink font-mono">
          {attendance?.percentage ?? '—'}
        </div>
        <p class="text-[11px] text-faint mt-1">
          {attendance?.presentCount || 0} of {attendance?.totalSessions || 0} class sessions attended
        </p>
      </div>

      <!-- Pending Assignments -->
      <div class="rounded-lg border border-line bg-panel p-4">
        <div class="text-xs font-medium text-muted">Pending Submissions</div>
        <div class="mt-2 text-2xl font-semibold tracking-tight text-ink font-mono">
          {pendingAssignments.length}
        </div>
        <p class="text-[11px] text-faint mt-1">
          Requires format validation before deadline
        </p>
      </div>

      <!-- Evaluated Grade -->
      <div class="rounded-lg border border-line bg-panel p-4">
        <div class="text-xs font-medium text-muted">Latest Score</div>
        <div class="mt-2 text-2xl font-semibold tracking-tight text-ink font-mono">
          {grades.length > 0 ? `${grades[0].effectivescore ?? grades[0].score}` : 'Pending'}
        </div>
        <p class="text-[11px] text-faint mt-1 truncate">
          {grades.length > 0 ? grades[0].assignment : 'Awaiting instructor evaluation'}
        </p>
      </div>

    </div>

    <!-- Student Pending Work Table -->
    <div class="rounded-lg border border-line bg-panel overflow-hidden">
      <div class="px-4 py-3 border-b border-line flex items-center justify-between">
        <div>
          <h2 class="text-sm font-semibold text-ink">Course Assignments</h2>
          <p class="text-xs text-faint">
            Submissions requiring technical format verification{#if hiddenCompletedCount > 0}
              <span class="font-mono"> • {hiddenCompletedCount} older completed hidden</span>
            {/if}
          </p>
        </div>
        <button
          on:click={() => onNavigate('assignments')}
          class="text-xs text-muted hover:text-ink inline-flex items-center space-x-1"
        >
          <span>View all</span>
          <ArrowUpRight class="w-3.5 h-3.5" />
        </button>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-line text-muted font-medium bg-canvas">
            <tr>
              <th class="h-9 px-4 align-middle font-medium">Course</th>
              <th class="h-9 px-4 align-middle font-medium">Assignment</th>
              <th class="h-9 px-4 align-middle font-medium">Format Required</th>
              <th class="h-9 px-4 align-middle font-medium">Deadline</th>
              <th class="h-9 px-4 align-middle font-medium text-right">Action</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-line text-ink">
            {#each dashboardRows as item}
              <tr class="transition-colors {item.done ? 'opacity-50 hover:bg-panel' : item.late ? 'bg-red-950/20 hover:bg-red-950/30' : 'hover:bg-white/5'}">
                <td class="p-4 font-mono font-medium text-ink">{item.coursecode}</td>
                <td class="p-4 text-ink font-medium">
                  {item.assignmentname}
                  {#if item.done}
                    <span class="ml-2 inline-flex items-center rounded border border-zinc-700 px-1.5 py-0.5 text-[11px] font-mono text-ink">
                      SUBMITTED{item.submittedOn ? ` ${item.submittedOn}` : ''}
                    </span>
                  {:else if item.late}
                    <span class="ml-2 inline-flex items-center rounded border border-red-900/60 bg-red-950/30 px-1.5 py-0.5 text-[11px] font-mono text-red-400">
                      OVERDUE
                    </span>
                  {/if}
                  {#if item.done && item.late}
                    <span class="ml-2 inline-flex items-center rounded border border-red-900/60 bg-red-950/30 px-1.5 py-0.5 text-[11px] font-mono text-red-400">
                      LATE
                    </span>
                  {/if}
                </td>
                <td class="p-4 font-mono text-muted">{item.format}</td>
                <td class="p-4 font-mono {item.late && !item.done ? 'text-red-400' : 'text-muted'}">{fmtDate(item.deadline)}</td>
                <td class="p-4 text-right">
                  {#if item.done}
                    <span class="font-mono text-xs font-medium text-ink">{item.score ?? 'Ungraded'}</span>
                  {:else}
                    <button
                      on:click={() => onNavigate('assignments', item.courseid, item.assignmentid)}
                      class="rounded-md bg-ink text-canvas px-2.5 py-1 text-xs font-medium hover:bg-white transition-colors"
                    >
                      Submit
                    </button>
                  {/if}
                </td>
              </tr>
            {/each}
            {#if dashboardRows.length === 0}
              <tr>
                <td colspan="5" class="p-6 text-center text-faint">All course submissions are up to date.</td>
              </tr>
            {/if}
          </tbody>
        </table>
      </div>
    </div>

  {:else}
    <!-- INSTRUCTOR / TA METRICS -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">

      <div class="rounded-lg border border-line bg-panel p-4">
        <div class="text-xs font-medium text-muted">At Risk Students</div>
        <div class="mt-2 text-2xl font-semibold tracking-tight text-ink font-mono">
          {pendingStudents.length}
        </div>
        <p class="text-[11px] text-faint mt-1">
          Missing critical assignments
        </p>
      </div>

      <div class="rounded-lg border border-line bg-panel p-4">
        <div class="text-xs font-medium text-muted">Active Courses</div>
        <div class="mt-2 text-2xl font-semibold tracking-tight text-ink font-mono">
          {staffCourseIds.size}
        </div>
        <p class="text-[11px] text-faint mt-1">
          Total courses managed
        </p>
      </div>

      <div class="rounded-lg border border-line bg-panel p-4">
        <div class="text-xs font-medium text-muted">Pending Grading</div>
        <div class="mt-2 text-2xl font-semibold tracking-tight text-ink font-mono">
          {pendingGradingCount}
        </div>
        <p class="text-[11px] text-faint mt-1">
          Submissions awaiting review
        </p>
      </div>

    </div>

    <!-- Post Announcement -->
    <div class="rounded-lg border border-line bg-panel p-4 space-y-3">
      <div>
        <h2 class="text-sm font-semibold text-ink">Post Announcement</h2>
        <p class="text-xs text-faint">Sends a notification to all {enrolledStudents.length} enrolled students in this course</p>
      </div>
      <textarea
        bind:value={announcementMsg}
        rows="2"
        placeholder="e.g. Reminder: Assignment 3 is due Friday at 23:59."
        class="w-full rounded-md border border-line bg-canvas px-3 py-2 text-xs text-ink placeholder:text-faint "
      ></textarea>
      <div class="flex items-center justify-between">
        {#if announcementStatus}
          <span class="text-[11px] text-emerald-400 font-mono">{announcementStatus}</span>
        {:else}
          <span></span>
        {/if}
        <button
          on:click={sendAnnouncement}
          disabled={!announcementMsg.trim() || enrolledStudents.length === 0}
          class="rounded-md bg-ink text-canvas px-3 py-1.5 text-xs font-medium hover:bg-white transition-colors disabled:opacity-50"
        >
          Send Announcement
        </button>
      </div>
    </div>

    <!-- Instructor Missing Submissions Table -->
    <div class="rounded-lg border border-line bg-panel overflow-hidden">
      <div class="px-4 py-3 border-b border-line flex items-center justify-between">
        <div>
          <h2 class="text-sm font-semibold text-ink">Students Missing Submissions</h2>
          <p class="text-xs text-faint">Students with past due or missing assignment submissions</p>
        </div>
        <button
          on:click={() => onNavigate('assignments')}
          class="text-xs text-muted hover:text-ink inline-flex items-center space-x-1"
        >
          <span>Grading report</span>
          <ArrowUpRight class="w-3.5 h-3.5" />
        </button>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-line text-muted font-medium bg-canvas">
            <tr>
              <th class="h-9 px-4 align-middle font-medium">Course</th>
              <th class="h-9 px-4 align-middle font-medium">Assignment</th>
              <th class="h-9 px-4 align-middle font-medium">Student</th>
              <th class="h-9 px-4 align-middle font-medium">Email</th>
              <th class="h-9 px-4 align-middle font-medium">Deadline</th>
              <th class="h-9 px-4 align-middle font-medium text-right">Status</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-line text-ink">
            {#each pendingStudents as st}
              <tr class="transition-colors {isOverdue(st.deadline) ? 'bg-red-950/20 hover:bg-red-950/30' : 'hover:bg-white/5'}">
                <td class="p-4 font-mono font-medium text-ink">{st.coursecode}</td>
                <td class="p-4 text-ink font-medium">{st.assignmentname}</td>
                <td class="p-4 text-ink">{st.firstname} {st.lastname}</td>
                <td class="p-4 font-mono text-faint">{st.email}</td>
                <td class="p-4 font-mono {isOverdue(st.deadline) ? 'text-red-400' : 'text-muted'}">{fmtDate(st.deadline)}</td>
                <td class="p-4 text-right">
                  <span class="inline-flex items-center rounded border border-line px-1.5 py-0.5 text-[11px] font-mono text-muted">
                    UNSUBMITTED
                  </span>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}

</div>
