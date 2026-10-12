<script lang="ts">
  import { onMount } from 'svelte';
  import { ExternalLink, Plus, Trash2 } from 'lucide-svelte';
  import ProfileCard from './ProfileCard.svelte';
  import { backdrop } from '../dialog';
  import { toast, undoableDelete } from '../toast';
  import { downloadCSV } from '../csv';
  import {
    api, nullStr, type Course, type ResourceCategory,
    type Resource
  } from '../api';

  export let selectedCourseId: number;
  export let currentRole: string;
  export let onCoursesChanged: () => void | Promise<void> = () => {};

  let courses: Course[] = [];
  let allCourses: Course[] = [];
  let loadedFor: number = 0;
  let categories: ResourceCategory[] = [];
  let resources: Resource[] = [];
  let batches: any[] = [];
  let prerequisites: any[] = [];
  let enrollments: any[] = [];
  let allUsers: any[] = [];
  let activeTab: 'resources' | 'batches' | 'prereqs' | 'enroll' = 'resources';
  let selectedCategoryId: number | null = null;
  let loadError = '';
  let profileUserId: number | null = null;

  // New Resource Form
  let showUploadModal = false;
  let newTitle = '';
  let newFileUrl = '';
  let newCatId = 1;
  let newResBatchIds: number[] = [];

  // New Course Form
  let showCourseModal = false;
  let newCourseCode = '';
  let newCourseTitle = '';
  let newCourseDesc = '';

  // New Batch Form
  let showBatchModal = false;
  let newBatchName = '';

  // New Category Form
  let showCategoryModal = false;
  let newCategoryName = '';

  // Enrollment management
  let enrollStudentId = 0;
  let enrollRoleId = 2;
  let enrollStatusMsg = '';
  let enrollStatusErr = false;

  // Prerequisite management
  let newPrereqCourseId = 0;
  let newPrereqMinGrade = 70;

  $: isTeacher = currentRole === 'Teacher' || currentRole === 'Teaching Assistant';
  $: if (!isTeacher && activeTab === 'enroll') {
    activeTab = 'resources';
  }
  $: availablePrereqCourses = (allCourses.length > 0 ? allCourses : courses).filter(
    c => c.courseid !== selectedCourseId && !prerequisites.some(p => p.requiredcourseid === c.courseid)
  );
  $: if (availablePrereqCourses.length > 0 && !availablePrereqCourses.some(c => c.courseid === newPrereqCourseId)) {
    newPrereqCourseId = availablePrereqCourses[0].courseid;
  }

  // Single source of truth: App's selectedCourseId. Reload when it changes.
  $: if (selectedCourseId && selectedCourseId !== loadedFor) {
    loadedFor = selectedCourseId;
    selectedCategoryId = null;
    loadCourseDetails(selectedCourseId);
  }

  onMount(async () => {
    courses = await api.getCourses();
    if (isTeacher) {
      allUsers = await api.getUsers().catch(() => []);
      allCourses = await api.getCatalog().catch(() => []);
      if (allUsers.length > 0) enrollStudentId = allUsers[0].userId;
    }
  });

  async function loadCourseDetails(courseId: number) {
    loadError = '';
    try {
      categories = await api.getCategories(courseId);
      resources = await api.getResources(courseId);
      batches = await api.getBatches(courseId);
      prerequisites = await api.getPrerequisites(courseId);
      enrollments = isTeacher ? await api.getEnrollments(courseId) : [];
      if (categories.length > 0) {
        newCatId = categories[0].categoryid;
      }
      enrollStatusMsg = '';
    } catch (err: any) {
      categories = [];
      resources = [];
      batches = [];
      prerequisites = [];
      enrollments = [];
      loadError = err?.message || 'Unable to load course details for this course.';
    }
  }

  function closeModals() {
    showUploadModal = false;
    showCourseModal = false;
    showBatchModal = false;
    showCategoryModal = false;
    profileUserId = null;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') closeModals();
  }

  async function handleAddResource() {
    if (!newTitle || !newFileUrl) return;
    try {
      await api.createResource(newCatId, newTitle, newFileUrl, newResBatchIds);
      newTitle = '';
      newFileUrl = '';
      newResBatchIds = [];
      showUploadModal = false;
      resources = await api.getResources(selectedCourseId);
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function handleCreateCourse() {
    if (!newCourseCode || !newCourseTitle) return;
    try {
      await api.createCourse({ courseCode: newCourseCode, title: newCourseTitle, description: newCourseDesc });
      newCourseCode = '';
      newCourseTitle = '';
      newCourseDesc = '';
      showCourseModal = false;
      courses = await api.getCourses();
      await onCoursesChanged();
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function handleCreateBatch() {
    if (!newBatchName) return;
    try {
      await api.createBatch(selectedCourseId, newBatchName);
      newBatchName = '';
      showBatchModal = false;
      batches = await api.getBatches(selectedCourseId);
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function handleCreateCategory() {
    if (!newCategoryName) return;
    try {
      await api.createCategory(selectedCourseId, newCategoryName);
      newCategoryName = '';
      showCategoryModal = false;
      categories = await api.getCategories(selectedCourseId);
      if (categories.length > 0) newCatId = categories[0].categoryid;
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function handleEnroll() {
    enrollStatusMsg = 'Validating prerequisite grades in MariaDB...';
    enrollStatusErr = false;
    try {
      await api.enroll(selectedCourseId, enrollStudentId, enrollRoleId);
      enrollStatusMsg = 'Prerequisite requirements satisfied. Enrolled successfully.';
      enrollStatusErr = false;
      enrollments = await api.getEnrollments(selectedCourseId);
    } catch (err: any) {
      enrollStatusMsg = err.message;
      enrollStatusErr = true;
    }
  }

  async function handleUnenroll(userId: number) {
    const e = enrollments.find((x: any) => x.userid === userId);
    const cid = selectedCourseId;
    undoableDelete(`Enrollment of ${e?.firstname || ''} ${e?.lastname || ''}`.trim() || 'Enrollment', async () => {
      try {
        await api.unenroll(cid, userId);
      } catch (err) {
        if (selectedCourseId === cid) enrollments = await api.getEnrollments(cid).catch(() => enrollments);
        throw err;
      }
      if (selectedCourseId !== cid) return;
      enrollments = await api.getEnrollments(cid);
    });
  }

  // Bulk enroll: emails matched against known users, per-row report.
  let bulkEnrollText = '';
  let bulkEnrolling = false;
  let bulkEnrollReport: { email: string; ok: boolean; error?: string }[] = [];

  async function handleBulkEnroll() {
    const cid = selectedCourseId;
    const roleId = enrollRoleId;
    const emails = [...new Set(
      bulkEnrollText.split(/[\n,;]+/).map((s) => s.trim().toLowerCase()).filter(Boolean)
    )].slice(0, 500);
    if (emails.length === 0) return;
    bulkEnrolling = true;
    bulkEnrollReport = [];
    const userIdOf = (x: any): number => x.userId ?? x.userid ?? 0;
    const emailOf = (x: any): string => String(x.email || '').toLowerCase();
    for (const email of emails) {
      const u = allUsers.find((x: any) => emailOf(x) === email && userIdOf(x) > 0);
      if (!u) {
        bulkEnrollReport = [...bulkEnrollReport, { email, ok: false, error: 'unknown user' }];
        continue;
      }
      try {
        await api.enroll(cid, userIdOf(u), roleId);
        bulkEnrollReport = [...bulkEnrollReport, { email, ok: true }];
      } catch (err: any) {
        bulkEnrollReport = [...bulkEnrollReport, { email, ok: false, error: err.message }];
      }
    }
    bulkEnrolling = false;
    if (selectedCourseId === cid) enrollments = await api.getEnrollments(cid).catch(() => enrollments);
    const ok = bulkEnrollReport.filter((r) => r.ok).length;
    toast(`Enrolled ${ok}/${emails.length}`);
  }

  async function handleUpdateEnrollmentBatch(enrollmentId: number, batchId: number) {
    try {
      await api.updateEnrollmentBatch(enrollmentId, batchId || null);
      enrollments = await api.getEnrollments(selectedCourseId);
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function handleAddPrereq() {
    if (!newPrereqCourseId) return;
    try {
      await api.addPrerequisite(selectedCourseId, newPrereqCourseId, newPrereqMinGrade);
      prerequisites = await api.getPrerequisites(selectedCourseId);
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function handleDeletePrereq(requiredCourseId: number) {
    const cid = selectedCourseId;
    undoableDelete('Prerequisite link', async () => {
      try {
        await api.deletePrerequisite(cid, requiredCourseId);
      } catch (err) {
        if (selectedCourseId === cid) prerequisites = await api.getPrerequisites(cid).catch(() => prerequisites);
        throw err;
      }
      if (selectedCourseId !== cid) return;
      prerequisites = await api.getPrerequisites(cid);
    });
  }

  async function handleDeleteBatch(batchId: number) {
    const b = batches.find((x: any) => x.batchid === batchId);
    const cid = selectedCourseId;
    undoableDelete(`Batch "${b?.batchname || batchId}"`, async () => {
      try {
        await api.deleteBatch(batchId);
      } catch (err) {
        if (selectedCourseId === cid) batches = await api.getBatches(cid).catch(() => batches);
        throw err;
      }
      if (selectedCourseId !== cid) return;
      batches = await api.getBatches(cid);
      // Roster rows show batch names; refresh them too.
      if (isTeacher) enrollments = await api.getEnrollments(cid).catch(() => enrollments);
    });
  }

  async function handleDeleteResource(resourceId: number) {
    const r = resources.find((x) => x.resourceid === resourceId);
    const cid = selectedCourseId;
    undoableDelete(`Resource "${r?.title || resourceId}"`, async () => {
      try {
        await api.deleteResource(resourceId);
      } catch (err) {
        if (selectedCourseId === cid) resources = await api.getResources(cid).catch(() => resources);
        throw err;
      }
      if (selectedCourseId !== cid) return;
      resources = await api.getResources(cid);
    });
  }
</script>
<svelte:window on:keydown={handleKeydown} />

<div class="space-y-6">

  <!-- Header (course switching lives in the navbar) -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-800 pb-5">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight text-zinc-100">Course & Resource Center</h1>
      <p class="text-xs text-zinc-500 mt-0.5">Categorized materials, section batches, and recursive prerequisite validation</p>
    </div>

    {#if isTeacher}
      <button
        on:click={() => showCourseModal = true}
        class="rounded-md border border-zinc-800 bg-zinc-900/40 px-2.5 py-1.5 text-xs font-medium text-zinc-300 hover:bg-zinc-900 transition-colors inline-flex items-center space-x-1"
      >
        <Plus class="w-3.5 h-3.5" />
        <span>New Course</span>
      </button>
    {/if}
  </div>

  <!-- Selected Course Metadata -->
  {#if loadError}
    <div class="p-3 rounded-md border border-red-900/60 bg-red-950/30 text-red-300 text-xs">
      {loadError}
    </div>
  {/if}
  {#if courses.find(c => c.courseid === selectedCourseId)}
    {@const cur = courses.find(c => c.courseid === selectedCourseId)}
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-5">
      <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div class="flex items-center space-x-2">
            <span class="text-xs font-mono font-medium text-zinc-400">{cur?.coursecode}</span>
            <span class="text-zinc-500">•</span>
            <h2 class="text-sm font-semibold text-zinc-100">{cur?.title}</h2>
          </div>
          <p class="text-xs text-zinc-400 mt-1 max-w-2xl">{cur?.description?.String || 'No description.'}</p>
        </div>

        <div class="flex items-center space-x-6 text-xs border-t md:border-t-0 md:border-l border-zinc-800 pt-3 md:pt-0 md:pl-6 text-zinc-400">
          {#if isTeacher}
            <div>
              <div class="font-mono text-zinc-100 font-medium">{enrollments.length}</div>
              <div class="text-[11px] text-zinc-500">Enrolled</div>
            </div>
          {/if}
          <div>
            <div class="font-mono text-zinc-100 font-medium">{batches.length}</div>
            <div class="text-[11px] text-zinc-500">Batches</div>
          </div>
          <div>
            <div class="font-mono text-zinc-100 font-medium">{prerequisites.length}</div>
            <div class="text-[11px] text-zinc-500">Prereqs</div>
          </div>
        </div>
      </div>
    </div>
  {/if}

  <!-- ShadCN Style Tabs Bar -->
  <div class="inline-flex h-9 items-center justify-center rounded-lg bg-zinc-900/60 p-1 text-zinc-400 border border-zinc-800">
    <button
      class="rounded-md px-3 py-1 text-xs font-medium transition-colors {activeTab === 'resources' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
      on:click={() => activeTab = 'resources'}
    >
      Resources ({resources.length})
    </button>
    <button
      class="rounded-md px-3 py-1 text-xs font-medium transition-colors {activeTab === 'batches' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
      on:click={() => activeTab = 'batches'}
    >
      Batches ({batches.length})
    </button>
    <button
      class="rounded-md px-3 py-1 text-xs font-medium transition-colors {activeTab === 'prereqs' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
      on:click={() => activeTab = 'prereqs'}
    >
      Prerequisites ({prerequisites.length})
    </button>
    {#if isTeacher}
      <button
        class="rounded-md px-3 py-1 text-xs font-medium transition-colors {activeTab === 'enroll' ? 'bg-zinc-800 text-zinc-100 shadow-xs' : 'hover:text-zinc-200'}"
        on:click={() => activeTab = 'enroll'}
      >
        Enrollment Enforcement
      </button>
    {/if}
  </div>

  <!-- TAB 1: RESOURCE CENTER -->
  {#if activeTab === 'resources'}
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center space-x-1">
          <button
            class="px-2.5 py-1 rounded text-xs transition-colors {selectedCategoryId === null ? 'bg-zinc-800 text-zinc-100 font-medium' : 'text-zinc-400 hover:text-zinc-200'}"
            on:click={() => selectedCategoryId = null}
          >
            All
          </button>
          {#each categories as cat}
            <button
              class="px-2.5 py-1 rounded text-xs transition-colors {selectedCategoryId === cat.categoryid ? 'bg-zinc-800 text-zinc-100 font-medium' : 'text-zinc-400 hover:text-zinc-200'}"
              on:click={() => selectedCategoryId = cat.categoryid}
            >
              {cat.categoryname}
            </button>
          {/each}
        </div>

        {#if isTeacher}
          <div class="flex items-center space-x-2">
            <button
              on:click={() => showCategoryModal = true}
              class="rounded-md border border-zinc-800 bg-zinc-900/40 px-2.5 py-1.5 text-xs font-medium text-zinc-300 hover:bg-zinc-900 transition-colors inline-flex items-center space-x-1"
            >
              <Plus class="w-3.5 h-3.5" />
              <span>New Category</span>
            </button>
            <button
              on:click={() => showUploadModal = true}
              class="rounded-md bg-zinc-100 text-zinc-950 px-2.5 py-1.5 text-xs font-medium hover:bg-zinc-200 transition-colors inline-flex items-center space-x-1"
            >
              <Plus class="w-3.5 h-3.5" />
              <span>Upload Resource</span>
            </button>
          </div>
        {/if}
      </div>

      <div class="rounded-lg border border-zinc-800 bg-zinc-900/20 overflow-hidden divide-y divide-zinc-800/40">
        {#each (selectedCategoryId ? resources.filter(r => r.categoryid === selectedCategoryId) : resources) as res}
          <div class="p-4 flex items-center justify-between hover:bg-zinc-900/40 transition-colors">
              <div>
                <div class="flex items-center space-x-2">
                  <span class="font-mono text-[11px] text-zinc-500 uppercase px-1.5 py-0.5 rounded border border-zinc-800 bg-zinc-950">
                    {res.categoryname || 'Document'}
                  </span>
                  <span class="text-xs font-medium text-zinc-100">{res.title}</span>
                </div>
                <p class="text-[11px] text-zinc-500 font-mono mt-0.5 truncate max-w-md">{res.fileurl}</p>
                {#if nullStr(res.batchnames)}
                  <p class="text-[11px] text-zinc-400 mt-0.5">Batches: {nullStr(res.batchnames)}</p>
                {:else}
                  <p class="text-[11px] text-zinc-400 mt-0.5">Batches: All</p>
                {/if}
              </div>

            <div class="flex items-center space-x-2">
              <a
                href={res.fileurl}
                target="_blank"
                rel="noreferrer"
                aria-label="Open document"
                class="h-8 w-8 inline-flex items-center justify-center rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 transition-colors relative after:absolute after:-inset-2 after:content-['']"
              >
                <ExternalLink class="w-3.5 h-3.5" />
              </a>
              {#if isTeacher}
                <button
                  on:click={() => handleDeleteResource(res.resourceid)}
                  aria-label="Delete resource"
                  class="h-8 w-8 inline-flex items-center justify-center rounded-md border border-zinc-800 text-zinc-500 hover:text-red-300 hover:border-red-900/60 transition-colors relative after:absolute after:-inset-2 after:content-['']"
                >
                  <Trash2 class="w-3.5 h-3.5" />
                </button>
              {/if}
            </div>
          </div>
        {/each}
        {#if resources.length === 0}
          <div class="p-8 text-center text-xs text-zinc-500">No resources cataloged yet.</div>
        {/if}
      </div>
    </div>
  {/if}

  <!-- TAB 2: BATCHES -->
  {#if activeTab === 'batches'}
    <div class="space-y-4">
      {#if isTeacher}
        <div class="flex justify-end">
          <button
            on:click={() => showBatchModal = true}
            class="rounded-md bg-zinc-100 text-zinc-950 px-2.5 py-1.5 text-xs font-medium hover:bg-zinc-200 transition-colors inline-flex items-center space-x-1"
          >
            <Plus class="w-3.5 h-3.5" />
            <span>New Batch</span>
          </button>
        </div>
      {/if}
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        {#each batches as b}
          <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
            <div class="flex items-center justify-between text-xs">
              <span class="font-mono text-zinc-500">Batch #{b.batchid}</span>
              <div class="flex items-center space-x-1.5">
                <span class="inline-flex items-center rounded border border-zinc-800 px-1.5 py-0.2 text-[11px] font-mono text-zinc-400">Active</span>
                {#if isTeacher}
                  <button
                    on:click={() => handleDeleteBatch(b.batchid)}
                    aria-label="Delete batch"
                    class="h-6 w-6 inline-flex items-center justify-center rounded border border-zinc-800 text-zinc-500 hover:text-red-300 hover:border-red-900/60 transition-colors relative after:absolute after:-inset-3 after:content-['']"
                  >
                    <Trash2 class="w-3 h-3" />
                  </button>
                {/if}
              </div>
            </div>
            <div class="mt-2 text-sm font-semibold text-zinc-100">{b.batchname}</div>
            <p class="text-[11px] text-zinc-500 mt-1">Class & lab section assignment</p>
          </div>
        {/each}
        {#if batches.length === 0}
          <div class="sm:col-span-3 rounded-lg border border-zinc-800/60 bg-zinc-950/40 p-8 text-center text-xs text-zinc-500">
            No batches configured for this course.
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <!-- TAB 3: PREREQUISITES -->
  {#if activeTab === 'prereqs'}
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-5 space-y-4">
      <div>
        <h3 class="text-sm font-semibold text-zinc-100">Prerequisite Requirements</h3>
        <p class="text-xs text-zinc-500">Courses required before enrollment</p>
      </div>

      {#if isTeacher}
        <div class="flex flex-wrap items-end gap-3 pt-1">
          <div class="flex-1 min-w-[10rem]">
            <label for="prereq-course" class="block text-xs font-medium text-zinc-400 mb-1">Required Course</label>
            <select
              id="prereq-course"
              bind:value={newPrereqCourseId}
              disabled={availablePrereqCourses.length === 0}
              class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100  disabled:opacity-50"
            >
              {#each availablePrereqCourses as c}
                <option value={c.courseid}>{c.coursecode} — {c.title}</option>
              {/each}
              {#if availablePrereqCourses.length === 0}
                <option value={0}>No eligible courses</option>
              {/if}
            </select>
          </div>
          <div class="w-32">
            <label for="prereq-grade" class="block text-xs font-medium text-zinc-400 mb-1">Min Grade (%)</label>
            <input
              id="prereq-grade"
              type="number"
              min="0"
              max="100"
              bind:value={newPrereqMinGrade}
              class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 "
            />
          </div>
          <button
            on:click={handleAddPrereq}
            disabled={availablePrereqCourses.length === 0}
            class="rounded-md bg-zinc-100 text-zinc-950 px-3 py-1.5 text-xs font-medium hover:bg-zinc-200 transition-colors disabled:opacity-50"
          >
            Add Prerequisite
          </button>
        </div>
      {/if}

      {#if prerequisites.length === 0}
        <div class="text-xs text-zinc-500 py-4">
          No prerequisites configured for this course.
        </div>
      {:else}
        <div class="divide-y divide-zinc-800/40 border-t border-zinc-800/60">
          {#each prerequisites as p}
            <div class="p-3.5 flex items-center justify-between text-xs">
              <div>
                <span class="font-medium text-zinc-200">Requires {p.requiredcoursecode}</span>
                <span class="text-zinc-500 ml-1.5">({p.requiredcoursetitle})</span>
              </div>
              <div class="flex items-center space-x-3">
                <div class="font-mono text-zinc-400">
                  Minimum Grade: <strong class="text-zinc-100">{p.minpassinggrade?.String || '70.00'}%</strong>
                </div>
                {#if isTeacher}
                  <button
                    on:click={() => handleDeletePrereq(p.requiredcourseid)}
                    aria-label="Remove prerequisite"
                    class="h-6 w-6 inline-flex items-center justify-center rounded border border-zinc-800 text-zinc-500 hover:text-red-300 hover:border-red-900/60 transition-colors relative after:absolute after:-inset-3 after:content-['']"
                  >
                    <Trash2 class="w-3 h-3" />
                  </button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}

  <!-- TAB 4: ENROLLMENTS -->
  {#if activeTab === 'enroll'}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      
      <!-- Current Enrollments -->
      <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-5">
        <div class="flex items-center justify-between mb-1">
          <h3 class="text-sm font-semibold text-zinc-100">Enrolled Students & Faculty</h3>
          <button
            on:click={() => {
              downloadCSV(`roster-${selectedCourseId}.csv`, [
                ['Name', 'Email', 'Role', 'Batch'],
                ...enrollments.map((e: any) => [
                  `${e.firstname} ${e.lastname}`, e.email, e.rolename,
                  e.batchname?.Valid ? e.batchname.String : '',
                ]),
              ]);
              toast(`Exported ${enrollments.length} rows`);
            }}
            class="rounded-md border border-zinc-800 bg-zinc-900 px-2 py-1 text-[11px] font-mono text-zinc-400 hover:text-zinc-100 transition-colors"
          >
            CSV
          </button>
        </div>
        <p class="text-xs text-zinc-500 mb-4">Course roster</p>

        <div class="divide-y divide-zinc-800/40 max-h-72 overflow-y-auto">
          {#each enrollments as e}
            <div class="py-2.5 flex items-center justify-between text-xs">
              <div>
                <button
                  class="font-medium text-zinc-200 hover:text-zinc-100 hover:underline underline-offset-2"
                  on:click={() => { profileUserId = e.userid; }}
                >
                  {e.firstname} {e.lastname}
                </button>
                <span class="text-zinc-500 font-mono text-[11px] ml-1">({e.email})</span>
              </div>
              <div class="flex items-center space-x-1.5">
                {#if isTeacher && e.rolename === 'Student'}
                  <select
                    value={e.batchid?.Valid ? e.batchid.Int32 : 0}
                    on:change={(ev) => handleUpdateEnrollmentBatch(e.enrollmentid, Number((ev.currentTarget as HTMLSelectElement).value))}
                    aria-label="Assign batch"
                    class="rounded border border-zinc-800 bg-zinc-950 px-1.5 py-0.5 text-[11px] font-mono text-zinc-300"
                  >
                    <option value={0}>Unassigned</option>
                    {#each batches as b}
                      <option value={b.batchid}>{b.batchname}</option>
                    {/each}
                  </select>
                {:else if e.batchname?.Valid}
                  <span class="font-mono text-[11px] text-zinc-400 px-1.5 py-0.5 rounded border border-zinc-800">
                    {e.batchname.String}
                  </span>
                {/if}
                <span class="font-mono text-[11px] text-zinc-400 px-1.5 py-0.5 rounded border border-zinc-800">
                  {e.rolename}
                </span>
                {#if isTeacher}
                  <button
                    on:click={() => handleUnenroll(e.userid)}
                    aria-label="Remove enrollment"
                    class="h-6 w-6 inline-flex items-center justify-center rounded border border-zinc-800 text-zinc-500 hover:text-red-300 hover:border-red-900/60 transition-colors relative after:absolute after:-inset-3 after:content-['']"
                  >
                    <Trash2 class="w-3 h-3" />
                  </button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      </div>

      <!-- Live Prerequisite Validator -->
      <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-5 space-y-4">
        <div>
          <h3 class="text-sm font-semibold text-zinc-100">Prerequisite Enforcement Test</h3>
          <p class="text-xs text-zinc-500">Evaluates grade history before committing enrollment to MariaDB</p>
        </div>

        <div>
          <label for="enroll-student-select" class="block text-xs font-medium text-zinc-400 mb-1.5">Select Student Candidate</label>
          <select 
            id="enroll-student-select"
            bind:value={enrollStudentId} 
            class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-xs text-zinc-100 "
          >
            {#each allUsers as u}
              <option value={u.userId}>{u.firstName} {u.lastName} ({u.email})</option>
            {/each}
          </select>
        </div>

        <div>
          <label for="enroll-role-select" class="block text-xs font-medium text-zinc-400 mb-1.5">Enroll As</label>
          <select
            id="enroll-role-select"
            bind:value={enrollRoleId}
            class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-xs text-zinc-100 "
          >
            <option value={2}>Student</option>
            <option value={3}>Teaching Assistant</option>
            <option value={1}>Teacher</option>
          </select>
        </div>

        <button
          on:click={handleEnroll}
          class="w-full rounded-md bg-zinc-100 text-zinc-950 py-2 text-xs font-medium hover:bg-zinc-200 transition-colors"
        >
          Validate & Attempt Enrollment
        </button>

        {#if enrollStatusMsg}
          <div class="p-3 rounded-md border text-xs {enrollStatusErr ? 'border-red-900/60 bg-red-950/30 text-red-300' : 'border-emerald-900/60 bg-emerald-950/30 text-emerald-300'}">
            {enrollStatusMsg}
          </div>
        {/if}
      </div>

      <!-- Bulk Enroll via CSV / email list -->
      <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-5 space-y-4">
        <div>
          <h3 class="text-sm font-semibold text-zinc-100">Bulk Enroll</h3>
          <p class="text-xs text-zinc-500">Paste emails (one per line or comma-separated). Matched against known users, enrolled as the role above. Per-row report below.</p>
        </div>
        <textarea
          bind:value={bulkEnrollText}
          rows="4"
          placeholder={"bob.student@university.edu\nemma.student@university.edu"}
          class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs font-mono text-zinc-100 placeholder:text-zinc-600"
        ></textarea>
        <button
          on:click={handleBulkEnroll}
          disabled={!bulkEnrollText.trim() || bulkEnrolling}
          class="w-full rounded-md bg-zinc-100 text-zinc-950 py-2 text-xs font-medium hover:bg-zinc-200 transition-colors disabled:opacity-50"
        >
          {bulkEnrolling ? 'Enrolling…' : 'Enroll List'}
        </button>
        {#if bulkEnrollReport.length > 0}
          <div class="max-h-40 overflow-y-auto divide-y divide-zinc-800/40 rounded-md border border-zinc-800 font-mono text-[11px]">
            {#each bulkEnrollReport as row}
              <div class="px-2.5 py-1.5 {row.ok ? 'text-emerald-300' : 'text-red-300'}">
                {row.ok ? '✓' : '✗'} {row.email}{row.ok ? '' : ` — ${row.error}`}
              </div>
            {/each}
          </div>
        {/if}
      </div>

    </div>
  {/if}

  <!-- Upload Modal -->
  {#if showUploadModal}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" use:backdrop={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-sm w-full p-5 space-y-4 shadow-xl">
        <h3 class="text-sm font-semibold text-zinc-100">Upload Learning Resource</h3>
        
        <div>
          <label for="upload-category" class="block text-xs font-medium text-zinc-400 mb-1">Category</label>
          <select 
            id="upload-category"
            bind:value={newCatId} 
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100"
          >
            {#each categories as cat}
              <option value={cat.categoryid}>{cat.categoryname}</option>
            {/each}
          </select>
        </div>

        <div>
          <label for="upload-title" class="block text-xs font-medium text-zinc-400 mb-1">Title</label>
          <input 
            id="upload-title"
            bind:value={newTitle} 
            placeholder="e.g. Relational Normalization Slides" 
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 placeholder:text-zinc-500 " 
          />
        </div>

        <div>
          <label for="upload-url" class="block text-xs font-medium text-zinc-400 mb-1">File URL</label>
          <input 
            id="upload-url"
            bind:value={newFileUrl} 
            placeholder="https://..." 
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 font-mono placeholder:text-zinc-500 " 
          />
        </div>

        <div>
          <span class="block text-xs font-medium text-zinc-400 mb-1">Target Batches</span>
          <div class="flex flex-wrap gap-2">
            {#each batches as b}
              <label class="inline-flex items-center gap-1 text-xs text-zinc-300">
                <input type="checkbox" value={b.batchid} bind:group={newResBatchIds} />
                {b.batchname}
              </label>
            {/each}
          </div>
          <p class="text-[11px] text-zinc-500 mt-1">Leave all unchecked to make it course-wide.</p>
        </div>

        <div class="flex justify-end space-x-2 pt-2 border-t border-zinc-800">
          <button on:click={() => showUploadModal = false} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
          <button on:click={handleAddResource} class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200">Upload</button>
        </div>
      </div>
    </div>
  {/if}

  <!-- New Course Modal -->
  {#if showCourseModal}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" use:backdrop={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-sm w-full p-5 space-y-4 shadow-xl">
        <h3 class="text-sm font-semibold text-zinc-100">Create Course</h3>
        <div>
          <label for="course-code" class="block text-xs font-medium text-zinc-400 mb-1">Course Code</label>
          <input id="course-code" bind:value={newCourseCode} placeholder="CS701"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 font-mono " />
        </div>
        <div>
          <label for="course-title" class="block text-xs font-medium text-zinc-400 mb-1">Title</label>
          <input id="course-title" bind:value={newCourseTitle} placeholder="Advanced Topics"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 " />
        </div>
        <div>
          <label for="course-desc" class="block text-xs font-medium text-zinc-400 mb-1">Description</label>
          <textarea id="course-desc" bind:value={newCourseDesc} rows="3"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 "></textarea>
        </div>
        <div class="flex justify-end space-x-2 pt-2 border-t border-zinc-800">
          <button on:click={() => showCourseModal = false} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
          <button on:click={handleCreateCourse} class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200">Create</button>
        </div>
      </div>
    </div>
  {/if}

  <!-- New Batch Modal -->
  {#if showBatchModal}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" use:backdrop={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-sm w-full p-5 space-y-4 shadow-xl">
        <h3 class="text-sm font-semibold text-zinc-100">Create Batch</h3>
        <div>
          <label for="batch-name" class="block text-xs font-medium text-zinc-400 mb-1">Batch Name</label>
          <input id="batch-name" bind:value={newBatchName} placeholder="Batch C - CS101"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 " />
        </div>
        <div class="flex justify-end space-x-2 pt-2 border-t border-zinc-800">
          <button on:click={() => showBatchModal = false} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
          <button on:click={handleCreateBatch} class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200">Create</button>
        </div>
      </div>
    </div>
  {/if}

  <!-- New Category Modal -->
  {#if showCategoryModal}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" use:backdrop={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-sm w-full p-5 space-y-4 shadow-xl">
        <h3 class="text-sm font-semibold text-zinc-100">Create Category</h3>
        <div>
          <label for="category-name" class="block text-xs font-medium text-zinc-400 mb-1">Category Name</label>
          <input id="category-name" bind:value={newCategoryName} placeholder="Lab Manuals"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 " />
        </div>
        <div class="flex justify-end space-x-2 pt-2 border-t border-zinc-800">
          <button on:click={() => showCategoryModal = false} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
          <button on:click={handleCreateCategory} class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200">Create</button>
        </div>
      </div>
    </div>
  {/if}

  <ProfileCard userId={profileUserId} onClose={() => (profileUserId = null)} />

</div>
