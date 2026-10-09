<script lang="ts">
  import { ArrowUpRight } from 'lucide-svelte';
  import { api, type User, type Course, type AttendanceSummary, type StudentGradeItem, type PendingStudent } from '../api';

  export let currentUser: User;
  export let selectedCourseId: number;
  export let currentRole: string;
  export let onNavigate: (tab: string) => void;

  let courses: Course[] = [];
  let attendance: AttendanceSummary | null = null;
  let grades: StudentGradeItem[] = [];
  let pendingAssignments: any[] = [];
  let pendingStudents: PendingStudent[] = [];
  let pendingGradingCount = 0;
  let enrolledStudents: any[] = [];
  let announcementMsg = '';
  let announcementStatus = '';
  let loading = true;

  $: isStudent = currentRole === 'Student';
  $: isTeacherOrTA = currentRole === 'Teacher' || currentRole === 'Teaching Assistant';

  $: if (currentUser && selectedCourseId) {
    loadDashboardData();
  }

  async function loadDashboardData() {
    loading = true;
    try {
      courses = await api.getCourses();
      if (isStudent) {
        attendance = await api.getStudentAttendance(selectedCourseId, currentUser.userId).catch(() => null);
        grades = await api.getStudentGrades(selectedCourseId, currentUser.userId).catch(() => []);
        pendingAssignments = await api.getPendingAssignmentsForStudent(currentUser.userId).catch(() => []);
      } else if (isTeacherOrTA) {
        pendingStudents = await api.getPendingStudentsForCourse(selectedCourseId).catch(() => []);
        const report = await api.getSubmissionReport(selectedCourseId).catch(() => []);
        pendingGradingCount = report.filter(r => !r.score).length;
        const enr = await api.getEnrollments(selectedCourseId).catch(() => []);
        enrolledStudents = enr.filter((e: any) => e.rolename === 'Student');
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
        enrolledStudents.map(s => api.createNotification(s.userid, message, 'Announcement'))
      );
      announcementStatus = `Announcement sent to ${enrolledStudents.length} students.`;
      announcementMsg = '';
      setTimeout(() => announcementStatus = '', 3000);
    } catch (err: any) {
      alert(err.message);
    }
  }
</script>

<div class="space-y-8">
  
  <!-- Page Header -->
  <div class="flex items-center justify-between border-b border-zinc-800 pb-5">
    <div>
      <h1 class="text-xl font-semibold tracking-tight text-zinc-100">
        {currentUser.firstName} {currentUser.lastName}
      </h1>
      <p class="text-xs text-zinc-500 mt-0.5">
        {isStudent ? 'Enrolled in Computer Science • Fall 2026' : 'Faculty Instructor • Department of Computer Science'}
      </p>
    </div>
    
    <div class="flex items-center space-x-2">
      <span class="inline-flex items-center rounded-md border border-zinc-800 bg-zinc-900/60 px-2 py-1 text-xs font-mono text-zinc-400">
        Role: {currentRole || '—'}
      </span>
    </div>
  </div>

  {#if isStudent}
    <!-- STUDENT METRICS -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      
      <!-- Attendance Card -->
      <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
        <div class="text-xs font-medium text-zinc-400">Attendance </div>
        <div class="mt-2 text-2xl font-semibold tracking-tight text-zinc-100 font-mono">
          {attendance?.percentage || '100%'}
        </div>
        <p class="text-[11px] text-zinc-500 mt-1">
          {attendance?.presentCount || 0} of {attendance?.totalSessions || 0} class sessions attended
        </p>
      </div>

      <!-- Pending Assignments -->
      <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
        <div class="text-xs font-medium text-zinc-400">Pending Submissions</div>
        <div class="mt-2 text-2xl font-semibold tracking-tight text-zinc-100 font-mono">
          {pendingAssignments.length}
        </div>
        <p class="text-[11px] text-zinc-500 mt-1">
          Requires format validation before deadline
        </p>
      </div>

      <!-- Evaluated Grade -->
      <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
        <div class="text-xs font-medium text-zinc-400">Latest Score</div>
        <div class="mt-2 text-2xl font-semibold tracking-tight text-zinc-100 font-mono">
          {grades.length > 0 ? `${grades[0].score}` : 'Pending'}
        </div>
        <p class="text-[11px] text-zinc-500 mt-1 truncate">
          {grades.length > 0 ? grades[0].assignment : 'Awaiting instructor evaluation'}
        </p>
      </div>

    </div>

    <!-- Student Pending Work Table -->
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/20 overflow-hidden">
      <div class="px-4 py-3 border-b border-zinc-800 flex items-center justify-between">
        <div>
          <h2 class="text-sm font-semibold text-zinc-100">Course Assignments</h2>
          <p class="text-xs text-zinc-500">Submissions requiring technical format verification</p>
        </div>
        <button 
          on:click={() => onNavigate('assignments')}
          class="text-xs text-zinc-400 hover:text-zinc-100 inline-flex items-center space-x-1"
        >
          <span>View all</span>
          <ArrowUpRight class="w-3.5 h-3.5" />
        </button>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-zinc-800 text-zinc-400 font-medium bg-zinc-950/40">
            <tr>
              <th class="h-9 px-4 align-middle font-medium">Course</th>
              <th class="h-9 px-4 align-middle font-medium">Assignment</th>
              <th class="h-9 px-4 align-middle font-medium">Format Required</th>
              <th class="h-9 px-4 align-middle font-medium">Deadline</th>
              <th class="h-9 px-4 align-middle font-medium text-right">Action</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-zinc-800/40 text-zinc-300">
            {#each pendingAssignments as item}
              <tr class="hover:bg-zinc-900/40 transition-colors">
                <td class="p-4 font-mono font-medium text-zinc-200">{item.coursecode}</td>
                <td class="p-4 text-zinc-100 font-medium">{item.assignmentname}</td>
                <td class="p-4 font-mono text-zinc-400">{item.formatrequired?.String || 'ANY'}</td>
                <td class="p-4 font-mono text-zinc-400">{item.deadline ? new Date(item.deadline).toISOString().split('T')[0] : 'N/A'}</td>
                <td class="p-4 text-right">
                  <button 
                    on:click={() => onNavigate('assignments')}
                    class="rounded-md bg-zinc-100 text-zinc-900 px-2.5 py-1 text-xs font-medium hover:bg-zinc-200 transition-colors"
                  >
                    Submit
                  </button>
                </td>
              </tr>
            {/each}
            {#if pendingAssignments.length === 0}
              <tr>
                <td colspan="5" class="p-6 text-center text-zinc-500">All course submissions are up to date.</td>
              </tr>
            {/if}
          </tbody>
        </table>
      </div>
    </div>

  {:else}
    <!-- INSTRUCTOR / TA METRICS -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      
      <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
        <div class="text-xs font-medium text-zinc-400">At Risk Students</div>
        <div class="mt-2 text-2xl font-semibold tracking-tight text-zinc-100 font-mono">
          {pendingStudents.length}
        </div>
        <p class="text-[11px] text-zinc-500 mt-1">
          Missing critical assignments
        </p>
      </div>

      <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
        <div class="text-xs font-medium text-zinc-400">Active Courses</div>
        <div class="mt-2 text-2xl font-semibold tracking-tight text-zinc-100 font-mono">
          {courses.length}
        </div>
        <p class="text-[11px] text-zinc-500 mt-1">
          Total courses managed
        </p>
      </div>

      <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
        <div class="text-xs font-medium text-zinc-400">Pending Grading</div>
        <div class="mt-2 text-2xl font-semibold tracking-tight text-zinc-100 font-mono">
          {pendingGradingCount}
        </div>
        <p class="text-[11px] text-zinc-500 mt-1">
          Submissions awaiting review
        </p>
      </div>

    </div>

    <!-- Post Announcement -->
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4 space-y-3">
      <div>
        <h2 class="text-sm font-semibold text-zinc-100">Post Announcement</h2>
        <p class="text-xs text-zinc-500">Sends a notification to all {enrolledStudents.length} enrolled students in this course</p>
      </div>
      <textarea
        bind:value={announcementMsg}
        rows="2"
        placeholder="e.g. Reminder: Assignment 3 is due Friday at 23:59."
        class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs text-zinc-100 placeholder:text-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-400"
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
          class="rounded-md bg-zinc-100 text-zinc-950 px-3 py-1.5 text-xs font-medium hover:bg-zinc-200 transition-colors disabled:opacity-50"
        >
          Send Announcement
        </button>
      </div>
    </div>

    <!-- Instructor Missing Submissions Table -->
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/20 overflow-hidden">
      <div class="px-4 py-3 border-b border-zinc-800 flex items-center justify-between">
        <div>
          <h2 class="text-sm font-semibold text-zinc-100">Students Missing Submissions</h2>
          <p class="text-xs text-zinc-500">Students with past due or missing assignment submissions</p>
        </div>
        <button 
          on:click={() => onNavigate('assignments')}
          class="text-xs text-zinc-400 hover:text-zinc-100 inline-flex items-center space-x-1"
        >
          <span>Grading report</span>
          <ArrowUpRight class="w-3.5 h-3.5" />
        </button>
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
            {#each pendingStudents as st}
              <tr class="hover:bg-zinc-900/40 transition-colors">
                <td class="p-4 font-mono font-medium text-zinc-200">{st.coursecode}</td>
                <td class="p-4 text-zinc-100 font-medium">{st.assignmentname}</td>
                <td class="p-4 text-zinc-300">{st.firstname} {st.lastname}</td>
                <td class="p-4 font-mono text-zinc-500">{st.email}</td>
                <td class="p-4 font-mono text-zinc-400">{new Date(st.deadline).toISOString().split('T')[0]}</td>
                <td class="p-4 text-right">
                  <span class="inline-flex items-center rounded border border-zinc-800 px-1.5 py-0.5 text-[10px] font-mono text-zinc-400">
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
