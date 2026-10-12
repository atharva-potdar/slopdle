<script lang="ts">
  import { ExternalLink } from 'lucide-svelte';
  import {
    api, nullStr, nullStrIds,
    type Assignment, type SubmissionReportItem,
    type StudentGradeItem, type PendingStudent, type User, type Rubric
  } from '../api';

  export let currentUser: User;
  export let selectedCourseId: number;
  export let currentRole: string;
  export let focus: { id: number; n: number } | null = null;
  export let tabFocus: { tab: string; n: number } | null = null;
  let consumedTabFocusN = 0;

  let assignments: Assignment[] = [];
  let rubrics: Rubric[] = [];
  let selectedAssignmentId: number = 0;
  let activeTab: 'assignments' | 'report' | 'pending' | 'grades' | 'rubrics' = 'assignments';
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

  $: isTeacher = currentRole === 'Teacher' || currentRole === 'Teaching Assistant';

  $: if (isTeacher && activeTab === 'grades') {
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
    gradingCriteria = [];
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') closeModals();
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
      showGradeModal = false;
      gradingCriteria = [];
      await loadTabReports();
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
    gradeFeedback = 'Meets rubric specifications and criteria.';
    showGradeModal = true;
    // Load rubric criteria + previous criterion scores for slider grading.
    try {
      const a = assignments.find((x) => x.assignmentid === rep.assignmentId);
      const rid = a?.rubricid?.Valid ? a.rubricid.Int32 : 0;
      if (!rid) return;
      const crits = await api.getRubricCriteria(rid);
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
    newAssignDeadline = a.deadline ? new Date(a.deadline).toISOString().slice(0, 16) : '';
    newAssignFormat = a.formatrequired?.String || 'PDF/ZIP';
    newAssignRubricId = a.rubricid?.Valid ? a.rubricid.Int32 : 0;
    newAssignBatchIds = parseBatchIds(a.batchids);
    newAssignPenalty = a.latepenaltyperday ? parseFloat(a.latepenaltyperday) : 10;
    newAssignPenaltyEnabled = a.latepenaltyenabled ?? true;
    showCreateModal = true;
  }

  async function handleDeleteAssignment(id: number) {
    if (!confirm('Delete this assignment? Submissions referencing it may block deletion.')) return;
    try {
      await api.deleteAssignment(id);
      await loadInitialData();
    } catch (err: any) {
      alert(err.message);
    }
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
      <h1 class="text-xl font-semibold tracking-tight text-zinc-100">Assessments & Reports</h1>
      <p class="text-xs text-zinc-500 mt-0.5">Format-enforced submissions, rubric grading, and Case Study SQL query reports</p>
    </div>

    <!-- ShadCN Sub-tabs -->
    <div class="inline-flex h-9 items-center justify-center rounded-lg bg-zinc-900/60 p-1 text-zinc-400 border border-zinc-800 text-xs">
      <button
        class="rounded-md px-3 py-1 font-medium transition-all {activeTab === 'assignments' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
        on:click={() => activeTab = 'assignments'}
      >
        Assignments ({assignments.length})
      </button>

      {#if isTeacher}
        <button
          class="rounded-md px-3 py-1 font-medium transition-all {activeTab === 'report' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
          on:click={() => { activeTab = 'report'; loadTabReports(); }}
        >
          Submission Report
        </button>

        <button
          class="rounded-md px-3 py-1 font-medium transition-all {activeTab === 'pending' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
          on:click={() => { activeTab = 'pending'; loadTabReports(); }}
        >
          Missing Submissions
        </button>
      {:else}
        <button
          class="rounded-md px-3 py-1 font-medium transition-all {activeTab === 'grades' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
          on:click={() => { activeTab = 'grades'; loadTabReports(); }}
        >
          Grade Book
        </button>
      {/if}

      {#if isTeacher}
        <button
          class="rounded-md px-3 py-1 font-medium transition-all {activeTab === 'rubrics' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
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
                <span class="font-mono text-[10px] text-zinc-400 px-1.5 py-0.5 rounded border border-zinc-800 bg-zinc-950 uppercase">
                  {a.formatrequired?.String || 'Text'}
                </span>
                <span class="font-mono text-[10px] text-zinc-400 px-1.5 py-0.5 rounded border border-zinc-800 bg-zinc-950">
                  {a.rubrictitle?.String || 'No rubric'}
                </span>
                {#if a.latepenaltyenabled ?? true}
                  <span class="font-mono text-[10px] text-zinc-400 px-1.5 py-0.5 rounded border border-zinc-800 bg-zinc-950">
                    Late −{parseFloat(a.latepenaltyperday || '10')}%/day
                  </span>
                {:else}
                  <span class="font-mono text-[10px] text-zinc-500 px-1.5 py-0.5 rounded border border-zinc-800 bg-zinc-950">
                    No late penalty
                  </span>
                {/if}
              </div>
            </div>

            <h3 class="text-sm font-semibold text-zinc-100 mt-2">{a.title}</h3>
            <p class="text-xs text-zinc-400 mt-1">{a.description?.String || ''}</p>

            <div class="mt-4 pt-3 border-t border-zinc-800/60 text-xs text-zinc-500 font-mono space-y-1">
              <div>Due: {new Date(a.deadline).toISOString().replace('T', ' ').slice(0, 16)}</div>
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
          No assignments for this course yet.
        </div>
      {/if}
      </div>
    </div>
  {/if}

  <!-- TAB 2: SUBMISSION REPORT -->
  {#if activeTab === 'report'}
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/20 overflow-hidden">
      <div class="p-4 border-b border-zinc-800 flex items-center justify-between">
        <div>
          <h2 class="text-sm font-semibold text-zinc-100">Submission Timeliness & Grader Report</h2>
          <p class="text-xs text-zinc-500">Overview of submission times and evaluation status</p>
        </div>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-zinc-800 text-zinc-400 font-medium bg-zinc-950/40">
              <tr>
                <th class="h-9 px-4 align-middle font-medium">Assignment</th>
                <th class="h-9 px-4 align-middle font-medium">Student</th>
                <th class="h-9 px-4 align-middle font-medium">Submission</th>
                <th class="h-9 px-4 align-middle font-medium">Submitted</th>
                <th class="h-9 px-4 align-middle font-medium">Deadline</th>
                <th class="h-9 px-4 align-middle font-medium">Timeliness</th>
                <th class="h-9 px-4 align-middle font-medium">Raw</th>
                <th class="h-9 px-4 align-middle font-medium">Late</th>
                <th class="h-9 px-4 align-middle font-medium">Final</th>
                <th class="h-9 px-4 align-middle font-medium">Graded By</th>
                {#if isTeacher}
                  <th class="h-9 px-4 align-middle font-medium text-right">Action</th>
                {/if}
              </tr>
          </thead>
          <tbody class="divide-y divide-zinc-800/40 text-zinc-300">
            {#each submissionReport as rep}
              <tr class="hover:bg-zinc-900/40 transition-colors">
                <td class="p-4 font-medium text-zinc-100">{rep.assignmentName}</td>
                <td class="p-4 text-zinc-300">{rep.studentName}</td>
                <td class="p-4 max-w-[12rem]">
                  {#if isUrl(rep.content)}
                    <span class="font-mono text-[9px] uppercase px-1 py-0.5 rounded border border-zinc-800 bg-zinc-950 text-zinc-500 mr-1.5">
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
                  <span class="inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-mono {rep.timeliness === 'ON TIME' ? 'border-zinc-700 bg-zinc-800/40 text-zinc-200' : 'border-red-900/60 bg-red-950/30 text-red-400'}">
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
                    <button
                      on:click={() => openGradeModal(rep)}
                      class="rounded border border-zinc-800 bg-zinc-900 px-2 py-1 text-[11px] text-zinc-300 hover:text-white hover:bg-zinc-800"
                    >
                      Grade
                    </button>
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
            class="rounded-md border border-zinc-800 bg-zinc-950 px-2.5 py-1 text-xs text-zinc-200 focus:outline-none focus:ring-1 focus:ring-zinc-400"
          >
            {#each assignments as a}
              <option value={a.assignmentid}>{a.title}</option>
            {/each}
          </select>
        </div>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-zinc-800 text-zinc-400 font-medium bg-zinc-950/40">
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
                <td class="p-4 font-mono text-zinc-400">{new Date(p.deadline).toISOString().split('T')[0]}</td>
                <td class="p-4 text-right">
                  <span class="inline-flex items-center rounded border border-zinc-800 px-1.5 py-0.5 text-[10px] font-mono text-zinc-400">
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
      <div class="p-4 border-b border-zinc-800">
        <h2 class="text-sm font-semibold text-zinc-100">Grade Book</h2>
        <p class="text-xs text-zinc-500">Student grades and rubric feedback</p>
      </div>

      <div class="divide-y divide-zinc-800/40">
        {#each myGrades as g}
          <div class="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-zinc-900/30 transition-colors">
            <div>
              <h4 class="text-xs font-semibold text-zinc-100">
                {g.assignment}
                {#if (g.dayslate || 0) > 0}
                  <span class="ml-2 inline-flex items-center rounded border border-red-900/60 bg-red-950/30 px-1.5 py-0.5 text-[10px] font-mono text-red-400">
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

            <div class="flex items-center space-x-4">
              <span class="font-mono text-xs font-semibold text-zinc-100 px-2 py-0.5 rounded border border-zinc-800 bg-zinc-900">
                {g.effectivescore ?? g.score} / 100
              </span>
            </div>
          </div>
        {/each}
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
                  <span class="font-mono text-[10px] text-zinc-400 px-1.5 py-0.5 rounded border border-zinc-800 bg-zinc-950">
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
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" on:click={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-md w-full p-5 space-y-4 shadow-xl" on:click|stopPropagation>
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
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 font-mono placeholder:text-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-400"
          />
          <p class="text-[10px] text-zinc-500 mt-1">
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
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" on:click={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-sm w-full p-5 space-y-4 shadow-xl" on:click|stopPropagation>
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
          <div class="rounded-md border border-zinc-800 bg-zinc-900/40 p-3">
            <div class="text-[10px] font-mono uppercase tracking-wider text-zinc-500 mb-1.5">Submitted work</div>
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
          <div class="rounded-md border border-zinc-800 bg-zinc-900/40 p-3 space-y-3">
            <div class="text-[10px] font-mono uppercase tracking-wider text-zinc-500">Rubric criteria</div>
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
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400"
          />
        </div>

        <div>
          <label for="grade-feedback-input" class="block text-xs font-medium text-zinc-400 mb-1">Rubric Feedback</label>
          <textarea 
            id="grade-feedback-input"
            rows="3" 
            bind:value={gradeFeedback} 
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400"
          ></textarea>
        </div>

        <div class="flex justify-end space-x-2 pt-2 border-t border-zinc-800">
          <button on:click={() => showGradeModal = false} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
          <button on:click={handleGradeSubmission} class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200">Save</button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Create Assignment Modal -->
  {#if showCreateModal}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" on:click={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-md w-full p-5 space-y-4 shadow-xl" on:click|stopPropagation>
        <h3 class="text-sm font-semibold text-zinc-100">{editingAssignmentId ? 'Edit Assignment' : 'Create Assignment'}</h3>

        <div>
          <label for="assign-title" class="block text-xs font-medium text-zinc-400 mb-1">Title</label>
          <input id="assign-title" bind:value={newAssignTitle}
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400" />
        </div>

        <div>
          <label for="assign-desc" class="block text-xs font-medium text-zinc-400 mb-1">Description</label>
          <textarea id="assign-desc" bind:value={newAssignDesc} rows="3"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400"></textarea>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="assign-deadline" class="block text-xs font-medium text-zinc-400 mb-1">Deadline</label>
            <input id="assign-deadline" type="datetime-local" bind:value={newAssignDeadline}
              class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400" />
          </div>
          <div>
            <label for="assign-format" class="block text-xs font-medium text-zinc-400 mb-1">Format</label>
            <select id="assign-format" bind:value={newAssignFormat}
              class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400">
              <option value="PDF/ZIP">PDF/ZIP</option>
              <option value="GitHub">GitHub</option>
              <option value="Text">Text</option>
            </select>
          </div>
        </div>

          <div>
            <label for="assign-rubric" class="block text-xs font-medium text-zinc-400 mb-1">Rubric</label>
            <select id="assign-rubric" bind:value={newAssignRubricId}
              class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400">
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
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" on:click={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-md w-full p-5 space-y-4 shadow-xl" on:click|stopPropagation>
        <h3 class="text-sm font-semibold text-zinc-100">{editingRubricId ? 'Edit Rubric' : 'Create Rubric'}</h3>

        <div>
          <label for="rubric-title" class="block text-xs font-medium text-zinc-400 mb-1">Title</label>
          <input id="rubric-title" bind:value={rubricTitle}
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400" />
        </div>

        <div>
          <label for="rubric-department" class="block text-xs font-medium text-zinc-400 mb-1">Department</label>
          <input id="rubric-department" bind:value={rubricDepartment} placeholder="Computer Science"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400" />
        </div>

        <div>
          <label for="rubric-description" class="block text-xs font-medium text-zinc-400 mb-1">Description</label>
          <textarea id="rubric-description" bind:value={rubricDescription} rows="3"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400"></textarea>
        </div>

        <div class="border-t border-zinc-800 pt-3 space-y-2">
          <span class="block text-xs font-medium text-zinc-400">Grading Criteria</span>
          {#each rubricCriteria as crit, idx}
            <div class="flex items-center gap-2">
              <input
                bind:value={crit.title}
                placeholder="Criterion title"
                aria-label="Criterion title"
                class="flex-1 rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 placeholder:text-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-400"
              />
              <input
                type="number"
                bind:value={crit.maxPoints}
                min="0.5"
                step="0.5"
                aria-label="Max points"
                class="w-20 rounded-md border border-zinc-800 bg-zinc-900 px-2 py-1.5 text-xs text-zinc-100 font-mono focus:outline-none focus:ring-1 focus:ring-zinc-400"
              />
              <button
                type="button"
                on:click={() => removeCriterionRow(idx)}
                aria-label="Remove criterion"
                class="h-7 w-7 inline-flex items-center justify-center rounded border border-zinc-800 text-zinc-500 hover:text-red-300 hover:border-red-900/60 transition-colors"
              >
                ×
              </button>
            </div>
          {/each}
          {#if rubricCriteria.length === 0}
            <p class="text-[11px] text-zinc-600">No criteria — grading uses a single flat score.</p>
          {/if}
          <div class="flex items-center gap-2">
            <input
              bind:value={newCriterionTitle}
              placeholder="e.g. Correctness"
              aria-label="New criterion title"
              class="flex-1 rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 placeholder:text-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-400"
            />
            <input
              type="number"
              bind:value={newCriterionMax}
              min="0.5"
              step="0.5"
              aria-label="New criterion max points"
              class="w-20 rounded-md border border-zinc-800 bg-zinc-900 px-2 py-1.5 text-xs text-zinc-100 font-mono focus:outline-none focus:ring-1 focus:ring-zinc-400"
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
