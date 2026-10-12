<script lang="ts">
  import { ExternalLink } from 'lucide-svelte';
  import { api, type User } from '../api';

  export let userId: number | null = null;
  export let onClose: () => void = () => {};

  let profile: User | null = null;
  let loadError = '';

  $: if (userId) {
    loadProfile(userId);
  } else {
    profile = null;
    loadError = '';
  }

  async function loadProfile(id: number) {
    profile = null;
    loadError = '';
    try {
      profile = await api.getUser(id);
    } catch (err: any) {
      loadError = err?.message || 'Unable to load profile.';
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onClose();
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if userId !== null}
  <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4" on:click={onClose}>
    <div
      class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-sm w-full p-5 space-y-4 shadow-xl"
      on:click|stopPropagation
    >
      {#if loadError}
        <p class="text-xs text-red-300">{loadError}</p>
      {:else if !profile}
        <p class="text-xs text-zinc-500 font-mono">Loading profile...</p>
      {:else}
        <div class="flex items-center space-x-3">
          {#if profile.photos && profile.photos.length > 0}
            <img src={profile.photos[0]} alt="" class="h-12 w-12 rounded-full object-cover border border-zinc-800" />
          {:else}
            <div class="h-12 w-12 rounded-full bg-zinc-800 flex items-center justify-center text-sm font-mono text-zinc-300">
              {profile.firstName?.[0] || 'U'}{profile.lastName?.[0] || ''}
            </div>
          {/if}
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-zinc-100 truncate">
              {profile.firstName} {profile.lastName}
            </h3>
            <p class="text-[11px] text-zinc-500 font-mono truncate">{profile.email}</p>
          </div>
        </div>

        {#if profile.bio}
          <p class="text-xs text-zinc-300 leading-relaxed whitespace-pre-wrap">{profile.bio}</p>
        {/if}

        {#if profile.photos && profile.photos.length > 1}
          <div class="flex flex-wrap gap-2">
            {#each profile.photos.slice(1) as url}
              <img src={url} alt="" class="h-12 w-12 rounded-md object-cover border border-zinc-800" />
            {/each}
          </div>
        {/if}

        <div class="flex flex-col gap-1.5 pt-1">
          {#if profile.githubLink}
            <a
              href={profile.githubLink}
              target="_blank"
              rel="noreferrer"
              class="inline-flex items-center gap-1.5 text-xs text-zinc-300 hover:text-white underline underline-offset-2 break-all"
            >
              <ExternalLink class="w-3 h-3 shrink-0" />
              <span class="break-all font-mono text-[11px]">{profile.githubLink}</span>
            </a>
          {/if}
          {#if profile.linkedinLink}
            <a
              href={profile.linkedinLink}
              target="_blank"
              rel="noreferrer"
              class="inline-flex items-center gap-1.5 text-xs text-zinc-300 hover:text-white underline underline-offset-2 break-all"
            >
              <ExternalLink class="w-3 h-3 shrink-0" />
              <span class="break-all font-mono text-[11px]">{profile.linkedinLink}</span>
            </a>
          {/if}
          {#if !profile.githubLink && !profile.linkedinLink}
            <p class="text-[11px] text-zinc-600">No professional links shared.</p>
          {/if}
        </div>
      {/if}

      <div class="flex justify-end pt-2 border-t border-zinc-800">
        <button
          on:click={onClose}
          class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100"
        >
          Close
        </button>
      </div>
    </div>
  </div>
{/if}
