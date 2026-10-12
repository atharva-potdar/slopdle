<script lang="ts">
  import { ExternalLink } from 'lucide-svelte';
  import {
    api, nullStr, nullStrIds,
    type Assignment, type SubmissionReportItem,
    type StudentGradeItem, type PendingStudent, type User, type Rubric
  } from '../api';
  import { backdrop } from '../dialog';
  import { toast, undoableDelete } from '../toast';
  import { downloadCSV } from '../csv';
  import { fmtDate, fmtWhen, fmtInput } from '../time';

  export let currentUser: User;
  export let selectedCourseId: number;
  export let currentRole: string;
  export let focus: { id: number; n: number } | null = null;
  export let tabFocus: { tab: string; n: number } | null = null;
  let consumedTabFocusN = 0;

  let assignments: Assignment[] = [];
  let rubrics: Rubric[] = [];
  let selectedAssignmentId: number = 0;
  let activeTab: 'assignments' | 'report' | 'pending' | 'grades' | 'rubrics' | 'gradebook' = 'assignments';
  let batches: any[] = [];
  let mySubmissions: any[] = [];
  let consumedFocusN = 0;

  // Submission form
  let showSubmitModal = false;
  let submissionContent = '';
  let submitError = '';
  let submitSuccess = '';

  // Deep-link from dashboard: preselect assignment and open its submit modal.
  $: {
    const f = focus;
    if (f && f.n !== consumedFocusN && assignments.some((a) => a.assignmentid === f.id)) {
      consumedFocusN = f.n;
      selectedAssignmentId = f.id;
      showSubmitModal = true;
    }
  }

  // Grading form
  let showGradeModal = false;
  let gradingSubmissionId: number = 0;
  let gradingContext = '';
  let gradingContent = '';
  let gradingBreakdown = '';
  let gradingPenalty = { days: 0, enabled: false, perDay: 0 };
  let gradeScore: number = 90;
  let gradeFeedback: string = '';
  let gradingCriteria: { criterionid: number; title: string; maxpoints: number; value: number }[] = [];

  $: criterionMaxTotal = gradingCriteria.reduce((s, c) => s + c.maxpoints, 0);
  $: criterionSum = gradingCriteria.reduce((s, c) => s + (c.value || 0), 0);
  $: criterionNormalized =
    criterionMaxTotal > 0 ? Math.round((criterionSum / criterionMaxTotal) * 100 * 100) / 100 : 0;
  $: criterionFinal = finalAfterPenalty(criterionNormalized);
  // Criterion sliders drive the suggested score while the modal is open.
  $: if (showGradeModal && gradingCriteria.length > 0 && criterionMaxTotal > 0) {
    gradeScore = criterionNormalized;
  }

  function finalAfterPenalty(n: number): number {
    if (!gradingPenalty.enabled || gradingPenalty.days <= 0) return Math.round(n * 100) / 100;
    const rate = Math.min(1, (gradingPenalty.days * gradingPenalty.perDay) / 100);
    return Math.max(0, Math.round(n * (1 - rate) * 100) / 100);
  }

  // Create assignment form
  let showCreateModal = false;
  let editingAssignmentId = 0;
  let newAssignTitle = '';
  let newAssignDesc = '';
  let newAssignDeadline = '';
  let newAssignFormat = 'PDF/ZIP';
  let newAssignRubricId = 0;
  let newAssignBatchIds: number[] = [];
  let newAssignPenalty = 10;
  let newAssignPenaltyEnabled = true;

  // Rubric management form
  let showRubricModal = false;
  let editingRubricId = 0;
  let rubricTitle = '';
  let rubricDepartment = '';
  let rubricDescription = '';
  let rubricCriteria: { criterionId?: number; title: string; maxPoints: number }[] = [];
  let newCriterionTitle = '';
  let newCriterionMax: number = 10;

  // Query data
  let submissionReport: SubmissionReportItem[] = [];
  let pendingStudents: PendingStudent[] = [];
  let myGrades: StudentGradeItem[] = [];

  // Report filter + sort (sticky headers + sortable columns)
  let reportFilter = '';
  let reportSortKey = '';
  let reportSortDir: 1 | -1 = 1;

  function toggleReportSort(key: string) {
    if (reportSortKey === key) {
      reportSortDir = reportSortDir === 1 ? -1 : 1;
    } else {
      reportSortKey = key;
      reportSortDir = 1;
    }
  }

  function reportVal(r: SubmissionReportItem, key: string): string | number | null {
    switch (key) {
      case 'assignment': return r.assignmentName || '';
      case 'student': return r.studentName || '';
      case 'submitted': return r.submissionTime || null;
      case 'deadline': return r.deadline || null;
      case 'timeliness': return r.timeliness || '';
      case 'final': {
        const n = parseFloat(r.effectiveScore || r.score || 'NaN');
        return isNaN(n) ? null : n;
      }
      default: return '';
    }
  }

  $: filteredReport = submissionReport
    .filter((r) => {
      const f = reportFilter.trim().toLowerCase();
      return !f || (r.studentName || '').toLowerCase().includes(f) || (r.assignmentName || '').toLowerCase().includes(f);
    })
    .sort((a, b) => {
      if (!reportSortKey) return 0;
      const va = reportVal(a, reportSortKey);
      const vb = reportVal(b, reportSortKey);
      // Ungraded/missing sort last in both directions.
      if (va === null && vb === null) return 0;
      if (va === null) return 1;
      if (vb === null) return -1;
      return (va < vb ? -1 : va > vb ? 1 : 0) * reportSortDir;
    });

  // Speed grader: queue of ungraded submissions, arrow-key navigation
  let speedQueue: SubmissionReportItem[] = [];
  let speedIdx = 0;

  function startSpeedGrading() {
    const q = filteredReport.filter((r) => !r.score);
    if (q.length === 0) {
      toast('Nothing left to grade');
      return;
    }
    speedQueue = q;
    speedIdx = 0;
    openGradeModal(q[0]);
  }

  function stepSpeed(dir: 1 | -1) {
    if (speedQueue.length === 0) return;
    speedIdx = (speedIdx + dir + speedQueue.length) % speedQueue.length;
    openGradeModal(speedQueue[speedIdx]);
  }

  function exitSpeed() {
    speedQueue = [];
    showGradeModal = false;
  }

  // Gradebook grid: pivot report by student-id × assignment. Roster join
  // adds never-submitted students as empty cells (report holds submissions
  // only). Avg = graded cells shown; label says so.
  let gbRoster: { id: number; name: string }[] = [];
  $: gbStudents = (() => {
    const map = new Map<number, string>();
    submissionReport.forEach((r) => {
      if (!map.has(r.studentId)) map.set(r.studentId, r.studentName);
    });
    gbRoster.forEach((s) => {
      if (!map.has(s.id)) map.set(s.id, s.name);
    });
    return [...map.entries()].map(([id, name]) => ({ id, name })).sort((a, b) => a.name.localeCompare(b.name));
  })();
  $: gbCell = new Map(submissionReport.map((r) => [`${r.studentId}|${r.assignmentId}`, r]));
  $: gbAvg = new Map(
    gbStudents.map(({ id, name }) => {
      const scores = submissionReport
        .filter((r) => r.studentId === id)
        .map((r) => parseFloat(r.effectiveScore || r.score || 'NaN'))
        .filter((n) => !isNaN(n));
      return [name, scores.length ? scores.reduce((s, n) => s + n, 0) / scores.length : null] as [string, number | null];
    })
  );

  let gbEditId = 0;
  let gbEditVal = '';

  // Ledger foot: per-assignment class average + graded count.
  $: gbColStats = assignments.map((a) => {
    const vs = submissionReport
      .filter((r) => r.assignmentId === a.assignmentid)
      .map((r) => parseFloat(r.effectiveScore || r.score || 'NaN'))
      .filter((n) => !isNaN(n));
    return vs.length ? `${(vs.reduce((s, n) => s + n, 0) / vs.length).toFixed(1)} · n=${vs.length}` : '—';
  });

  function startGbEdit(rep: SubmissionReportItem) {
    gbEditId = rep.submissionId;
    gbEditVal = rep.effectiveScore || rep.score || '';
  }

  async function saveGbEdit(rep: SubmissionReportItem) {
    const v = parseFloat(gbEditVal);
    if (isNaN(v) || v < 0 || v > 100) {
      toast('Score must be 0–100');
      return;
    }
    gbEditId = 0;
    try {
      // Preserve existing feedback: quick edit only touches the number.
      const sub = await api.getSubmission(rep.submissionId);
      await api.evaluateSubmission(rep.submissionId, v, sub?.feedback?.String || '', currentUser.userId, []);
      await loadTabReports();
    } catch (err: any) {
      gbEditVal = String(v);
      gbEditId = rep.submissionId;
      toast(`${err.message} — value kept, retry`);
    }
  }

  function exportReportCSV() {
    downloadCSV(`submission-report-${selectedCourseId}.csv`, [
      ['Assignment', 'Student', 'Submitted', 'Deadline', 'Timeliness', 'Raw', 'Late', 'Final', 'Graded By'],
      ...filteredReport.map((r) => [
        r.assignmentName, r.studentName, r.submissionTime || '', r.deadline, r.timeliness,
        r.score || '', r.daysLate > 0 ? `${r.daysLate}d −${r.penaltyApplied || ''}` : '',
        r.effectiveScore || '', r.gradedBy,
      ]),
    ]);
    toast(`Exported ${filteredReport.length} rows`);
  }

  // What-if calculator (student): override scores, see projected average.
  // Grades deduped by assignment (latest row wins) so resubmits neither
  // share an override nor double-count. Ungraded excluded, counted aloud.
  let wiOverrides: Record<number, string> = {};

  $: wiDeduped = (() => {
    const map = new Map<number, StudentGradeItem>();
    myGrades.forEach((g) => map.set(g.assignmentid, g));
    return [...map.values()];
  })();
  $: wiRows = wiDeduped.map((g) => {
    const ov = wiOverrides[g.assignmentid];
    const raw = ov !== undefined && ov !== '' ? parseFloat(ov) : parseFloat(g.effectivescore ?? g.score);
    const eff = isNaN(raw) || raw < 0 || raw > 100 ? NaN : raw;
    return { ...g, shown: isNaN(eff) ? null : eff, tweaked: ov !== undefined && ov !== '' };
  });
  $: wiAvg = (() => {
    const vs = wiRows.map((r) => r.shown).filter((n): n is number => n !== null);
    return vs.length ? vs.reduce((s, n) => s + n, 0) / vs.length : null;
  })();
  $: wiExcluded = wiRows.filter((r) => r.shown === null).length;

  $: isTeacher = currentRole === 'Teacher' || currentRole === 'Teaching Assistant';

  $: if (isTeacher && activeTab === 'grades') {
    activeTab = 'assignments';
  }
  // Gradebook is staff-only (backend report endpoint 403s students anyway).
  $: if (!isTeacher && activeTab === 'gradebook') {
    activeTab = 'assignments';
  }
  $: if (!isTeacher && (activeTab === 'report' || activeTab === 'pending' || activeTab === 'rubrics')) {
    activeTab = 'assignments';
  }

  // Deep-link from notifications: land on the relevant subtab for this role.
  $: {
    const t = tabFocus;
    if (t && t.n !== consumedTabFocusN) {
      consumedTabFocusN = t.n;
      if (t.tab === 'assignments') {
        activeTab = 'assignments';
      } else if (t.tab === 'grades' && !isTeacher) {
        activeTab = 'grades';
      } else if ((t.tab === 'report' || t.tab === 'pending') && isTeacher) {
        activeTab = t.tab;
      }
    }
  }

  $: if (currentUser && selectedCourseId) {
    loadInitialData();
  }

  async function loadInitialData() {
    assignments = await api.getAssignments(selectedCourseId);
    rubrics = await api.getRubrics();
    try {
      batches = await api.getBatches(selectedCourseId);
    } catch (e) {
      batches = [];
    }
    if (!isTeacher) {
      try {
        mySubmissions = await api.getStudentSubmissions(selectedCourseId, currentUser.userId);
      } catch (e) {
        mySubmissions = [];
      }
    }
    if (assignments.length > 0) {
      // Preserve an explicit selection (deep-link focus, user pick) across
      // reloads; concurrent loads must not clobber it back to the first row.
      if (!assignments.some((a) => a.assignmentid === selectedAssignmentId)) {
        selectedAssignmentId = assignments[0].assignmentid;
      }
      await loadTabReports();
    } else {
      selectedAssignmentId = 0;
      submissionReport = [];
      pendingStudents = [];
      myGrades = [];
    }
  }

  async function loadTabReports() {
    if (!selectedCourseId) return;
    if (isTeacher) {
      submissionReport = await api.getSubmissionReport(selectedCourseId);
      // Roster for gradebook never-submitted rows (staff-only fetch).
      try {
        const enr = await api.getEnrollments(selectedCourseId);
        gbRoster = enr
          .filter((e: any) => e.rolename === 'Student')
          .map((e: any) => ({ id: e.userid, name: `${e.firstname} ${e.lastname}` }));
      } catch (e) {
        gbRoster = [];
      }
      if (selectedAssignmentId) {
        pendingStudents = await api.getPendingStudents(selectedAssignmentId);
      }
    } else {
      myGrades = await api.getStudentGrades(selectedCourseId, currentUser.userId);
    }
  }

  async function loadPending() {
    if (!selectedAssignmentId) {
      pendingStudents = [];
      return;
    }
    pendingStudents = await api.getPendingStudents(selectedAssignmentId);
  }

  function closeModals() {
    showSubmitModal = false;
    showGradeModal = false;
    showCreateModal = false;
    showRubricModal = false;
    speedQueue = [];
    gradingCriteria = [];
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') closeModals();
    // Speed-grader arrows: ignored inside inputs so number fields keep them.
    const tag = (e.target as HTMLElement)?.tagName;
    if (showGradeModal && speedQueue.length > 0 && tag !== 'INPUT' && tag !== 'TEXTAREA' && tag !== 'SELECT') {
      if (e.key === 'ArrowRight') stepSpeed(1);
      else if (e.key === 'ArrowLeft') stepSpeed(-1);
    }
  }

  async function handleSubmitWork() {
    submitError = '';
    submitSuccess = '';
    try {
      await api.submitAssignment(selectedAssignmentId, submissionContent);
      submitSuccess = 'Submission recorded successfully.';
      submissionContent = '';
      if (!isTeacher) {
        try {
          mySubmissions = await api.getStudentSubmissions(selectedCourseId, currentUser.userId);
        } catch (e) {
          mySubmissions = [];
        }
      }
      setTimeout(() => {
        showSubmitModal = false;
        loadTabReports();
      }, 1000);
    } catch (err: any) {
      submitError = err.message;
    }
  }

  async function handleGradeSubmission() {
    try {
      const critPayload =
        gradingCriteria.length > 0
          ? gradingCriteria.map((c) => ({ criterionId: c.criterionid, score: c.value }))
          : [];
      await api.evaluateSubmission(gradingSubmissionId, gradeScore, gradeFeedback, currentUser.userId, critPayload);
      await loadTabReports();
      // Speed mode: re-resolve the next item from the FRESH report (the
      // pre-save rep object is stale: score/penalty/content changed).
      if (speedQueue.length > 0) {
        const remainingIds = new Set(
          speedQueue.filter((r) => r.submissionId !== gradingSubmissionId).map((r) => r.submissionId)
        );
        speedQueue = submissionReport.filter((r) => remainingIds.has(r.submissionId) && !r.score);
        if (speedQueue.length === 0) {
          showGradeModal = false;
          toast('Queue clear — all graded');
        } else {
          if (speedIdx >= speedQueue.length) speedIdx = 0;
          openGradeModal(speedQueue[speedIdx]);
        }
      } else {
        showGradeModal = false;
      }
      gradingCriteria = [];
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function openGradeModal(rep: SubmissionReportItem) {
    gradingSubmissionId = rep.submissionId;
    gradingContext = `${rep.studentName} — ${rep.assignmentName}`;
    gradingContent = rep.content || '';
    gradingCriteria = [];
    gradingPenalty = {
      days: rep.daysLate || 0,
      enabled: rep.penaltyEnabled,
      perDay: parseFloat(rep.penaltyPerDay || '0') || 0,
    };
    // Suggested score: post-penalty effective score applied by default.
    const suggested = rep.effectiveScore ?? rep.score;
    gradeScore = suggested ? parseFloat(suggested) : 90;
    if (rep.score && rep.daysLate > 0 && rep.penaltyEnabled) {
      gradingBreakdown = `Raw ${rep.score} • ${rep.daysLate}d late × ${parseFloat(rep.penaltyPerDay || '0')}%/day • −${rep.penaltyApplied} → suggested ${rep.effectiveScore}`;
    } else if (rep.score && rep.daysLate > 0) {
      gradingBreakdown = `Raw ${rep.score} • ${rep.daysLate}d late • penalty disabled → suggested ${rep.score}`;
    } else {
      gradingBreakdown = '';
    }
    // Preload stored feedback when re-grading; default text only for fresh.
    try {
      const sub = await api.getSubmission(rep.submissionId);
      gradeFeedback = sub.feedback?.String || 'Meets rubric specifications and criteria.';
    } catch (e) {
      gradeFeedback = 'Meets rubric specifications and criteria.';
    }
    showGradeModal = true;
    // Load rubric criteria + previous criterion scores for slider grading.
    // Guard: a second open (speed advance) supersedes this fetch.
    const token = rep.submissionId;
    try {
      const a = assignments.find((x) => x.assignmentid === rep.assignmentId);
      const rid = a?.rubricid?.Valid ? a.rubricid.Int32 : 0;
      if (!rid) return;
      const crits = await api.getRubricCriteria(rid);
      if (gradingSubmissionId !== token) return;
      if (crits.length === 0) return;
      const existing: Record<number, number> = {};
      try {
        const scores = await api.getSubmissionCriteria(rep.submissionId);
        scores.forEach((s) => {
          existing[s.criterionid] = parseFloat(s.score) || 0;
        });
      } catch (e) {
        console.error(e);
      }
      if (gradingSubmissionId !== token) return;
      gradingCriteria = crits.map((c) => {
        const max = parseFloat(c.maxpoints) || 0;
        return {
          criterionid: c.criterionid,
          title: c.title,
          maxpoints: max,
          value: existing[c.criterionid] ?? max,
        };
      });
    } catch (err) {
      console.error(err);
    }
  }

  async function handleCreateAssignment() {
    if (!newAssignTitle || !newAssignDeadline) return;
    const payload = {
      title: newAssignTitle,
      description: newAssignDesc,
      deadline: new Date(newAssignDeadline + ':00Z').toISOString(),
      formatRequired: newAssignFormat,
      rubricId: newAssignRubricId || null,
      batchIds: newAssignBatchIds,
      latePenaltyPerDay: newAssignPenalty,
      latePenaltyEnabled: newAssignPenaltyEnabled,
    };
    try {
      if (editingAssignmentId) {
        await api.updateAssignment(editingAssignmentId, payload);
      } else {
        await api.createAssignment(selectedCourseId, payload);
      }
      newAssignTitle = '';
      newAssignDesc = '';
      newAssignDeadline = '';
      newAssignFormat = 'PDF/ZIP';
      newAssignRubricId = 0;
      newAssignBatchIds = [];
      newAssignPenalty = 10;
      newAssignPenaltyEnabled = true;
      editingAssignmentId = 0;
      showCreateModal = false;
      await loadInitialData();
    } catch (err: any) {
      alert(err.message);
    }
  }

  function openCreateAssignment() {
    editingAssignmentId = 0;
    newAssignTitle = '';
    newAssignDesc = '';
    newAssignDeadline = '';
    newAssignFormat = 'PDF/ZIP';
    newAssignRubricId = 0;
    newAssignBatchIds = [];
    newAssignPenalty = 10;
    newAssignPenaltyEnabled = true;
    showCreateModal = true;
  }

  function openEditAssignment(a: Assignment) {
    editingAssignmentId = a.assignmentid;
    newAssignTitle = a.title;
    newAssignDesc = a.description?.String || '';
    newAssignDeadline = a.deadline ? fmtInput(a.deadline) : '';
    newAssignFormat = a.formatrequired?.String || 'PDF/ZIP';
    newAssignRubricId = a.rubricid?.Valid ? a.rubricid.Int32 : 0;
    newAssignBatchIds = parseBatchIds(a.batchids);
    newAssignPenalty = a.latepenaltyperday ? parseFloat(a.latepenaltyperday) : 10;
    newAssignPenaltyEnabled = a.latepenaltyenabled ?? true;
    showCreateModal = true;
  }

  async function handleDeleteAssignment(id: number) {
    const a = assignments.find((x) => x.assignmentid === id);
    const cid = selectedCourseId;
    undoableDelete(`Assignment "${a?.title || id}"`, async () => {
      try {
        await api.deleteAssignment(id);
      } catch (err) {
        // Keep the current course's list truthful even on failure.
        if (selectedCourseId === cid) await loadInitialData().catch(() => {});
        throw err;
      }
      if (selectedCourseId !== cid) selectedCourseId = cid;
      activeTab = 'assignments';
      await loadInitialData();
    });
  }

  function openCreateRubric() {
    editingRubricId = 0;
    rubricTitle = '';
    rubricDepartment = '';
    rubricDescription = '';
    rubricCriteria = [];
    newCriterionTitle = '';
    newCriterionMax = 10;
    showRubricModal = true;
  }

  async function openEditRubric(r: Rubric) {
    editingRubricId = r.rubricid;
    rubricTitle = r.title;
    rubricDepartment = r.department?.String || '';
    rubricDescription = r.description?.String || '';
    rubricCriteria = [];
    newCriterionTitle = '';
    newCriterionMax = 10;
    showRubricModal = true;
    try {
      const crits = await api.getRubricCriteria(r.rubricid);
      rubricCriteria = crits.map((c) => ({
        criterionId: c.criterionid,
        title: c.title,
        maxPoints: parseFloat(c.maxpoints) || 0,
      }));
    } catch (err) {
      console.error(err);
    }
  }

  function addCriterionRow() {
    if (!newCriterionTitle.trim() || !(newCriterionMax > 0)) return;
    rubricCriteria = [...rubricCriteria, { title: newCriterionTitle.trim(), maxPoints: newCriterionMax }];
    newCriterionTitle = '';
    newCriterionMax = 10;
  }

  function removeCriterionRow(idx: number) {
    rubricCriteria = rubricCriteria.filter((_, i) => i !== idx);
  }

  async function handleSaveRubric() {
    if (!rubricTitle) return;
    const data = { title: rubricTitle, department: rubricDepartment, description: rubricDescription };
    try {
      let rid = editingRubricId;
      if (editingRubricId) {
        await api.updateRubric(editingRubricId, data);
      } else {
        const res: any = await api.createRubric(data);
        rid = res.rubricId;
      }
      if (rid) {
        await api.setRubricCriteria(rid, rubricCriteria);
      }
      showRubricModal = false;
      rubrics = await api.getRubrics();
      // Assignment cards show rubric titles; refresh them too.
      await loadInitialData();
    } catch (err: any) {
      alert(err.message);
    }
  }

  function parseBatchIds(s: string | { String: string; Valid: boolean } | null | undefined): number[] {
    return nullStrIds(s as any);
  }

  // Reactive: template reads this set so status updates when submissions load.
  $: submittedIds = new Set<number>(mySubmissions.map((s: any) => s.assignmentid));

  function isSubmitted(ids: Set<number>, aid: number): boolean {
    return ids.has(aid);
  }

  function isUrl(s: string | null | undefined): boolean {
    if (!s) return false;
    const t = s.trim().toLowerCase();
    return t.startsWith('http://') || t.startsWith('https://') || t.startsWith('/');
  }

  function isPdf(s: string | null | undefined): boolean {
    if (!s) return false;
    return s.trim().toLowerCase().split(/[?#]/)[0].endsWith('.pdf');
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="space-y-6">

  <!-- Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-800 pb-5">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight text-zinc-100">Assessments & Reports</h1>
      <p class="text-xs text-zinc-500 mt-0.5">Format-enforced submissions, rubric grading, and Case Study SQL query reports</p>
    </div>

    <!-- ShadCN Sub-tabs -->
    <div class="inline-flex h-9 items-center justify-center rounded-lg bg-zinc-900/60 p-1 text-zinc-400 border border-zinc-800 text-xs">
      <button
        class="rounded-md px-3 py-1 font-medium transition-colors {activeTab === 'assignments' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
        on:click={() => activeTab = 'assignments'}
      >
        Assignments ({assignments.length})
      </button>

      {#if isTeacher}
        <button
          class="rounded-md px-3 py-1 font-medium transition-colors {activeTab === 'report' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
          on:click={() => { activeTab = 'report'; loadTabReports(); }}
        >
          Submission Report
        </button>

        <button
          class="rounded-md px-3 py-1 font-medium transition-colors {activeTab === 'pending' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
          on:click={() => { activeTab = 'pending'; loadTabReports(); }}
        >
          Missing Submissions
        </button>

        <button
          class="rounded-md px-3 py-1 font-medium transition-colors {activeTab === 'gradebook' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
          on:click={() => { activeTab = 'gradebook'; loadTabReports(); }}
        >
          Gradebook
        </button>
      {:else}
        <button
          class="rounded-md px-3 py-1 font-medium transition-colors {activeTab === 'grades' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
          on:click={() => { activeTab = 'grades'; loadTabReports(); }}
        >
          Grade Book
        </button>
      {/if}

      {#if isTeacher}
        <button
          class="rounded-md px-3 py-1 font-medium transition-colors {activeTab === 'rubrics' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
          on:click={() => activeTab = 'rubrics'}
        >
          Rubrics ({rubrics.length})
        </button>
      {/if}
    </div>
  </div>

  <!-- TAB 1: ASSIGNMENTS LIST -->
  {#if activeTab === 'assignments'}
    <div class="space-y-4">
      {#if isTeacher}
        <div class="flex justify-end">
          <button
            on:click={openCreateAssignment}
            class="rounded-md bg-zinc-100 text-zinc-950 px-3 py-1.5 text-xs font-medium hover:bg-zinc-200 transition-colors"
          >
            New Assignment
          </button>
        </div>
      {/if}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      {#each assignments as a}
        <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-5 flex flex-col justify-between space-y-4">
          <div>
            <div class="flex items-center justify-between text-xs">
              <span class="font-mono text-zinc-500">ID #{a.assignmentid}</span>
              <div class="flex items-center space-x-1.5">
                <span class="font-mono text-[11px] text-zinc-400 px-1.5 py-0.5 rounded border border-zinc-800 bg-zinc-950 uppercase">
                  {a.formatrequired?.String || 'Text'}
                </span>
                <span class="font-mono text-[11px] text-zinc-400 px-1.5 py-0.5 rounded border border-zinc-800 bg-zinc-950">
                  {a.rubrictitle?.String || 'No rubric'}
                </span>
                {#if a.latepenaltyenabled ?? true}
                  <span class="font-mono text-[11px] text-zinc-400 px-1.5 py-0.5 rounded border border-zinc-800 bg-zinc-950">
                    Late −{parseFloat(a.latepenaltyperday || '10')}%/day
                  </span>
                {:else}
                  <span class="font-mono text-[11px] text-zinc-500 px-1.5 py-0.5 rounded border border-zinc-800 bg-zinc-950">
                    No late penalty
                  </span>
                {/if}
              </div>
            </div>

            <h3 class="text-sm font-semibold text-zinc-100 mt-2">{a.title}</h3>
            <p class="text-xs text-zinc-400 mt-1">{a.description?.String || ''}</p>

            <div class="mt-4 pt-3 border-t border-zinc-800/60 text-xs text-zinc-500 font-mono space-y-1">
              <div>Due: {fmtWhen(a.deadline)}</div>
              {#if nullStr(a.batchnames)}
                <div class="text-zinc-400">Batches: {nullStr(a.batchnames)}</div>
              {:else}
                <div class="text-zinc-400">Batches: All</div>
              {/if}
              {#if !isTeacher}
                <div class="text-zinc-200">Status: {isSubmitted(submittedIds, a.assignmentid) ? 'Submitted' : 'Not submitted'}</div>
              {/if}
            </div>
          </div>

          <div class="flex justify-end space-x-2 pt-2">
            {#if isTeacher}
              <button
                on:click={() => openEditAssignment(a)}
                class="rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-300 hover:bg-zinc-800 transition-colors"
              >
                Edit
              </button>
              <button
                on:click={() => handleDeleteAssignment(a.assignmentid)}
                class="rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-300 hover:text-red-300 hover:border-red-900/60 transition-colors"
              >
                Delete
              </button>
            {/if}
            {#if !isTeacher}
              <button
                on:click={() => { selectedAssignmentId = a.assignmentid; showSubmitModal = true; }}
                class="rounded-md bg-zinc-100 text-zinc-950 px-3 py-1.5 text-xs font-medium hover:bg-zinc-200 transition-colors"
              >
                Submit Work
              </button>
            {/if}
          </div>
        </div>
      {/each}
      {#if assignments.length === 0}
        <div class="md:col-span-2 rounded-lg border border-zinc-800/60 bg-zinc-950/40 p-8 text-center text-xs text-zinc-500">
          No assignments here yet. The instructor hasn't posted any.
        </div>
      {/if}
      </div>
    </div>
  {/if}

  <!-- TAB 2: SUBMISSION REPORT -->
  {#if activeTab === 'report'}
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/20 overflow-hidden">
      <div class="p-4 border-b border-zinc-800 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold text-zinc-100">Submission Timeliness & Grader Report</h2>
          <p class="text-xs text-zinc-500">Overview of submission times and evaluation status</p>
        </div>
        <div class="flex items-center gap-2">
          <input
            bind:value={reportFilter}
            placeholder="Filter student / assignment…"
            aria-label="Filter report rows"
            class="rounded-md border border-zinc-800 bg-zinc-950 px-2.5 py-1.5 text-xs text-zinc-100 placeholder:text-zinc-600 w-48"
          />
          <button
            on:click={startSpeedGrading}
            class="rounded-md bg-zinc-100 text-zinc-950 px-2.5 py-1.5 text-xs font-medium hover:bg-zinc-200 transition-colors whitespace-nowrap"
          >
            Speed grade ({filteredReport.filter((r) => !r.score).length})
          </button>
          <button
            on:click={exportReportCSV}
            class="rounded-md border border-zinc-800 bg-zinc-900 px-2.5 py-1.5 text-xs text-zinc-300 hover:bg-zinc-800 transition-colors"
          >
            CSV
          </button>
        </div>
      </div>

      <div class="overflow-auto max-h-[70vh]">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-zinc-800 text-zinc-400 font-medium bg-zinc-950 sticky top-0 z-10">
              <tr>
                <th class="h-9 px-4 align-middle font-medium"><button on:click={() => toggleReportSort('assignment')} class="hover:text-zinc-100">Assignment{reportSortKey === 'assignment' ? (reportSortDir === 1 ? ' ↑' : ' ↓') : ''}</button></th>
                <th class="h-9 px-4 align-middle font-medium"><button on:click={() => toggleReportSort('student')} class="hover:text-zinc-100">Student{reportSortKey === 'student' ? (reportSortDir === 1 ? ' ↑' : ' ↓') : ''}</button></th>
                <th class="h-9 px-4 align-middle font-medium">Submission</th>
                <th class="h-9 px-4 align-middle font-medium"><button on:click={() => toggleReportSort('submitted')} class="hover:text-zinc-100">Submitted{reportSortKey === 'submitted' ? (reportSortDir === 1 ? ' ↑' : ' ↓') : ''}</button></th>
                <th class="h-9 px-4 align-middle font-medium"><button on:click={() => toggleReportSort('deadline')} class="hover:text-zinc-100">Deadline{reportSortKey === 'deadline' ? (reportSortDir === 1 ? ' ↑' : ' ↓') : ''}</button></th>
                <th class="h-9 px-4 align-middle font-medium"><button on:click={() => toggleReportSort('timeliness')} class="hover:text-zinc-100">Timeliness{reportSortKey === 'timeliness' ? (reportSortDir === 1 ? ' ↑' : ' ↓') : ''}</button></th>
                <th class="h-9 px-4 align-middle font-medium">Raw</th>
                <th class="h-9 px-4 align-middle font-medium">Late</th>
                <th class="h-9 px-4 align-middle font-medium"><button on:click={() => toggleReportSort('final')} class="hover:text-zinc-100">Final{reportSortKey === 'final' ? (reportSortDir === 1 ? ' ↑' : ' ↓') : ''}</button></th>
                <th class="h-9 px-4 align-middle font-medium">Graded By</th>
                {#if isTeacher}
                  <th class="h-9 px-4 align-middle font-medium text-right">Action</th>
                {/if}
              </tr>
          </thead>
          <tbody class="divide-y divide-zinc-800/40 text-zinc-300">
            {#each filteredReport as rep}
              <tr class="hover:bg-zinc-900/40 transition-colors">
                <td class="p-4 font-medium text-zinc-100">{rep.assignmentName}</td>
                <td class="p-4 text-zinc-300">{rep.studentName}</td>
                <td class="p-4 max-w-[12rem]">
                  {#if isUrl(rep.content)}
                    <span class="font-mono text-[11px] uppercase px-1 py-0.5 rounded border border-zinc-800 bg-zinc-950 text-zinc-500 mr-1.5">
                      {isPdf(rep.content) ? 'PDF' : 'Link'}
                    </span>
                    <a
                      href={rep.content}
                      target="_blank"
                      rel="noreferrer"
                      class="inline-flex items-center gap-1 font-mono text-[11px] text-zinc-300 hover:text-white underline underline-offset-2 truncate"
                    >
                      <ExternalLink class="w-3 h-3 shrink-0" />
                      <span class="truncate">{rep.content}</span>
                    </a>
                  {:else}
                    <span class="font-mono text-[11px] text-zinc-400 truncate block" title={rep.content}>{rep.content || '—'}</span>
                  {/if}
                </td>
                <td class="p-4 font-mono text-zinc-400">{rep.submissionTime || 'N/A'}</td>
                <td class="p-4 font-mono text-zinc-400">{rep.deadline}</td>
                <td class="p-4">
                  <span class="inline-flex items-center rounded border px-1.5 py-0.5 text-[11px] font-mono {rep.timeliness === 'ON TIME' ? 'border-zinc-700 bg-zinc-800/40 text-zinc-200' : 'border-red-900/60 bg-red-950/30 text-red-400'}">
                    {rep.timeliness}
                  </span>
                </td>
                <td class="p-4 font-mono font-medium text-zinc-100">{rep.score || 'Ungraded'}</td>
                <td class="p-4 font-mono text-[11px] {rep.daysLate > 0 ? 'text-red-400' : 'text-zinc-500'}">
                  {rep.daysLate > 0 ? `${rep.daysLate}d × ${parseFloat(rep.penaltyPerDay || '0')}%${rep.penaltyApplied ? ` −${rep.penaltyApplied}` : ''}` : '—'}
                </td>
                <td class="p-4 font-mono font-medium text-zinc-100">{rep.effectiveScore || 'Ungraded'}</td>
                <td class="p-4 text-zinc-400">{rep.gradedBy}</td>
                {#if isTeacher}
                  <td class="p-4 text-right">
                    <div class="inline-flex flex-col items-end gap-1">
                      {#if rep.score}
                        <span class="inline-flex items-center rounded border border-emerald-900/60 px-1.5 py-0.5 text-[11px] font-mono text-emerald-300">
                          GRADED
                        </span>
                      {/if}
                      <button
                        on:click={() => openGradeModal(rep)}
                        class="rounded border border-zinc-800 bg-zinc-900 px-2.5 py-1.5 text-[11px] text-zinc-300 hover:text-white hover:bg-zinc-800"
                      >
                        {rep.score ? 'Regrade' : 'Grade'}
                      </button>
                    </div>
                  </td>
                {/if}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}

  <!-- TAB 3: PENDING STUDENTS -->
  {#if activeTab === 'pending'}
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/20 overflow-hidden">
      <div class="p-4 border-b border-zinc-800 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold text-zinc-100">Students Missing Submissions</h2>
          <p class="text-xs text-zinc-500">Students with past due or missing assignment submissions</p>
        </div>
        <div class="flex items-center space-x-2">
          <label for="pending-assignment-select" class="text-xs text-zinc-500 shrink-0">Assignment</label>
          <select
            id="pending-assignment-select"
            bind:value={selectedAssignmentId}
            on:change={loadPending}
            class="rounded-md border border-zinc-800 bg-zinc-950 px-2.5 py-1 text-xs text-zinc-200 "
          >
            {#each assignments as a}
              <option value={a.assignmentid}>{a.title}</option>
            {/each}
          </select>
        </div>
      </div>

      <div class="overflow-auto max-h-[70vh]">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-zinc-800 text-zinc-400 font-medium bg-zinc-950 sticky top-0 z-10">
            <tr>
              <th class="h-9 px-4 align-middle font-medium">Course</th>
              <th class="h-9 px-4 align-middle font-medium">Assignment</th>
              <th class="h-9 px-4 align-middle font-medium">Student</th>
              <th class="h-9 px-4 align-middle font-medium">Email</th>
              <th class="h-9 px-4 align-middle font-medium">Deadline</th>
              <th class="h-9 px-4 align-middle font-medium text-right">Status</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-zinc-800/40 text-zinc-300">
            {#each pendingStudents as p}
              <tr class="hover:bg-zinc-900/40 transition-colors">
                <td class="p-4 font-mono font-medium text-zinc-200">{p.coursecode}</td>
                <td class="p-4 text-zinc-100 font-medium">{p.assignmentname}</td>
                <td class="p-4 text-zinc-300">{p.firstname} {p.lastname}</td>
                <td class="p-4 font-mono text-zinc-500">{p.email}</td>
                <td class="p-4 font-mono text-zinc-400">{fmtDate(p.deadline)}</td>
                <td class="p-4 text-right">
                  <span class="inline-flex items-center rounded border border-zinc-800 px-1.5 py-0.5 text-[11px] font-mono text-zinc-400">
                    PENDING
                  </span>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}

  <!-- TAB 4: STUDENT GRADES -->
  {#if activeTab === 'grades'}
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/20 overflow-hidden">
      <div class="p-4 border-b border-zinc-800 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
        <div>
          <h2 class="text-sm font-semibold text-zinc-100">Grade Book</h2>
          <p class="text-xs text-zinc-500">Student grades and rubric feedback</p>
        </div>
        <div class="font-mono text-xs text-zinc-300">
          Projected avg: <span class="text-zinc-100 font-semibold">{wiAvg?.toFixed(1) ?? '—'}</span>
          {#if wiExcluded > 0}
            <span class="text-zinc-500">excl. {wiExcluded} ungraded</span>
          {/if}
          {#if wiRows.some((r) => r.tweaked)}
            <button on:click={() => (wiOverrides = {})} class="ml-2 text-zinc-500 hover:text-zinc-200 underline underline-offset-2">reset</button>
          {/if}
        </div>
      </div>

      <div class="divide-y divide-zinc-800/40">
        {#each wiRows as g}
          <div class="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-zinc-900/30 transition-colors">
            <div>
              <h4 class="text-xs font-semibold text-zinc-100">
                {g.assignment}
                {#if g.tweaked}
                  <span class="ml-2 inline-flex items-center rounded border border-zinc-600 px-1.5 py-0.5 text-[11px] font-mono text-zinc-300">WHAT-IF</span>
                {/if}
                {#if (g.dayslate || 0) > 0}
                  <span class="ml-2 inline-flex items-center rounded border border-red-900/60 bg-red-950/30 px-1.5 py-0.5 text-[11px] font-mono text-red-400">
                    {g.dayslate}d LATE −{g.penaltyapplied}
                  </span>
                {/if}
              </h4>
              <p class="text-xs text-zinc-400 mt-0.5">{g.feedback?.String || 'No feedback recorded.'}</p>
              {#if (g.dayslate || 0) > 0}
                <p class="text-[11px] font-mono text-zinc-500 mt-0.5">Raw {g.score} − {g.penaltyapplied} penalty</p>
              {/if}
              {#if g.criterionbreakdown?.String}
                <p class="text-[11px] font-mono text-zinc-500 mt-0.5">{g.criterionbreakdown.String}</p>
              {/if}
            </div>

            <div class="flex items-center space-x-2">
              <input
                type="number" min="0" max="100"
                value={wiOverrides[g.assignmentid] ?? ''}
                placeholder={g.shown?.toFixed(0) ?? '–'}
                aria-label={`What-if score for ${g.assignment}`}
                on:input={(e) => {
                  const v = (e.target as HTMLInputElement).value;
                  wiOverrides = v === '' ? Object.fromEntries(Object.entries(wiOverrides).filter(([k]) => k !== String(g.assignmentid))) : { ...wiOverrides, [g.assignmentid]: v };
                }}
                class="w-16 rounded-md border border-zinc-800 bg-zinc-950 px-2 py-1 font-mono text-xs text-zinc-100 placeholder:text-zinc-600 text-center"
              />
              <span class="font-mono text-xs font-semibold text-zinc-100 px-2 py-0.5 rounded border border-zinc-800 bg-zinc-900">
                {g.shown?.toFixed(1) ?? 'Ungraded'} / 100
              </span>
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}
  <!-- TAB: GRADEBOOK GRID -->
  {#if activeTab === 'gradebook' && isTeacher}
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/20 overflow-hidden">
      <div class="p-4 border-b border-zinc-800 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold text-zinc-100">Gradebook</h2>
          <p class="text-xs text-zinc-500">Students × assignments · click a score to edit inline · avg over graded cells only</p>
        </div>
        <button
          on:click={() => {
            downloadCSV(`gradebook-${selectedCourseId}.csv`, [
              ['Student', ...assignments.map((a) => a.title), 'Average'],
              ...gbStudents.map(({ id, name }) => [
                name,
                ...assignments.map((a) => {
                  const r = gbCell.get(`${id}|${a.assignmentid}`);
                  return r ? r.effectiveScore || r.score || '' : '';
                }),
                gbAvg.get(name)?.toFixed(1) || '',
              ]),
            ]);
            toast(`Exported ${gbStudents.length} students`);
          }}
          class="rounded-md border border-zinc-800 bg-zinc-900 px-2.5 py-1.5 text-xs text-zinc-300 hover:bg-zinc-800 transition-colors self-start"
        >
          CSV
        </button>
      </div>

      <div class="overflow-auto max-h-[70vh]">
        <table class="w-full text-left text-xs border-collapse">
          <thead class="text-zinc-400 font-medium">
            <tr>
              <th class="h-9 px-4 align-middle font-medium sticky left-0 top-0 z-20 bg-zinc-950 border-b border-zinc-800">Student</th>
              {#each assignments as a}
                <th class="h-9 px-4 align-middle font-medium sticky top-0 z-10 bg-zinc-950 border-b border-zinc-800 whitespace-nowrap">{a.title}</th>
              {/each}
              <th class="h-9 px-4 align-middle font-medium sticky top-0 z-10 bg-zinc-950 border-b border-zinc-800 text-right">Avg</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-zinc-800/40 text-zinc-300">
            {#each gbStudents as { id, name }}
              <tr class="hover:bg-zinc-900/40 transition-colors">
                <td class="p-4 font-medium text-zinc-100 sticky left-0 bg-zinc-950 whitespace-nowrap">{name}</td>
                {#each assignments as a}
                  {@const rep = gbCell.get(`${id}|${a.assignmentid}`)}
                  <td class="p-2 text-center">
                    {#if rep}
                      {#if gbEditId === rep.submissionId}
                        <input
                          type="number" min="0" max="100"
                          bind:value={gbEditVal}
                          on:keydown={(e) => { if (e.key === 'Enter') saveGbEdit(rep); else if (e.key === 'Escape') gbEditId = 0; }}
                          on:blur={() => { if (gbEditId) saveGbEdit(rep); }}
                          aria-label={`Score for ${name} on ${a.title}`}
                          class="w-16 rounded border border-zinc-600 bg-zinc-900 px-1.5 py-1 font-mono text-xs text-zinc-100 text-center"
                        />
                      {:else}
                        <button
                          on:click={() => startGbEdit(rep)}
                          on:dblclick={() => openGradeModal(rep)}
                          title="Click to quick-edit · double-click for full grading"
                          class="min-w-12 rounded border border-zinc-800 bg-zinc-900/60 px-2 py-1 font-mono text-xs {rep.score ? 'text-zinc-100' : 'text-zinc-600'} hover:border-zinc-600 transition-colors"
                        >
                          {rep.effectiveScore || rep.score || '—'}
                        </button>
                      {/if}
                    {:else}
                      <span class="font-mono text-xs text-zinc-700">·</span>
                    {/if}
                  </td>
                {/each}
                <td class="p-4 font-mono text-xs text-zinc-200 text-right">{gbAvg.get(name)?.toFixed(1) ?? '—'}</td>
              </tr>
            {/each}
            {#if gbStudents.length === 0}
              <tr><td colspan={assignments.length + 2} class="p-6 text-center text-zinc-500">No submissions yet — the grid fills itself.</td></tr>
            {/if}
          </tbody>
          {#if gbStudents.length > 0}
            <tfoot>
              <tr class="border-t border-zinc-700 text-zinc-400">
                <td class="p-4 font-mono text-[11px] uppercase tracking-widest sticky left-0 bg-zinc-950">Class avg</td>
                {#each gbColStats as s}
                  <td class="p-2 font-mono text-[11px] text-center whitespace-nowrap">{s}</td>
                {/each}
                <td></td>
              </tr>
            </tfoot>
          {/if}
        </table>
      </div>
    </div>
  {/if}

  <!-- TAB 5: RUBRICS -->
  {#if activeTab === 'rubrics' && isTeacher}
    <div class="space-y-4">
      <div class="flex justify-end">
        <button
          on:click={openCreateRubric}
          class="rounded-md bg-zinc-100 text-zinc-950 px-3 py-1.5 text-xs font-medium hover:bg-zinc-200 transition-colors"
        >
          New Rubric
        </button>
      </div>

      <div class="rounded-lg border border-zinc-800 bg-zinc-900/20 overflow-hidden divide-y divide-zinc-800/40">
        {#each rubrics as r}
          <div class="p-4 flex items-start justify-between gap-4 hover:bg-zinc-900/40 transition-colors">
            <div>
              <div class="flex items-center space-x-2">
                <span class="text-xs font-semibold text-zinc-100">{r.title}</span>
                {#if r.department?.String}
                  <span class="font-mono text-[11px] text-zinc-400 px-1.5 py-0.5 rounded border border-zinc-800 bg-zinc-950">
                    {r.department.String}
                  </span>
                {/if}
              </div>
              <p class="text-xs text-zinc-400 mt-1">{r.description?.String || 'No description.'}</p>
            </div>
            <button
              on:click={() => openEditRubric(r)}
              class="shrink-0 rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-300 hover:bg-zinc-800 transition-colors"
            >
              Edit
            </button>
          </div>
        {/each}
        {#if rubrics.length === 0}
          <div class="p-8 text-center text-xs text-zinc-500">No rubrics defined yet.</div>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Submit Work Modal -->
  {#if showSubmitModal}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" use:backdrop={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-md w-full p-5 space-y-4 shadow-xl">
        <h3 class="text-sm font-semibold text-zinc-100">Submit Assignment</h3>
        <p class="text-xs text-zinc-500">
          Target: <span class="text-zinc-300 font-medium">{assignments.find(a => a.assignmentid === selectedAssignmentId)?.title}</span>
        </p>

        <div>
          <label for="submission-content-input" class="block text-xs font-medium text-zinc-400 mb-1">
            GitHub Repository URL or Source Code Link
          </label>
          <input
            id="submission-content-input"
            bind:value={submissionContent}
            placeholder="https://github.com/username/repository"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 font-mono placeholder:text-zinc-500 "
          />
          <p class="text-[11px] text-zinc-500 mt-1">
            * Validated on Go server backend per format requirements.
          </p>
        </div>

        {#if submitError}
          <div class="p-2.5 rounded-md border border-red-900/60 bg-red-950/30 text-red-300 text-xs">
            {submitError}
          </div>
        {/if}

        {#if submitSuccess}
          <div class="p-2.5 rounded-md border border-emerald-900/60 bg-emerald-950/30 text-emerald-300 text-xs">
            {submitSuccess}
          </div>
        {/if}

        <div class="flex justify-end space-x-2 pt-2 border-t border-zinc-800">
          <button on:click={() => showSubmitModal = false} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
          <button on:click={handleSubmitWork} class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200">Submit</button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Grade Modal -->
  {#if showGradeModal}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" use:backdrop={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-sm w-full p-5 space-y-4 shadow-xl">
        <div>
          <h3 class="text-sm font-semibold text-zinc-100">Evaluate Submission</h3>
          {#if gradingContext}
            <p class="text-xs text-zinc-500 mt-0.5">{gradingContext}</p>
          {/if}
          {#if gradingBreakdown}
            <p class="text-[11px] font-mono text-zinc-400 mt-1">{gradingBreakdown}</p>
          {/if}
        </div>

        {#if gradingContent}
          <div class="pt-3 border-t border-zinc-800/60">
            <div class="text-[11px] font-mono uppercase tracking-wider text-zinc-500 mb-1.5">Submitted work</div>
            {#if isPdf(gradingContent) && isUrl(gradingContent)}
              <iframe
                src={gradingContent}
                title="Submission preview"
                class="w-full h-72 rounded border border-zinc-800 bg-white"
              ></iframe>
              <a
                href={gradingContent}
                target="_blank"
                rel="noreferrer"
                class="mt-2 inline-flex items-center gap-1.5 text-xs text-zinc-100 hover:text-white underline underline-offset-2 break-all"
              >
                <ExternalLink class="w-3.5 h-3.5 shrink-0" />
                <span class="break-all">Open full PDF</span>
              </a>
            {:else if isUrl(gradingContent)}
              <a
                href={gradingContent}
                target="_blank"
                rel="noreferrer"
                class="inline-flex items-center gap-1.5 text-xs text-zinc-100 hover:text-white underline underline-offset-2 break-all"
              >
                <ExternalLink class="w-3.5 h-3.5 shrink-0" />
                <span class="break-all">{gradingContent}</span>
              </a>
            {:else}
              <pre class="max-h-32 overflow-y-auto whitespace-pre-wrap break-words font-mono text-[11px] text-zinc-300">{gradingContent}</pre>
            {/if}
          </div>
        {/if}

        {#if gradingCriteria.length > 0}
          <div class="pt-3 border-t border-zinc-800/60 space-y-3">
            <div class="text-[11px] font-mono uppercase tracking-wider text-zinc-500">Rubric criteria</div>
            {#each gradingCriteria as c}
              <div>
                <div class="flex items-center justify-between text-xs mb-1">
                  <span class="text-zinc-300">{c.title}</span>
                  <span class="font-mono text-zinc-400">{c.value.toFixed(1)} / {c.maxpoints}</span>
                </div>
                <input
                  type="range"
                  min="0"
                  max={c.maxpoints}
                  step="0.5"
                  bind:value={c.value}
                  aria-label={c.title}
                  class="w-full accent-zinc-100"
                />
              </div>
            {/each}
            <p class="text-[11px] font-mono text-zinc-400">
              {criterionSum.toFixed(2)} / {criterionMaxTotal.toFixed(2)} → {criterionNormalized.toFixed(2)} → final {criterionFinal.toFixed(2)} after penalty
            </p>
          </div>
        {/if}

        <div>
          <label for="grade-score-input" class="block text-xs font-medium text-zinc-400 mb-1">Score (0 - 100)</label>
          <input
            id="grade-score-input"
            type="number"
            bind:value={gradeScore}
            min="0"
            max="100"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 "
          />
        </div>

        <div>
          <label for="grade-feedback-input" class="block text-xs font-medium text-zinc-400 mb-1">Rubric Feedback</label>
          <textarea 
            id="grade-feedback-input"
            rows="3" 
            bind:value={gradeFeedback} 
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 "
          ></textarea>
        </div>

        <div class="flex items-center justify-between pt-2 border-t border-zinc-800">
          {#if speedQueue.length > 0}
            <div class="flex items-center gap-1.5">
              <button on:click={() => stepSpeed(-1)} aria-label="Previous ungraded" class="px-2 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">←</button>
              <span class="font-mono text-[11px] text-zinc-500">{speedIdx + 1} / {speedQueue.length}</span>
              <button on:click={() => stepSpeed(1)} aria-label="Next ungraded" class="px-2 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">→</button>
              <button on:click={exitSpeed} class="px-2 py-1.5 text-xs rounded-md text-zinc-500 hover:text-zinc-200">Exit</button>
            </div>
          {:else}
            <span></span>
          {/if}
          <div class="flex space-x-2">
            <button on:click={() => { showGradeModal = false; speedQueue = []; }} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
            <button on:click={handleGradeSubmission} class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200">{speedQueue.length > 0 ? 'Save & next →' : 'Save'}</button>
          </div>
        </div>
      </div>
    </div>
  {/if}

  <!-- Create Assignment Modal -->
  {#if showCreateModal}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" use:backdrop={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-md w-full p-5 space-y-4 shadow-xl">
        <h3 class="text-sm font-semibold text-zinc-100">{editingAssignmentId ? 'Edit Assignment' : 'Create Assignment'}</h3>

        <div>
          <label for="assign-title" class="block text-xs font-medium text-zinc-400 mb-1">Title</label>
          <input id="assign-title" bind:value={newAssignTitle}
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 " />
        </div>

        <div>
          <label for="assign-desc" class="block text-xs font-medium text-zinc-400 mb-1">Description</label>
          <textarea id="assign-desc" bind:value={newAssignDesc} rows="3"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 "></textarea>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="assign-deadline" class="block text-xs font-medium text-zinc-400 mb-1">Deadline</label>
            <input id="assign-deadline" type="datetime-local" bind:value={newAssignDeadline}
              class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 " />
          </div>
          <div>
            <label for="assign-format" class="block text-xs font-medium text-zinc-400 mb-1">Format</label>
            <select id="assign-format" bind:value={newAssignFormat}
              class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 ">
              <option value="PDF/ZIP">PDF/ZIP</option>
              <option value="GitHub">GitHub</option>
              <option value="Text">Text</option>
            </select>
          </div>
        </div>

          <div>
            <label for="assign-rubric" class="block text-xs font-medium text-zinc-400 mb-1">Rubric</label>
            <select id="assign-rubric" bind:value={newAssignRubricId}
              class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 ">
              <option value={0}>None</option>
              {#each rubrics as r}
                <option value={r.rubricid}>{r.title}</option>
              {/each}
            </select>
          </div>

          <div>
            <span class="block text-xs font-medium text-zinc-400 mb-1">Target Batches</span>
            <div class="flex flex-wrap gap-2">
              {#each batches as b}
                <label class="inline-flex items-center gap-1 text-xs text-zinc-300">
                  <input type="checkbox" value={b.batchid} bind:group={newAssignBatchIds} />
                  {b.batchname}
                </label>
              {/each}
            </div>
            <p class="text-[11px] text-zinc-500 mt-1">Leave all unchecked to make it course-wide.</p>
          </div>

          <div>
            <div class="flex items-center justify-between mb-1">
              <label for="assign-penalty" class="text-xs font-medium text-zinc-400">Late penalty (% per day)</label>
              <label class="inline-flex items-center gap-1.5 text-xs text-zinc-300">
                <input id="assign-penalty-enabled" type="checkbox" bind:checked={newAssignPenaltyEnabled} />
                Enabled
              </label>
            </div>
            <div class="flex items-center gap-3">
              <input
                id="assign-penalty"
                type="range"
                min="0"
                max="100"
                step="5"
                bind:value={newAssignPenalty}
                disabled={!newAssignPenaltyEnabled}
                class="flex-1 accent-zinc-100 disabled:opacity-40"
              />
              <span class="font-mono text-xs text-zinc-200 w-12 text-right">{newAssignPenalty}%</span>
            </div>
            <p class="text-[11px] text-zinc-500 mt-1">Deducted per day late from the raw score. 0% keeps the late label with no deduction.</p>
          </div>

        <div class="flex justify-end space-x-2 pt-2 border-t border-zinc-800">
          <button on:click={() => showCreateModal = false} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
          <button on:click={handleCreateAssignment} class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200">{editingAssignmentId ? 'Save' : 'Create'}</button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Rubric Modal -->
  {#if showRubricModal}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" use:backdrop={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-md w-full p-5 space-y-4 shadow-xl">
        <h3 class="text-sm font-semibold text-zinc-100">{editingRubricId ? 'Edit Rubric' : 'Create Rubric'}</h3>

        <div>
          <label for="rubric-title" class="block text-xs font-medium text-zinc-400 mb-1">Title</label>
          <input id="rubric-title" bind:value={rubricTitle}
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 " />
        </div>

        <div>
          <label for="rubric-department" class="block text-xs font-medium text-zinc-400 mb-1">Department</label>
          <input id="rubric-department" bind:value={rubricDepartment} placeholder="Computer Science"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 " />
        </div>

        <div>
          <label for="rubric-description" class="block text-xs font-medium text-zinc-400 mb-1">Description</label>
          <textarea id="rubric-description" bind:value={rubricDescription} rows="3"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 "></textarea>
        </div>

        <div class="border-t border-zinc-800 pt-3 space-y-2">
          <span class="block text-xs font-medium text-zinc-400">Grading Criteria</span>
          {#each rubricCriteria as crit, idx}
            <div class="flex items-center gap-2">
              <input
                bind:value={crit.title}
                placeholder="Criterion title"
                aria-label="Criterion title"
                class="flex-1 rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 placeholder:text-zinc-500 "
              />
              <input
                type="number"
                bind:value={crit.maxPoints}
                min="0.5"
                step="0.5"
                aria-label="Max points"
                class="w-20 rounded-md border border-zinc-800 bg-zinc-900 px-2 py-1.5 text-xs text-zinc-100 font-mono "
              />
              <button
                type="button"
                on:click={() => removeCriterionRow(idx)}
                aria-label="Remove criterion"
                class="h-7 w-7 inline-flex items-center justify-center rounded border border-zinc-800 text-zinc-500 hover:text-red-300 hover:border-red-900/60 transition-colors relative after:absolute after:-inset-2 after:content-['']"
              >
                ×
              </button>
            </div>
          {/each}
          {#if rubricCriteria.length === 0}
            <p class="text-[11px] text-zinc-500">No criteria — grading uses a single flat score.</p>
          {/if}
          <div class="flex items-center gap-2">
            <input
              bind:value={newCriterionTitle}
              placeholder="e.g. Correctness"
              aria-label="New criterion title"
              class="flex-1 rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 placeholder:text-zinc-500 "
            />
            <input
              type="number"
              bind:value={newCriterionMax}
              min="0.5"
              step="0.5"
              aria-label="New criterion max points"
              class="w-20 rounded-md border border-zinc-800 bg-zinc-900 px-2 py-1.5 text-xs text-zinc-100 font-mono "
            />
            <button
              type="button"
              on:click={addCriterionRow}
              class="rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-300 hover:bg-zinc-800 transition-colors"
            >
              Add
            </button>
          </div>
        </div>

        <div class="flex justify-end space-x-2 pt-2 border-t border-zinc-800">
          <button on:click={() => showRubricModal = false} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
          <button on:click={handleSaveRubric} class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200">{editingRubricId ? 'Save' : 'Create'}</button>
        </div>
      </div>
    </div>
  {/if}

</div>
