<script lang="ts">
  import { onMount } from 'svelte';
  import { Bell, ChevronDown, GraduationCap, LogOut, RefreshCw, User } from 'lucide-svelte';
  import type { User as UserType, NotificationItem } from '../api';
  import { timeAgo } from '../time';

  export let currentUser: UserType | null;
  export let enrollments: any[] = [];
  export let selectedCourseId: number | null;
  export let currentRole: string;
  export let activeTab: string;
  export let onTabChange: (tab: string) => void;
  export let onCourseChange: (courseId: number) => void;
  export let onLogout: () => void;
  export let onEditProfile: () => void;
  export let notifications: NotificationItem[] = [];
  export let onMarkRead: (id: number) => void;
  export let onMarkAllRead: () => void = () => {};
  export let onRefresh: () => void = () => {};
  export let onOpen: (n: NotificationItem) => void = (n) => onMarkRead(n.notificationid);

  let showUserDropdown = false;
  let showNotifDropdown = false;
  let showCourseDropdown = false;
  let notifTab: 'all' | 'unread' = 'all';
  let notifType = '';

  onMount(() => {
    const closeAll = (e: MouseEvent) => {
      if ((e.target as Element)?.closest('[data-dropdown]')) return;
      showUserDropdown = false;
      showNotifDropdown = false;
      showCourseDropdown = false;
    };
    window.addEventListener('click', closeAll);
    return () => window.removeEventListener('click', closeAll);
  });

  $: unreadCount = (notifications || []).filter(n => n.isread?.Bool === false).length;
  $: activeCourse = enrollments.find(e => e.courseid === selectedCourseId);
  $: notifTypes = [...new Set((notifications || []).map((n) => n.type?.String || 'Alert'))].sort();
  // Type filter resets when its option vanishes (else empty list, no cue).
  $: if (notifType && !notifTypes.includes(notifType)) notifType = '';
  $: visibleNotifs = (notifications || []).filter((n) =>
    (notifTab === 'all' || n.isread?.Bool === false) &&
    (!notifType || (n.type?.String || 'Alert') === notifType)
  );
  // "Catch me up": one-line digest of unread by type.
  $: digest = (() => {
    const counts = new Map<string, number>();
    (notifications || []).filter((n) => n.isread?.Bool === false).forEach((n) => {
      const t = (n.type?.String || 'Alert').toLowerCase();
      counts.set(t, (counts.get(t) || 0) + 1);
    });
    return [...counts.entries()].map(([t, c]) => `${c} ${t}${c > 1 ? 's' : ''}`).join(' • ');
  })();
</script>

<header class="border-b border-zinc-800 bg-zinc-950 sticky top-0 z-50">
  <div class="max-w-6xl mx-auto px-4 sm:px-6">
    <div class="flex items-center justify-between h-14">
      
      <!-- Brand & Tabs -->
      <div class="flex items-center space-x-6">
        <button 
          class="flex items-center space-x-2.5 text-left group" 
          on:click={() => onTabChange('dashboard')}
        >
          <div class="w-6 h-6 rounded-md bg-zinc-100 text-zinc-950 flex items-center justify-center font-bold text-xs">
            M
          </div>
          <span class="text-sm font-semibold tracking-tight text-zinc-100">moodle++</span>
        </button>

        <!-- Understated Tabs -->
        <nav class="hidden md:flex items-center space-x-1 text-xs">
          <button
            class="px-2.5 py-1.5 rounded-md transition-colors {activeTab === 'dashboard' ? 'bg-zinc-800 text-zinc-100 font-medium' : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'}"
            on:click={() => onTabChange('dashboard')}
          >
            Dashboard
          </button>
          <button
            class="px-2.5 py-1.5 rounded-md transition-colors {activeTab === 'courses' ? 'bg-zinc-800 text-zinc-100 font-medium' : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'}"
            on:click={() => onTabChange('courses')}
          >
            Courses & Resources
          </button>
          <button
            class="px-2.5 py-1.5 rounded-md transition-colors {activeTab === 'assignments' ? 'bg-zinc-800 text-zinc-100 font-medium' : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'}"
            on:click={() => onTabChange('assignments')}
          >
            Assignments
          </button>
          <button
            class="px-2.5 py-1.5 rounded-md transition-colors {activeTab === 'attendance' ? 'bg-zinc-800 text-zinc-100 font-medium' : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'}"
            on:click={() => onTabChange('attendance')}
          >
            Attendance
          </button>
          <button
            class="px-2.5 py-1.5 rounded-md transition-colors {activeTab === 'qa' ? 'bg-zinc-800 text-zinc-100 font-medium' : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'}"
            on:click={() => onTabChange('qa')}
          >
            Q&A
          </button>
        </nav>
      </div>

      <!-- Right Controls: Notifications & Minimal Persona Dropdown -->
      <div class="flex items-center space-x-2">
        
        <!-- Course Selector Button -->
        <div class="relative" data-dropdown>
          <button
            class="h-8 inline-flex items-center space-x-1.5 px-2.5 rounded-md text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900 transition-colors relative after:absolute after:-inset-2 after:content-['']"
            on:click={() => { showCourseDropdown = !showCourseDropdown; showNotifDropdown = false; showUserDropdown = false; }}
          >
            <GraduationCap class="w-4 h-4" />
            <span class="text-xs font-medium font-mono hidden sm:inline-block">
              {activeCourse ? activeCourse.coursecode : 'No Course'}
            </span>
            <ChevronDown class="w-3 h-3 text-zinc-500" />
          </button>

          {#if showCourseDropdown}
            <div class="absolute right-0 mt-2 w-56 rounded-md border border-zinc-800 bg-zinc-950 p-1 shadow-lg z-50">
              <div class="px-2 py-1.5 text-[11px] font-mono text-zinc-500 uppercase tracking-wider">Enrolled Courses</div>
              {#each enrollments as enr}
                <button
                  class="w-full text-left px-2.5 py-1.5 rounded text-xs flex flex-col justify-center hover:bg-zinc-900 transition-colors {selectedCourseId === enr.courseid ? 'text-zinc-100 font-medium bg-zinc-900/50' : 'text-zinc-400'}"
                  on:click={() => {
                    onCourseChange(enr.courseid);
                    showCourseDropdown = false;
                  }}
                >
                  <div class="flex justify-between items-center w-full">
                    <span class="font-mono">{enr.coursecode}</span>
                    <span class="text-[11px] text-zinc-500">{enr.rolename}</span>
                  </div>
                  <div class="text-[11px] truncate text-zinc-500 mt-0.5">{enr.coursetitle}</div>
                </button>
              {/each}
              {#if enrollments.length === 0}
                <div class="px-2.5 py-2 text-xs text-zinc-500">Not enrolled in any courses</div>
              {/if}
            </div>
          {/if}
        </div>

        <!-- Notifications Button -->
        <div class="relative" data-dropdown>
          <button
            aria-label="Notifications"
            class="h-8 w-8 inline-flex items-center justify-center rounded-md text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900 transition-colors relative after:absolute after:-inset-2 after:content-['']"
            on:click={() => { showNotifDropdown = !showNotifDropdown; showUserDropdown = false; showCourseDropdown = false; }}
          >
            <Bell class="w-4 h-4" />
            {#if unreadCount > 0}
              <span class="absolute top-1.5 right-1.5 w-1.5 h-1.5 bg-zinc-100 rounded-full"></span>
            {/if}
          </button>

          <!-- Notifications Menu -->
          {#if showNotifDropdown}
            <div class="absolute right-0 mt-2 w-80 rounded-md border border-zinc-800 bg-zinc-950 p-1 shadow-lg z-50">
              <div class="px-3 py-2 border-b border-zinc-800 text-xs font-medium text-zinc-400 flex justify-between items-center">
                <span>Notifications</span>
                <div class="flex items-center gap-2">
                  {#if unreadCount > 0}
                    <button
                      on:click={onMarkAllRead}
                      class="font-mono text-[11px] text-zinc-500 hover:text-zinc-200 transition-colors"
                    >
                      Mark all read
                    </button>
                  {:else}
                    <span class="font-mono text-[11px] text-zinc-500">{(notifications || []).length} total</span>
                  {/if}
                  <button
                    on:click={onRefresh}
                    aria-label="Refresh notifications"
                    class="text-zinc-500 hover:text-zinc-200 transition-colors p-1.5 -m-1.5"
                  >
                    <RefreshCw class="w-3 h-3" />
                  </button>
                </div>
              </div>
              {#if digest}
                <div class="px-3 py-1.5 border-b border-zinc-800 font-mono text-[11px] text-zinc-400">
                  Catch up: {digest}
                </div>
              {/if}
              <div class="px-2 py-1.5 border-b border-zinc-800 flex items-center gap-1.5">
                <button
                  on:click={() => (notifTab = 'all')}
                  class="px-2 py-0.5 rounded text-[11px] font-mono {notifTab === 'all' ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-500 hover:text-zinc-200'}"
                >
                  All
                </button>
                <button
                  on:click={() => (notifTab = 'unread')}
                  class="px-2 py-0.5 rounded text-[11px] font-mono {notifTab === 'unread' ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-500 hover:text-zinc-200'}"
                >
                  Unread{unreadCount > 0 ? ` (${unreadCount})` : ''}
                </button>
                {#if notifTypes.length > 1}
                  <select
                    bind:value={notifType}
                    aria-label="Filter by type"
                    class="ml-auto rounded border border-zinc-800 bg-zinc-900 px-1.5 py-0.5 text-[11px] font-mono text-zinc-400 max-w-28"
                  >
                    <option value="">All types</option>
                    {#each notifTypes as t}
                      <option value={t}>{t}</option>
                    {/each}
                  </select>
                {/if}
              </div>
              <div class="max-h-64 overflow-y-auto divide-y divide-zinc-800/40">
                {#if visibleNotifs.length === 0}
                  <div class="py-6 text-center text-xs text-zinc-500">No notifications</div>
                {:else}
                  {#each visibleNotifs as notif}
                    <button
                      class="w-full text-left p-2.5 hover:bg-zinc-900/60 rounded transition-colors flex items-start justify-between text-xs {notif.isread?.Bool === false ? 'text-zinc-100' : 'text-zinc-400'}"
                      on:click={() => onOpen(notif)}
                    >
                      <div class="pr-2">
                        <span class="font-mono text-[11px] uppercase text-zinc-500">{notif.type?.String || 'Alert'}</span>
                        <p class="text-xs mt-0.5 line-clamp-2">{notif.message}</p>
                        {#if notif.createdat?.Time}
                          <span class="font-mono text-[11px] text-zinc-500">{timeAgo(notif.createdat)}</span>
                        {/if}
                      </div>
                      {#if notif.isread?.Bool === false}
                        <span class="w-1.5 h-1.5 rounded-full bg-zinc-100 shrink-0 mt-1"></span>
                      {/if}
                    </button>
                  {/each}
                {/if}
              </div>
            </div>
          {/if}
        </div>

        <!-- Account: avatar circle only (identity lives here, not in page heroes) -->
        <div class="relative" data-dropdown>
          <button
            aria-label="Account menu"
            class="h-8 w-8 inline-flex items-center justify-center rounded-full border border-zinc-800 bg-zinc-900/40 hover:bg-zinc-900 text-zinc-200 transition-colors overflow-hidden relative after:absolute after:-inset-2 after:content-['']"
            on:click={() => { showUserDropdown = !showUserDropdown; showNotifDropdown = false; showCourseDropdown = false; }}
          >
            {#if currentUser?.photos && currentUser.photos.length > 0}
              <img src={currentUser.photos[0]} alt="" class="w-full h-full object-cover" />
            {:else}
              <span class="text-xs font-mono">{currentUser?.firstName?.[0] || 'U'}</span>
            {/if}
          </button>

          <!-- Dropdown -->
          {#if showUserDropdown}
            <div class="absolute right-0 mt-2 w-56 rounded-md border border-zinc-800 bg-zinc-950 p-1 shadow-lg z-50">
              <div class="px-2.5 py-1.5 border-b border-zinc-800 mb-1">
                <div class="text-xs font-medium text-zinc-100 truncate">{currentUser?.firstName} {currentUser?.lastName}</div>
                <div class="text-[11px] font-mono text-zinc-500 truncate">{currentUser?.email} · {currentRole || 'No Role'}</div>
              </div>
              <button
                class="w-full text-left px-2.5 py-1.5 rounded text-xs flex items-center space-x-2 text-zinc-400 hover:bg-zinc-900 hover:text-zinc-100 transition-colors"
                on:click={() => { onEditProfile(); showUserDropdown = false; }}
              >
                <User class="w-3.5 h-3.5" />
                <span>Edit profile</span>
              </button>
              <button
                class="w-full text-left px-2.5 py-1.5 rounded text-xs flex items-center space-x-2 text-zinc-400 hover:bg-zinc-900 hover:text-zinc-100 transition-colors"
                on:click={() => { onLogout(); showUserDropdown = false; }}
              >
                <LogOut class="w-3.5 h-3.5" />
                <span>Log out</span>
              </button>
            </div>
          {/if}
        </div>

      </div>

    </div>
  </div>
</header>
