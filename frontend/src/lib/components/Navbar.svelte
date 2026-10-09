<script lang="ts">
  import { onMount } from 'svelte';
  import { Bell, ChevronDown, GraduationCap, LogOut, User } from 'lucide-svelte';
  import type { User as UserType, NotificationItem } from '../api';

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

  let showUserDropdown = false;
  let showNotifDropdown = false;
  let showCourseDropdown = false;

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

  $: unreadCount = notifications.filter(n => n.isread?.Bool === false).length;
  $: activeCourse = enrollments.find(e => e.courseid === selectedCourseId);
</script>

<header class="border-b border-zinc-800 bg-zinc-950/95 sticky top-0 z-50 backdrop-blur-xs">
  <div class="max-w-6xl mx-auto px-4 sm:px-6">
    <div class="flex items-center justify-between h-14">
      
      <!-- Brand & Tabs -->
      <div class="flex items-center space-x-6">
        <button 
          class="flex items-center space-x-2.5 text-left group focus:outline-none" 
          on:click={() => onTabChange('dashboard')}
        >
          <div class="w-6 h-6 rounded-md bg-zinc-100 text-zinc-950 flex items-center justify-center font-bold text-xs">
            M
          </div>
          <span class="text-sm font-semibold tracking-tight text-zinc-100">moodle++</span>
          <span class="text-[10px] font-mono text-zinc-500 uppercase tracking-widest px-1.5 py-0.5 rounded border border-zinc-800 bg-zinc-900/60">4NF</span>
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
            class="h-8 inline-flex items-center space-x-1.5 px-2.5 rounded-md text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900 transition-colors focus:outline-none"
            on:click={() => { showCourseDropdown = !showCourseDropdown; showNotifDropdown = false; showUserDropdown = false; }}
          >
            <GraduationCap class="w-4 h-4" />
            <span class="text-xs font-medium font-mono hidden sm:inline-block">
              {activeCourse ? activeCourse.coursecode : 'No Course'}
            </span>
            <ChevronDown class="w-3 h-3 text-zinc-600" />
          </button>

          {#if showCourseDropdown}
            <div class="absolute right-0 mt-2 w-56 rounded-md border border-zinc-800 bg-zinc-950 p-1 shadow-lg z-50">
              <div class="px-2 py-1.5 text-[10px] font-mono text-zinc-500 uppercase tracking-wider">Enrolled Courses</div>
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
                    <span class="text-[10px] text-zinc-500">{enr.rolename}</span>
                  </div>
                  <div class="text-[10px] truncate text-zinc-600 mt-0.5">{enr.coursetitle}</div>
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
            class="h-8 w-8 inline-flex items-center justify-center rounded-md text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900 transition-colors relative"
            on:click={() => { showNotifDropdown = !showNotifDropdown; showUserDropdown = false; showCourseDropdown = false; }}
          >
            <Bell class="w-4 h-4" />
            {#if unreadCount > 0}
              <span class="absolute top-1.5 right-1.5 w-1.5 h-1.5 bg-zinc-100 rounded-full"></span>
            {/if}
          </button>

          <!-- Notifications Menu -->
          {#if showNotifDropdown}
            <div class="absolute right-0 mt-2 w-72 rounded-md border border-zinc-800 bg-zinc-950 p-1 shadow-lg z-50">
              <div class="px-3 py-2 border-b border-zinc-800 text-xs font-medium text-zinc-400 flex justify-between items-center">
                <span>Notifications</span>
                {#if unreadCount > 0}
                  <button
                    on:click={onMarkAllRead}
                    class="font-mono text-[10px] text-zinc-500 hover:text-zinc-200 transition-colors"
                  >
                    Mark all read
                  </button>
                {:else}
                  <span class="font-mono text-[10px] text-zinc-500">{notifications.length} total</span>
                {/if}
              </div>
              <div class="max-h-64 overflow-y-auto divide-y divide-zinc-800/40">
                {#if notifications.length === 0}
                  <div class="py-6 text-center text-xs text-zinc-500">No notifications</div>
                {:else}
                  {#each notifications as notif}
                    <button
                      class="w-full text-left p-2.5 hover:bg-zinc-900/60 rounded transition-colors flex items-start justify-between text-xs {notif.isread?.Bool === false ? 'text-zinc-100' : 'text-zinc-400'}"
                      on:click={() => onMarkRead(notif.notificationid)}
                    >
                      <div class="pr-2">
                        <span class="font-mono text-[10px] uppercase text-zinc-500">{notif.type?.String || 'Alert'}</span>
                        <p class="text-xs mt-0.5 line-clamp-2">{notif.message}</p>
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

        <!-- User Menu -->
        <div class="relative" data-dropdown>
          <button
            class="h-8 px-2.5 inline-flex items-center space-x-2 rounded-md border border-zinc-800 bg-zinc-900/40 hover:bg-zinc-900 text-zinc-200 text-xs transition-colors"
            on:click={() => { showUserDropdown = !showUserDropdown; showNotifDropdown = false; showCourseDropdown = false; }}
          >
            {#if currentUser?.photos && currentUser.photos.length > 0}
              <img src={currentUser.photos[0]} alt="avatar" class="w-4 h-4 rounded-full object-cover" />
            {:else}
              <div class="w-4 h-4 rounded-full bg-zinc-800 text-[10px] flex items-center justify-center font-mono">
                {currentUser?.firstName?.[0] || 'U'}
              </div>
            {/if}
            <span class="font-medium text-xs text-zinc-200 hidden sm:inline">{currentUser?.firstName}</span>
            <span class="text-zinc-500 text-[10px] hidden sm:inline font-mono">
              {currentRole || 'No Role'}
            </span>
            <ChevronDown class="w-3.5 h-3.5 text-zinc-500" />
          </button>

          <!-- Dropdown -->
          {#if showUserDropdown}
            <div class="absolute right-0 mt-2 w-56 rounded-md border border-zinc-800 bg-zinc-950 p-1 shadow-lg z-50">
              <div class="px-2.5 py-1.5 text-[10px] font-mono text-zinc-500 border-b border-zinc-800 mb-1 truncate">
                {currentUser?.email}
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
