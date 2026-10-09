<script lang="ts">
  import { onMount } from 'svelte';
  import Navbar from './lib/components/Navbar.svelte';
  import DashboardView from './lib/components/DashboardView.svelte';
  import CoursesView from './lib/components/CoursesView.svelte';
  import AssignmentsView from './lib/components/AssignmentsView.svelte';
  import AttendanceView from './lib/components/AttendanceView.svelte';
  import QAView from './lib/components/QAView.svelte';
  import { api, type User, type NotificationItem } from './lib/api';

  let currentUser: User | null = null;
  let activeTab: string = 'dashboard';
  let notifications: NotificationItem[] = [];
  let loadingUser = true;

  let enrollments: any[] = [];
  let selectedCourseId: number | null = null;
  let currentRole: string = '';

  // Login form state (prefilled with a seeded demo account)
  let loginEmail = 'alice.teacher@university.edu';
  let loginPassword = 'password123';
  let loginError = '';
  let loggingIn = false;

  // Profile edit state
  let showProfile = false;
  let profileFirstName = '';
  let profileLastName = '';
  let profileBio = '';
  let profileGithub = '';
  let profileLinkedin = '';
  let profilePhotos: string[] = [];
  let newPhotoUrl = '';

  $: selectedCourseCode = enrollments.find(e => e.courseid === selectedCourseId)?.coursecode ?? '';

  onMount(async () => {
    try {
      const meRes = await api.me();
      await applySession(meRes);
    } catch (err) {
      currentUser = null;
    } finally {
      loadingUser = false;
    }
  });

  async function applySession(meRes: { user: User; enrollments: any[] }) {
    currentUser = meRes.user;
    enrollments = meRes.enrollments || [];
    if (enrollments.length > 0) {
      selectedCourseId = enrollments[0].courseid;
      currentRole = enrollments[0].rolename;
    } else {
      selectedCourseId = null;
      currentRole = '';
    }
    await refreshNotifications();
  }

  async function handleLogin() {
    loginError = '';
    loggingIn = true;
    try {
      await api.login(loginEmail, loginPassword);
      const meRes = await api.me();
      await applySession(meRes);
    } catch (err: any) {
      loginError = err?.message || 'Login failed';
    } finally {
      loggingIn = false;
    }
  }

  async function handleLogout() {
    try {
      await api.logout();
    } catch (err) {}
    currentUser = null;
    enrollments = [];
    notifications = [];
    selectedCourseId = null;
    currentRole = '';
    activeTab = 'dashboard';
  }

  async function refreshNotifications() {
    if (!currentUser) return;
    try {
      notifications = await api.getNotifications(currentUser.userId);
    } catch (err) {
      console.error('Failed to load notifications', err);
    }
  }

  async function handleMarkNotificationRead(id: number) {
    try {
      await api.markNotificationRead(id, currentUser?.userId);
      await refreshNotifications();
    } catch (err) {
      console.error(err);
    }
  }

  async function handleMarkAllNotificationsRead() {
    if (!currentUser) return;
    try {
      await api.markAllNotificationsRead(currentUser.userId);
      await refreshNotifications();
    } catch (err) {
      console.error(err);
    }
  }

  function handleTabChange(tab: string) {
    activeTab = tab;
  }

  function openProfile() {
    if (!currentUser) return;
    profileFirstName = currentUser.firstName || '';
    profileLastName = currentUser.lastName || '';
    profileBio = currentUser.bio || '';
    profileGithub = currentUser.githubLink || '';
    profileLinkedin = currentUser.linkedinLink || '';
    profilePhotos = currentUser.photos || [];
    newPhotoUrl = '';
    showProfile = true;
  }

  async function addProfilePhoto() {
    if (!currentUser || !newPhotoUrl.trim()) return;
    try {
      await api.addUserPhoto(currentUser.userId, newPhotoUrl.trim());
      newPhotoUrl = '';
      const meRes = await api.me();
      currentUser = meRes.user;
      profilePhotos = meRes.user.photos || [];
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function removeProfilePhoto(url: string) {
    if (!currentUser) return;
    try {
      await api.deleteUserPhoto(currentUser.userId, url);
      const meRes = await api.me();
      currentUser = meRes.user;
      profilePhotos = meRes.user.photos || [];
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function saveProfile() {
    if (!currentUser) return;
    try {
      await api.updateProfile(currentUser.userId, {
        firstName: profileFirstName,
        lastName: profileLastName,
        bio: profileBio,
        githubLink: profileGithub,
        linkedinLink: profileLinkedin,
      });
      const meRes = await api.me();
      currentUser = meRes.user;
      showProfile = false;
    } catch (err: any) {
      alert(err.message);
    }
  }

  function handleCourseChange(courseId: number) {
    selectedCourseId = courseId;
    const enr = enrollments.find(e => e.courseid === courseId);
    if (enr) currentRole = enr.rolename;
  }
</script>

<div class="min-h-screen flex flex-col bg-zinc-950 text-zinc-100 antialiased selection:bg-zinc-800 selection:text-zinc-100">

  {#if loadingUser}
    <div class="flex-1 flex items-center justify-center text-zinc-500 text-xs font-mono">
      Connecting to MariaDB backend...
    </div>
  {:else if !currentUser}
    <!-- Login -->
    <div class="flex-1 flex items-center justify-center px-4 py-16">
      <form on:submit|preventDefault={handleLogin} class="w-full max-w-sm rounded-lg border border-zinc-800 bg-zinc-900/30 p-6 space-y-4">
        <div class="flex items-center space-x-2.5">
          <div class="w-7 h-7 rounded-md bg-zinc-100 text-zinc-950 flex items-center justify-center font-bold text-sm">M</div>
          <div>
            <div class="text-sm font-semibold tracking-tight text-zinc-100">moodle++</div>
            <div class="text-[10px] font-mono text-zinc-500 uppercase tracking-widest">Sign in</div>
          </div>
        </div>

        <div>
          <label for="login-email" class="block text-xs font-medium text-zinc-400 mb-1">Email</label>
          <input id="login-email" type="email" bind:value={loginEmail} autocomplete="username"
            class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400" />
        </div>

        <div>
          <label for="login-password" class="block text-xs font-medium text-zinc-400 mb-1">Password</label>
          <input id="login-password" type="password" bind:value={loginPassword} autocomplete="current-password"
            class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400" />
        </div>

        {#if loginError}
          <div class="p-2.5 rounded-md border border-red-900/60 bg-red-950/30 text-red-300 text-xs">
            {loginError}
          </div>
        {/if}

        <button type="submit" disabled={loggingIn}
          class="w-full rounded-md bg-zinc-100 text-zinc-950 py-2 text-xs font-medium hover:bg-zinc-200 transition-colors disabled:opacity-50">
          {loggingIn ? 'Signing in...' : 'Sign in'}
        </button>

        <p class="text-[10px] text-zinc-500 leading-relaxed">
          Demo accounts (any seeded email, password <span class="font-mono text-zinc-400">password123</span>):
          <span class="font-mono text-zinc-400">alice.teacher@university.edu</span>,
          <span class="font-mono text-zinc-400">bob.student@university.edu</span>,
          <span class="font-mono text-zinc-400">dave.ta@university.edu</span>
        </p>
      </form>
    </div>
  {:else}

  <!-- Minimal Topbar -->
  <Navbar
    {currentUser}
    {enrollments}
    {selectedCourseId}
    {activeTab}
    {currentRole}
    onTabChange={handleTabChange}
    onCourseChange={handleCourseChange}
    onLogout={handleLogout}
    onEditProfile={openProfile}
    {notifications}
    onMarkRead={handleMarkNotificationRead}
    onMarkAllRead={handleMarkAllNotificationsRead}
  />

  <!-- Main Workspace -->
  <main class="flex-1 max-w-6xl w-full mx-auto px-4 sm:px-6 py-6">
    {#if currentUser && selectedCourseId}
      {#if activeTab === 'dashboard'}
        <DashboardView {currentUser} {selectedCourseId} {currentRole} onNavigate={handleTabChange} />
      {:else if activeTab === 'courses'}
        <CoursesView {selectedCourseId} {currentRole} />
      {:else if activeTab === 'assignments'}
        <AssignmentsView {currentUser} {selectedCourseId} {currentRole} />
      {:else if activeTab === 'attendance'}
        <AttendanceView {currentUser} {selectedCourseId} {currentRole} />
      {:else if activeTab === 'qa'}
        <QAView {currentUser} {selectedCourseId} {currentRole} courseCode={selectedCourseCode} />
      {/if}
    {:else}
      <div class="flex items-center justify-center py-24 text-zinc-500 text-xs font-mono">
        No active course enrollments found.
      </div>
    {/if}
  </main>

  <!-- Minimal Understated Footer -->
  <footer class="border-t border-zinc-800/60 bg-zinc-950 py-4 text-xs text-zinc-500">
    <div class="max-w-6xl mx-auto px-4 sm:px-6 flex flex-col sm:flex-row items-center justify-between gap-2 text-[11px]">
      <div class="flex items-center space-x-2">
        <span class="font-medium text-zinc-400">Moodle++</span>
        <span>•</span>
        <span class="text-zinc-500">Learning Management System</span>
      </div>
      <div class="flex items-center space-x-2 font-mono text-[10px] text-zinc-600">
        <span>v2.0.0</span>
      </div>
    </div>
  </footer>

  {#if showProfile}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4">
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-md w-full p-5 space-y-4 shadow-xl">
        <h3 class="text-sm font-semibold text-zinc-100">Edit Profile</h3>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="profile-first" class="block text-xs font-medium text-zinc-400 mb-1">First Name</label>
            <input id="profile-first" bind:value={profileFirstName}
              class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400" />
          </div>
          <div>
            <label for="profile-last" class="block text-xs font-medium text-zinc-400 mb-1">Last Name</label>
            <input id="profile-last" bind:value={profileLastName}
              class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400" />
          </div>
        </div>

        <div>
          <label for="profile-bio" class="block text-xs font-medium text-zinc-400 mb-1">Bio</label>
          <textarea id="profile-bio" bind:value={profileBio} rows="3"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 focus:outline-none focus:ring-1 focus:ring-zinc-400"></textarea>
        </div>

        <div>
          <label for="profile-github" class="block text-xs font-medium text-zinc-400 mb-1">GitHub Link</label>
          <input id="profile-github" bind:value={profileGithub} placeholder="https://github.com/username"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 font-mono focus:outline-none focus:ring-1 focus:ring-zinc-400" />
        </div>

        <div>
          <label for="profile-linkedin" class="block text-xs font-medium text-zinc-400 mb-1">LinkedIn Link</label>
          <input id="profile-linkedin" bind:value={profileLinkedin} placeholder="https://linkedin.com/in/username"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 font-mono focus:outline-none focus:ring-1 focus:ring-zinc-400" />
        </div>

        <div class="border-t border-zinc-800 pt-3">
          <span class="block text-xs font-medium text-zinc-400 mb-1.5">Profile Photos</span>
          <div class="flex flex-wrap gap-2 mb-2.5">
            {#each profilePhotos as url}
              <div class="relative">
                <img src={url} alt="" class="h-12 w-12 rounded-md object-cover border border-zinc-800" />
                <button
                  type="button"
                  on:click={() => removeProfilePhoto(url)}
                  aria-label="Remove photo"
                  class="absolute -top-1.5 -right-1.5 h-4 w-4 rounded-full bg-zinc-950 border border-zinc-700 text-zinc-400 hover:text-red-300 text-[11px] leading-none flex items-center justify-center"
                >
                  ×
                </button>
              </div>
            {/each}
            {#if profilePhotos.length === 0}
              <span class="text-[11px] text-zinc-600">No photos uploaded yet.</span>
            {/if}
          </div>
          <div class="flex gap-2">
            <input
              id="profile-photo-url"
              bind:value={newPhotoUrl}
              placeholder="https://example.com/photo.jpg"
              class="flex-1 rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 font-mono placeholder:text-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-400"
            />
            <button
              type="button"
              on:click={addProfilePhoto}
              class="rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-300 hover:bg-zinc-800 transition-colors"
            >
              Add Photo
            </button>
          </div>
        </div>

        <div class="flex justify-end space-x-2 pt-2 border-t border-zinc-800">
          <button on:click={() => showProfile = false} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
          <button on:click={saveProfile} class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200">Save</button>
        </div>
      </div>
    </div>
  {/if}

  {/if}

</div>
