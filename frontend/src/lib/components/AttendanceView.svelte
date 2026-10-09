<script lang="ts">
  import { api, type AttendanceSummary, type User } from '../api';

  export let currentUser: User;
  export let selectedCourseId: number;
  export let currentRole: string;

  let attendance: AttendanceSummary | null = null;
  let sessions: any[] = [];
  let students: any[] = [];
  let batches: any[] = [];
  let selectedSessionId: number = 0;
  let selectedStudentId: number = 0;
  let selectedStatus: string = 'Present';
  let statusUpdateMsg: string = '';

  // New session form
  let showSessionModal = false;
  let newSessionBatchId = 0;
  let newSessionStart = '';
  let newSessionEnd = '';
  let newSessionRoom = '';

  // Bulk attendance form
  let showBulkModal = false;
  let bulkStatus = 'Present';

  $: isStudent = currentRole === 'Student';
  $: isTeacher = currentRole === 'Teacher' || currentRole === 'Teaching Assistant';

  $: if (currentUser && selectedCourseId) {
    if (isStudent) {
      selectedStudentId = currentUser.userId;
    }
    loadAttendanceData();
  }

  async function loadAttendanceData() {
    if (!selectedCourseId) return;
    sessions = await api.getCourseSessions(selectedCourseId);
    if (sessions.length > 0 && selectedSessionId === 0) {
      selectedSessionId = sessions[0].sessionid || sessions[0].SessionID;
    }

    if (isTeacher) {
      const allEnrollments = await api.getEnrollments(selectedCourseId);
      students = allEnrollments.filter((e: any) => e.rolename === 'Student' || e.RoleName === 'Student');
      if (students.length > 0 && selectedStudentId === 0) {
        selectedStudentId = students[0].userid || students[0].UserID;
      }
      batches = await api.getBatches(selectedCourseId).catch(() => []);
      if (newSessionBatchId === 0 && batches.length > 0) {
        newSessionBatchId = batches[0].batchid;
      }
    }

    if (selectedStudentId) {
      attendance = await api.getStudentAttendance(selectedCourseId, selectedStudentId);
    }
  }

  function closeModals() {
    showSessionModal = false;
    showBulkModal = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') closeModals();
  }

  async function handleCreateSession() {
    if (!newSessionBatchId || !newSessionStart || !newSessionEnd) return;
    try {
      await api.createSession(
        newSessionBatchId,
        new Date(newSessionStart).toISOString(),
        new Date(newSessionEnd).toISOString(),
        newSessionRoom
      );
      newSessionStart = '';
      newSessionEnd = '';
      newSessionRoom = '';
      showSessionModal = false;
      selectedSessionId = 0;
      await loadAttendanceData();
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function handleMarkAttendance() {
    try {
      await api.recordAttendance(selectedSessionId, selectedStudentId, selectedStatus);
      statusUpdateMsg = `Recorded ${selectedStatus} for session #${selectedSessionId}.`;
      await loadAttendanceData();
      setTimeout(() => statusUpdateMsg = '', 2000);
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function handleBulkAttendance() {
    if (!selectedSessionId || students.length === 0) return;
    try {
      const records = students.map(st => ({ userId: st.userid || st.UserID, status: bulkStatus }));
      await api.recordBulkAttendance(selectedSessionId, records);
      showBulkModal = false;
      statusUpdateMsg = `Bulk recorded ${bulkStatus} for ${records.length} students in session #${selectedSessionId}.`;
      await loadAttendanceData();
      setTimeout(() => statusUpdateMsg = '', 2500);
    } catch (err: any) {
      alert(err.message);
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="space-y-6">

  <!-- Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-800 pb-5">
    <div>
      <h1 class="text-xl font-semibold tracking-tight text-zinc-100">Timetable & Attendance</h1>
      <p class="text-xs text-zinc-500 mt-0.5">{isStudent ? 'Your attendance record' : 'Manage and track student class attendance'}</p>
    </div>

    {#if !isStudent}
      <div class="flex items-center space-x-2">
        <label for="inspect-student-select" class="text-xs text-zinc-500">Inspecting:</label>
        <select
          id="inspect-student-select"
          bind:value={selectedStudentId}
          on:change={loadAttendanceData}
          class="rounded-md border border-zinc-800 bg-zinc-950 px-2.5 py-1 text-xs text-zinc-200"
        >
          {#each students as st}
            <option value={st.userid || st.UserID}>{st.firstname || st.FirstName} {st.lastname || st.LastName}</option>
          {/each}
        </select>
        {#if isTeacher}
          <button
            on:click={() => showBulkModal = true}
            class="rounded-md border border-zinc-800 bg-zinc-900/40 px-2.5 py-1.5 text-xs font-medium text-zinc-300 hover:bg-zinc-900 transition-colors"
          >
            Bulk Mark
          </button>
          <button
            on:click={() => showSessionModal = true}
            class="rounded-md bg-zinc-100 text-zinc-950 px-2.5 py-1.5 text-xs font-medium hover:bg-zinc-200 transition-colors"
          >
            New Session
          </button>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Metric Row -->
  <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
    
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
      <div class="text-xs font-medium text-zinc-400">Attendance Rate</div>
      <div class="mt-2 text-2xl font-semibold tracking-tight text-zinc-100 font-mono">
        {attendance?.percentage || '0%'}
      </div>
      <div class="w-full bg-zinc-800 h-1.5 rounded-full mt-2.5 overflow-hidden">
        <div class="bg-zinc-100 h-full rounded-full" style="width: {attendance?.percentageRaw || 0}%"></div>
      </div>
    </div>

    <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
      <div class="text-xs font-medium text-zinc-400">Present</div>
      <div class="mt-2 text-2xl font-semibold tracking-tight text-zinc-100 font-mono">
        {attendance?.presentCount || 0}
      </div>
      <p class="text-[11px] text-zinc-500 mt-1 font-mono">of {attendance?.totalSessions || 0} sessions</p>
    </div>

    <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
      <div class="text-xs font-medium text-zinc-400">Absent</div>
      <div class="mt-2 text-2xl font-semibold tracking-tight text-zinc-100 font-mono">
        {attendance?.absentCount || 0}
      </div>
      <p class="text-[11px] text-zinc-500 mt-1 font-mono">unexcused</p>
    </div>

    <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
      <div class="text-xs font-medium text-zinc-400">Excused</div>
      <div class="mt-2 text-2xl font-semibold tracking-tight text-zinc-100 font-mono">
        {attendance?.excusedCount || 0}
      </div>
      <p class="text-[11px] text-zinc-500 mt-1 font-mono">authorized</p>
    </div>

  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">

    <!-- Schedule Table -->
    <div class="lg:col-span-2 rounded-lg border border-zinc-800 bg-zinc-900/20 overflow-hidden">
      <div class="p-4 border-b border-zinc-800">
        <h3 class="text-sm font-semibold text-zinc-100">Scheduled Class Sessions</h3>
        <p class="text-xs text-zinc-500 font-mono">Linked to Batch & ClassSession(StartTime, EndTime, RoomLocation)</p>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-zinc-800 text-zinc-400 font-medium bg-zinc-950/40">
            <tr>
              <th class="h-9 px-4 align-middle font-medium">Session ID</th>
              <th class="h-9 px-4 align-middle font-medium">Group</th>
              <th class="h-9 px-4 align-middle font-medium">Schedule</th>
              <th class="h-9 px-4 align-middle font-medium">Room</th>
              <th class="h-9 px-4 align-middle font-medium text-right">Status</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-zinc-800/40 text-zinc-300">
            {#each (attendance?.sessions || []) as s}
              <tr class="hover:bg-zinc-900/40 transition-colors">
                <td class="p-4 font-mono text-zinc-500">#{s.sessionId}</td>
                <td class="p-4 font-medium text-zinc-200">{s.batchName}</td>
                <td class="p-4 font-mono text-zinc-400">{s.startTime} - {s.endTime}</td>
                <td class="p-4 font-mono text-zinc-500">{s.roomLocation || 'Online'}</td>
                <td class="p-4 text-right">
                  <span class="inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-mono {s.attendanceStatus === 'Present' ? 'border-zinc-700 bg-zinc-800/40 text-zinc-200' : s.attendanceStatus === 'Absent' ? 'border-red-900/60 bg-red-950/30 text-red-400' : 'border-zinc-800 text-zinc-500'}">
                    {s.attendanceStatus}
                  </span>
                </td>
              </tr>
            {/each}
            {#if !attendance?.sessions || attendance.sessions.length === 0}
              <tr>
                <td colspan="5" class="p-6 text-center text-zinc-500">No scheduled sessions.</td>
              </tr>
            {/if}
          </tbody>
        </table>
      </div>
    </div>

    <!-- Instructor Attendance Recorder -->
    {#if isTeacher}
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-5 space-y-4">
      <div>
        <h3 class="text-sm font-semibold text-zinc-100">Record Attendance</h3>
        <p class="text-xs text-zinc-500">Directly executes against MariaDB Attendance table</p>
      </div>

      <div>
        <label for="att-session-select" class="block text-xs font-medium text-zinc-400 mb-1">Session</label>
        <select 
          id="att-session-select"
          bind:value={selectedSessionId} 
          class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-2.5 py-1.5 text-xs text-zinc-100"
        >
          {#each sessions as ses}
            <option value={ses.sessionid}>
              #{ses.sessionid} - {ses.batchname} ({new Date(ses.starttime).toISOString().split('T')[0]})
            </option>
          {/each}
        </select>
      </div>

      <div>
        <label for="att-student-select" class="block text-xs font-medium text-zinc-400 mb-1">Student</label>
        <select 
          id="att-student-select"
          bind:value={selectedStudentId} 
          class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-2.5 py-1.5 text-xs text-zinc-100"
        >
          {#each students as st}
            <option value={st.userid || st.UserID}>{st.firstname || st.FirstName} {st.lastname || st.LastName}</option>
          {/each}
        </select>
      </div>

      <div>
        <span class="block text-xs font-medium text-zinc-400 mb-1.5">Status (CHECK constraint)</span>
        <div class="grid grid-cols-3 gap-1.5">
          {#each ['Present', 'Absent', 'Excused'] as st}
            <button
              class="py-1.5 text-xs font-medium rounded-md border transition-colors {selectedStatus === st ? 'bg-zinc-100 text-zinc-950 border-zinc-100' : 'border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-zinc-200'}"
              on:click={() => selectedStatus = st}
            >
              {st}
            </button>
          {/each}
        </div>
      </div>

      <button
        on:click={handleMarkAttendance}
        class="w-full rounded-md bg-zinc-100 text-zinc-950 py-2 text-xs font-medium hover:bg-zinc-200 transition-colors"
      >
        Update Attendance
      </button>

      {#if statusUpdateMsg}
        <div class="p-2.5 rounded-md border border-zinc-700 bg-zinc-900/60 text-zinc-200 text-xs text-center font-mono">
          {statusUpdateMsg}
        </div>
      {/if}
    </div>
    {/if}

  </div>

  <!-- New Session Modal -->
  {#if showSessionModal}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" on:click={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-sm w-full p-5 space-y-4 shadow-xl" on:click|stopPropagation>
        <h3 class="text-sm font-semibold text-zinc-100">Schedule Class Session</h3>

        <div>
          <label for="session-batch" class="block text-xs font-medium text-zinc-400 mb-1">Batch</label>
          <select id="session-batch" bind:value={newSessionBatchId}
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400"
          >
            {#each batches as b}
              <option value={b.batchid}>{b.batchname}</option>
            {/each}
          </select>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="session-start" class="block text-xs font-medium text-zinc-400 mb-1">Start</label>
            <input id="session-start" type="datetime-local" bind:value={newSessionStart}
              class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400" />
          </div>
          <div>
            <label for="session-end" class="block text-xs font-medium text-zinc-400 mb-1">End</label>
            <input id="session-end" type="datetime-local" bind:value={newSessionEnd}
              class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400" />
          </div>
        </div>

        <div>
          <label for="session-room" class="block text-xs font-medium text-zinc-400 mb-1">Room</label>
          <input id="session-room" bind:value={newSessionRoom} placeholder="Room 301"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400" />
        </div>

        <div class="flex justify-end space-x-2 pt-2 border-t border-zinc-800">
          <button on:click={() => showSessionModal = false} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
          <button on:click={handleCreateSession} class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200">Create</button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Bulk Attendance Modal -->
  {#if showBulkModal}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" on:click={closeModals}>
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-sm w-full p-5 space-y-4 shadow-xl" on:click|stopPropagation>
        <h3 class="text-sm font-semibold text-zinc-100">Bulk Attendance</h3>
        <p class="text-xs text-zinc-500">Apply one status to all {students.length} enrolled students.</p>

        <div>
          <label for="bulk-session-select" class="block text-xs font-medium text-zinc-400 mb-1">Session</label>
          <select
            id="bulk-session-select"
            bind:value={selectedSessionId}
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400"
          >
            {#each sessions as ses}
              <option value={ses.sessionid}>#{ses.sessionid} - {ses.batchname} ({new Date(ses.starttime).toISOString().split('T')[0]})</option>
            {/each}
          </select>
        </div>

        <div>
          <span class="block text-xs font-medium text-zinc-400 mb-1.5">Status</span>
          <div class="grid grid-cols-3 gap-1.5">
            {#each ['Present', 'Absent', 'Excused'] as st}
              <button
                class="py-1.5 text-xs font-medium rounded-md border transition-colors {bulkStatus === st ? 'bg-zinc-100 text-zinc-950 border-zinc-100' : 'border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-zinc-200'}"
                on:click={() => bulkStatus = st}
              >
                {st}
              </button>
            {/each}
          </div>
        </div>

        <div class="flex justify-end space-x-2 pt-2 border-t border-zinc-800">
          <button on:click={() => showBulkModal = false} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
          <button
            on:click={handleBulkAttendance}
            disabled={students.length === 0 || !selectedSessionId}
            class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200 disabled:opacity-50"
          >
            Apply to All
          </button>
        </div>
      </div>
    </div>
  {/if}

</div>
