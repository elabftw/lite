<!--
This file is part of eLabFTW lite.

@author Nicolas CARPi / Deltablot
@author Moustapha Camara / Deltablot
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang='ts'>
  import { onMount } from 'svelte';
  import {
    ListEntries,
    GetEntry,
    GetPreferredEditor,
    SaveEntry,
    UpdateEntry,
    DeleteEntry,
    LockProfile,
    PullEntryFromElabftw,
    ImportEntryFromElabftw,
    PushEntryToElabftw,
    PushAllEntriesToElabftw,
    ListEntryRemoteLinks,
    UnlinkEntryFromElabftw,
  } from '../../bindings/elabftw-lite/app';
  import type * as main from '../../bindings/elabftw-lite/models';
  import { autofocus, errorMessage, openExternalURL, preventDefaultSubmit, toRelativeTime } from '../utils/helpers';
  import ProfileSettings from './ProfileSelector/ProfileSettings.svelte';
  import InstancesPushModal from './Instances/InstancesPushModal.svelte';
  import EntryRemoteModal from './Instances/EntryRemoteModal.svelte';
  import EntryImportModal from './Instances/EntryImportModal.svelte';
  import MarkdownEditor from "./MarkdownEditor.svelte";
  import TinyMceEditor from "./TinyMceEditor.svelte";
  import { showAlert } from "./stores/alert.svelte";
  import UploadsPanel from './Uploads/UploadsPanel.svelte';
  import EntryReadonlyPanels from './EntryReadonlyPanels.svelte';
  import EntityTypeIcon from './EntityTypeIcon.svelte';
  import EntityCategoryStatus from './EntityCategoryStatus.svelte';

  type Props = {
    profileUuid: string;
    profileName: string;
    onLogout?: () => void;
  };

  type View = 'index' | 'editor' | 'profile';
  type PreferredEditor = 'tinymce' | 'markdown';

  let {profileUuid, profileName, onLogout}: Props = $props();

  let preferredEditor = $state<PreferredEditor>('tinymce');
  let preferencesReady = $state(false);
  let entryTitle = $state('');
  let entryMainText = $state('');
  let selectedEntry = $state<main.Entry | null>(null);
  let entryContentType = $state<1 | 2>(1);
  let entries = $state<main.EntrySummary[]>([]);
  type EntryFilter = 'all' | 'experiment' | 'resource';
  type EntrySort = 'recent' | 'oldest' | 'title-asc' | 'title-desc' | 'created';
  let entrySearch = $state('');
  let entryFilter = $state<EntryFilter>('all');
  let entrySort = $state<EntrySort>('recent');
  const visibleEntries = $derived.by(() => {
    const query = entrySearch.trim().toLocaleLowerCase();
    return entries
      .filter((entry) => (entryFilter === 'all' || entry.remoteType === entryFilter) &&
        (!query || (entry.title || '').toLocaleLowerCase().includes(query) || String(entry.id).includes(query)))
      .sort((a, b) => {
        if (entrySort === 'title-asc' || entrySort === 'title-desc') {
          const order = a.title.localeCompare(b.title, undefined, {sensitivity: 'base'});
          return entrySort === 'title-asc' ? order : -order;
        }
        const key = entrySort === 'created' ? 'createdAt' : 'modifiedAt';
        const difference = (Date.parse(b[key]) || 0) - (Date.parse(a[key]) || 0);
        return entrySort === 'oldest' ? -difference : difference;
      });
  });
  let view = $state<View>('index');
  let loading = $state(false);
  let currentEntryId = $state<number | null>(null); // null means a new entry, otherwise update the selected one
  // Prevent editing an existing body with the wrong editor format
  let editorFormatMismatch = $derived(entryContentType !== (preferredEditor === 'markdown' ? 2 : 1));
  let importModalOpen = $state(false);
  let pushModalOpen = $state(false);
  let pushMode = $state<'single' | 'all'>('single'); // from View of an entry, push a single entry. From list of entries, push all.
  let pushEntryId = $state<number | null>(null); // # currentEntryId. This is for the modal to push to eLab.
  let lastFailedPush = $state<{
    instanceId: number;
    entityType: 'experiment' | 'resource';
  } | null>(null);
  let remoteLinks = $state<main.EntryRemoteLink[]>([]);
  // The oldest link determines whether this local entry is an experiment or resource
  let linkedEntityType = $derived(
    remoteLinks[0]?.type === 'experiment' ? 'experiment'
      : remoteLinks[0]?.type === 'resource' ? 'resource' : null
  );
  let remoteModalMode = $state<'open' | 'pull' | 'unlink' | null>(null);
  let remoteActionInProgress = $state(false);
  // Recreate the uploads panel after a pull to load the replacement files
  let uploadsRefreshKey = $state(0);
  let entryInfoExpanded = $state(true);
  let mobileProfileOpen = $state(false);
  let desktopProfileOpen = $state(false);
  let mobileGlobalActionsOpen = $state(false);

  // Load the preferred editor from this profile's SQLite settings
  async function loadPreferences(): Promise<void> {
    try {
      const editor = await GetPreferredEditor(profileUuid);
      preferredEditor = editor === 'markdown' ? 'markdown' : 'tinymce';
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    } finally {
      preferencesReady = true;
    }
  }

  async function loadEntry(id: number, requireEditor = false): Promise<void> {
    const entry = await GetEntry(profileUuid, id);
    const links = await ListEntryRemoteLinks(profileUuid, id);
    // Ignore stale responses if the user has opened another entry
    if (currentEntryId !== id || (requireEditor && view !== 'editor')) return;
    entryTitle = entry.title;
    entryMainText = entry.body;
    selectedEntry = entry;
    entryContentType = entry.contentType === 2 ? 2 : 1;
    remoteLinks = links;
  }

  async function openEntry(id: number): Promise<void> {
    showAlert(null);
    currentEntryId = id;
    try {
      await loadEntry(id);
      view = 'editor';
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    }
  }

  async function refreshEntries(): Promise<void> {
    loading = true;
    try {
      entries = await ListEntries(profileUuid);
    } catch (e: unknown) {
      console.error(e);
      showAlert({type: 'error', message: errorMessage(e)});
    } finally {
      loading = false;
    }
  }

  async function openIndex(): Promise<void> {
    await refreshEntries();
    showAlert(null);
    view = 'index';
  }

  async function logout(): Promise<void> {
    try {
      await LockProfile();
      onLogout?.();
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    }
  }

  // Start a blank entry using the profile's preferred editor
  function openEditor(): void {
    view = 'editor';
    entryTitle = '';
    entryMainText = '';
    selectedEntry = null;
    entryContentType = preferredEditor === 'markdown' ? 2 : 1;
    showAlert(null);
    currentEntryId = null;
    remoteLinks = [];
  }

  // Save a new entry or update the selected one
  async function saveOrUpdateEntry(): Promise<void> {
    if (editorFormatMismatch) {
      showAlert({type: 'warning', message: 'Change your preferred editor in Profile settings to edit this entry.'});
      return;
    }
    showAlert({type: 'info', message: currentEntryId ? 'Updating...' : 'Saving...'});
    try {
      if (currentEntryId) {
        await UpdateEntry(profileUuid, currentEntryId, entryTitle, entryMainText, entryContentType);
        remoteLinks = await ListEntryRemoteLinks(profileUuid, currentEntryId);
        showAlert({type: 'success', message: 'Entry updated ✔'});
      } else {
        const id = await SaveEntry(profileUuid, entryTitle, entryMainText, entryContentType);
        currentEntryId = id;
        showAlert({type: 'success', message: `Saved with id ${id} ✔`});
      }

      await refreshEntries();
      if (currentEntryId) selectedEntry = await GetEntry(profileUuid, currentEntryId);
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    }
  }

  let profileSettingsReturnView = $state<'index' | 'editor'>('index');

  function openProfileSettings(): void {
    showAlert(null);
    if (view !== 'profile') profileSettingsReturnView = view;
    view = 'profile';
  }

  function closeProfileSettings(): void {
    showAlert(null);
    view = profileSettingsReturnView;
  }

  // Uploads need a saved entry ID before they can be attached
  async function ensureEntrySaved(): Promise<number | null> {
    if (!currentEntryId) await saveOrUpdateEntry();
    return currentEntryId;
  }

  async function deleteEntry(id: number, title: string): Promise<void> {
    showAlert(null);
    const confirmed = window.confirm(`Delete "${title}"? This cannot be undone.`);
    if (!confirmed) {
      return;
    }
    try {
      await DeleteEntry(profileUuid, id);
      await refreshEntries();
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    }
  }

  const handleSubmit = preventDefaultSubmit(saveOrUpdateEntry);

  onMount(() => {
    if (window.matchMedia('(max-width: 720px)').matches) entryInfoExpanded = false;
    showAlert(null);
    void refreshEntries();
    void loadPreferences();
  });

  async function confirmImport(instanceId: number, entityType: 'experiment' | 'resource', remoteId: number): Promise<void> {
    const result = await ImportEntryFromElabftw(profileUuid, instanceId, entityType, remoteId);
    importModalOpen = false;
    await refreshEntries();
    const warning = result.warnings?.length ? ` Warning: ${result.warnings.join(' ')}` : '';
    showAlert({
      type: result.warnings?.length ? 'warning' : 'success',
      message: `Imported ${result.type} #${result.remoteId} as local entry #${result.localId} ✔${warning}`,
    });
  }

  // modal helpers
  function openPushModal(mode: 'single' | 'all', entryId: number | null = null): void {
    pushMode = mode;
    pushEntryId = entryId;
    showAlert(null);
    pushModalOpen = true;
  }

  function closePushModal(): void {
    pushModalOpen = false;
    lastFailedPush = null;
  }

  function openRemoteModal(mode: 'open' | 'pull' | 'unlink'): void {
    if (remoteLinks.length === 0) return;
    showAlert(null);
    if (mode === 'open' && remoteLinks.length === 1) {
      openExternalURL(remoteLinks[0].url);
      return;
    }
    remoteModalMode = mode;
  }

  function closeRemoteModal(): void {
    if (remoteActionInProgress) return;
    remoteModalMode = null;
  }

  async function confirmPull(link: main.EntryRemoteLink): Promise<void> {
    const entryId = currentEntryId;
    if (entryId === null) {
      showAlert({type: 'error', message: 'No entry selected.'});
      return;
    }
    if (remoteActionInProgress) return;
    remoteActionInProgress = true;

    try {
      const result = await PullEntryFromElabftw(
        profileUuid,
        entryId,
        link.instanceId,
        link.type,
      );

      // Ignore a pull result if another entry is now selected
      if (currentEntryId !== entryId || view !== 'editor') {
        remoteModalMode = null;
        return;
      }

      await loadEntry(entryId, true);
      if (currentEntryId !== entryId || view !== 'editor') {
        remoteModalMode = null;
        return;
      }
      uploadsRefreshKey += 1;
      await refreshEntries();

      const warning = result.warnings?.length ? ` Warning: ${result.warnings.join(' ')}` : '';
      showAlert({
        type: result.warnings?.length ? 'warning' : 'success',
        message: `Pulled ${result.type} #${result.remoteId} and replaced the local entry ✔${warning}`,
      });
      remoteModalMode = null;
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    } finally {
      remoteActionInProgress = false;
    }
  }

  async function confirmUnlink(link: main.EntryRemoteLink): Promise<void> {
    const entryId = currentEntryId;
    if (entryId === null || remoteActionInProgress) return;
    remoteActionInProgress = true;

    try {
      await UnlinkEntryFromElabftw(profileUuid, entryId, link.instanceId, link.remoteId, link.type);
      remoteModalMode = null;
      if (currentEntryId !== entryId || view !== 'editor') return;

      remoteLinks = await ListEntryRemoteLinks(profileUuid, entryId);
      selectedEntry = await GetEntry(profileUuid, entryId);
      await refreshEntries();
      showAlert({type: 'success', message: 'Entry unlinked from eLabFTW ✔'});
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    } finally {
      remoteActionInProgress = false;
    }
  }

  async function confirmPush(
    instanceId: number,
    entityType: 'experiment' | 'resource',
    force = false,
  ): Promise<void> {
    try {
      if (pushMode === 'all') {
        const results = await PushAllEntriesToElabftw(profileUuid, instanceId, entityType, force);
        showAlert({type: 'success', message: `Pushed ${results.length} entries ✔`});
      } else {
        if (!pushEntryId) {
          showAlert({type: 'error', message: 'No entry selected.'});
          return;
        }

        const result = await PushEntryToElabftw(profileUuid, pushEntryId, instanceId, entityType, force);
        remoteLinks = await ListEntryRemoteLinks(profileUuid, currentEntryId);
        const warning = result.uploadWarnings?.length ? ` Upload warning: ${result.uploadWarnings.join(' ')}` : '';

        showAlert({
          type: result.uploadWarnings?.length ? 'warning' : 'success',
          message: `Entry ${result.action} as ${result.type} #${result.remoteId} ✔${warning}`,
        });
      }

      lastFailedPush = null;
      pushModalOpen = false;
    } catch (e: unknown) {
      const message = errorMessage(e);
      // warning if remote data is more recent than lite
      if (message.includes('was modified after your last sync')) {
        lastFailedPush = { instanceId, entityType };
        showAlert({ type: 'warning', message });
      } else {
        lastFailedPush = null;
        showAlert({ type: 'error', message });
      }
    }
  }
</script>

<!-- when using editor, the container becomes larger (side panel with uploads/metadata appears) -->
<div class='container' class:editor-container={view === 'editor'}>
  <header class='app-header flex justify-between items-center mb-2' class:border-bottom={view !== 'editor'}>
    <div class='w-100 text-ellipsis'>
      {#if view === 'index'}
        <h1>My Entries</h1>
      {:else if view === 'editor'}
        <div class='flex items-center gap-1'>
          {#if linkedEntityType}
            <h1 class='text-info'><span aria-hidden='true'><EntityTypeIcon type={linkedEntityType} /></span></h1>
          {/if}
          <div class='grid'>
            <h1 class='text-ellipsis'>{entryTitle.trim() || 'Untitled entry'}</h1>
            <EntityCategoryStatus remoteDetails={selectedEntry?.remoteDetails} />
          </div>
        </div>
      {:else} <!-- view === 'profile' -->
        <h1>Profile settings</h1>
        <h2>Manage preferences for {profileName}.</h2>
      {/if}
    </div>

    <div class='profile-actions flex gap-1 items-center'>
      <details class='desktop-profile-menu' bind:open={desktopProfileOpen}>
        <summary class='profile-pill flex items-center' aria-label={`Profile menu for ${profileName}`}>
          <span class='profile-avatar'>{profileName.slice(0, 2).toUpperCase()}</span>
          <span class='text-strong'>{profileName}</span>
          <svg class='desktop-profile-chevron' aria-hidden='true' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='m6 9 6 6 6-6'/></svg>
        </summary>
        <div class='desktop-profile-popover'>
          <button type='button' disabled={!preferencesReady || view === 'profile'}
            onclick={() => { desktopProfileOpen = false; openProfileSettings(); }}>
            <svg aria-hidden='true' width='19' height='19' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><circle cx='12' cy='12' r='3'/><path d='M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-1.9 1.9-.06-.06A1.7 1.7 0 0 0 16 18.4l-.3.12V21h-2.7v-2.48L12 18.4l-1 .12V21H8.3v-2.48l-.3-.12a1.7 1.7 0 0 0-1.88.34l-.06.06-1.9-1.9.06-.06A1.7 1.7 0 0 0 4.6 15L4.48 14H2v-2.7h2.48l.12-.3a1.7 1.7 0 0 0-.34-1.88L4.2 9.06l1.9-1.9.06.06A1.7 1.7 0 0 0 8 7.6l.3-.12V5H11v2.48l1 .12 1-.12V5h2.7v2.48l.3.12a1.7 1.7 0 0 0 1.88-.34l.06-.06 1.9 1.9-.06.06A1.7 1.7 0 0 0 19.4 11l.12.3H22V14h-2.48z'/></svg>
            Settings
          </button>
          <button type='button' onclick={() => { desktopProfileOpen = false; void logout(); }}>
            <svg aria-hidden='true' width='19' height='19' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='M10 17l5-5-5-5m5 5H3m9-9h8v18h-8'/></svg>
            Logout
          </button>
        </div>
      </details>
    </div>
  </header>

  <!-- VIEW MODE -->
  <!-- Navigation appears only on phone-sized screens; the desktop header is unchanged. -->
  <nav class='mobile-app-nav' aria-label='Main navigation'>
    <button type='button' disabled={view === 'index'} aria-label='Back'
      onclick={() => { mobileProfileOpen = false; mobileGlobalActionsOpen = false; if (view === 'profile') closeProfileSettings(); else void openIndex(); }}>
      <svg aria-hidden='true' width='21' height='21' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><path d='m15 18-6-6 6-6'/></svg>
      <span>Back</span>
    </button>
    <button type='button' class:active={view === 'index'} aria-current={view === 'index' ? 'page' : undefined}
      onclick={() => { mobileProfileOpen = false; mobileGlobalActionsOpen = false; void openIndex(); }}>
      <svg aria-hidden='true' width='21' height='21' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><path d='m3 10 9-7 9 7v10a1 1 0 0 1-1 1h-5v-7H9v7H4a1 1 0 0 1-1-1z'/></svg>
      <span>Home</span>
    </button>
    <details class='mobile-global-actions-menu' bind:open={mobileGlobalActionsOpen}>
      <summary aria-label='Actions menu' onclick={() => mobileProfileOpen = false}>
        <svg aria-hidden='true' width='21' height='21' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' stroke-linecap='round'><circle cx='5' cy='12' r='1'/><circle cx='12' cy='12' r='1'/><circle cx='19' cy='12' r='1'/></svg>
        <span>Actions</span>
      </summary>
      <div class='mobile-global-actions-popover'>
        {#if view === 'index'}
          <button type='button' class='mobile-primary-action' disabled={!preferencesReady} onclick={() => { mobileGlobalActionsOpen = false; openEditor(); }}>
            <svg aria-hidden='true' width='19' height='19' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='M12 5v14M5 12h14'/></svg>
            Create entry
          </button>
          <button type='button' onclick={() => { mobileGlobalActionsOpen = false; showAlert(null); importModalOpen = true; }}>
            <svg aria-hidden='true' width='19' height='19' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='M12 3v12m-5-5 5 5 5-5M5 20h14'/></svg>
            Import from eLabFTW
          </button>
          <button type='button' disabled={entries.length === 0} onclick={() => { mobileGlobalActionsOpen = false; openPushModal('all'); }}>
            <svg aria-hidden='true' width='19' height='19' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='M12 21V6m-5 5 5-5 5 5M5 3h14'/></svg>
            Push all to eLabFTW
          </button>
        {:else if view === 'editor'}
        <button class='mobile-save-action' type='submit' form='entry-editor-form' disabled={editorFormatMismatch} onclick={() => mobileGlobalActionsOpen = false}>
          <svg aria-hidden='true' width='19' height='19' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z'/><path d='M17 21v-8H7v8M7 3v5h8'/></svg>
          Save entry
        </button>
        {#if remoteLinks.length > 0}
          <button type='button' onclick={() => { mobileGlobalActionsOpen = false; openRemoteModal('open'); }}>
            <svg aria-hidden='true' width='19' height='19' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='M14 3h7v7M10 14 21 3M21 13v7H3V3h7'/></svg>
            Open in eLabFTW
          </button>
          <button type='button' onclick={() => { mobileGlobalActionsOpen = false; openRemoteModal('pull'); }}>
            <svg aria-hidden='true' width='19' height='19' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='M12 3v15m-5-5 5 5 5-5M5 21h14'/></svg>
            Pull from eLabFTW
          </button>
          <button type='button' onclick={() => { mobileGlobalActionsOpen = false; openRemoteModal('unlink'); }}>
            <svg aria-hidden='true' width='19' height='19' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='m17 7 4-4M3 21l4-4M9 15l6-6M6 12l-1 1a4 4 0 0 0 6 6l1-1m0-12 1-1a4 4 0 0 1 6 6l-1 1'/></svg>
            Unlink entry
          </button>
        {/if}
        <button type='button' disabled={!currentEntryId} onclick={() => { mobileGlobalActionsOpen = false; openPushModal('single', currentEntryId); }}>
          <svg aria-hidden='true' width='19' height='19' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='M12 21V6m-5 5 5-5 5 5M5 3h14'/></svg>
          Push to eLabFTW
        </button>
        {:else}
          <p class='mobile-menu-hint'>Nothing to do here yet.</p>
        {/if}
      </div>
    </details>
    <details class='mobile-profile-menu' bind:open={mobileProfileOpen}>
      <summary aria-label='Profile menu' onclick={() => mobileGlobalActionsOpen = false}>
        <svg aria-hidden='true' width='21' height='21' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><circle cx='12' cy='8' r='4'/><path d='M4 21v-2a8 8 0 0 1 16 0v2'/></svg>
        <span>Profile</span>
      </summary>
      <div class='mobile-profile-popover'>
        <p class='mobile-profile-name'><span class='mobile-profile-initials'>{profileName.slice(0, 2).toUpperCase()}</span>{profileName}</p>
        <button type='button' disabled={!preferencesReady || view === 'profile'}
          onclick={() => { mobileProfileOpen = false; openProfileSettings(); }}>
          <svg aria-hidden='true' width='19' height='19' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><circle cx='12' cy='12' r='3'/><path d='M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-1.9 1.9-.06-.06A1.7 1.7 0 0 0 16 18.4l-.3.12V21h-2.7v-2.48L12 18.4l-1 .12V21H8.3v-2.48l-.3-.12a1.7 1.7 0 0 0-1.88.34l-.06.06-1.9-1.9.06-.06A1.7 1.7 0 0 0 4.6 15L4.48 14H2v-2.7h2.48l.12-.3a1.7 1.7 0 0 0-.34-1.88L4.2 9.06l1.9-1.9.06.06A1.7 1.7 0 0 0 8 7.6l.3-.12V5H11v2.48l1 .12 1-.12V5h2.7v2.48l.3.12a1.7 1.7 0 0 0 1.88-.34l.06-.06 1.9 1.9-.06.06A1.7 1.7 0 0 0 19.4 11l.12.3H22V14h-2.48z'/></svg>
          Settings
        </button>
        <button type='button' onclick={() => { mobileProfileOpen = false; void logout(); }}>
          <svg aria-hidden='true' width='19' height='19' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><path d='M10 17l5-5-5-5m5 5H3m9-9h8v18h-8'/></svg>
          Logout
        </button>
      </div>
    </details>
  </nav>

  {#if view === 'index'}
    <section class='panel entries-page' aria-label='Entries'>
      <div class='mobile-entry-controls'>
        <label class='mobile-entry-search'>
          <svg aria-hidden='true' width='20' height='20' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><circle cx='11' cy='11' r='8'/><path d='m21 21-4.35-4.35'/></svg>
          <input type='search' placeholder='Search entries...' aria-label='Search entries' bind:value={entrySearch} />
        </label>
        <div class='mobile-entry-filters' role='group' aria-label='Filter entries'>
          <button type='button' class='entry-filter-option' class:active={entryFilter === 'all'} aria-pressed={entryFilter === 'all'} onclick={() => entryFilter = 'all'}>All</button>
          <button type='button' class='entry-filter-option' class:active={entryFilter === 'experiment'} aria-pressed={entryFilter === 'experiment'} onclick={() => entryFilter = 'experiment'}>Experiments</button>
          <button type='button' class='entry-filter-option' class:active={entryFilter === 'resource'} aria-pressed={entryFilter === 'resource'} onclick={() => entryFilter = 'resource'}>Resources</button>
        </div>
        <div class='mobile-entry-sort'>
          <span class='description'>{visibleEntries.length} {visibleEntries.length === 1 ? 'entry' : 'entries'}</span>
          <label for='entry-sort'>Sort by</label>
          <select id='entry-sort' class='input' bind:value={entrySort}>
            <option value='recent'>Last edited</option>
            <option value='oldest'>Oldest edited</option>
            <option value='created'>Recently created</option>
            <option value='title-asc'>Title A-Z</option>
            <option value='title-desc'>Title Z-A</option>
          </select>
        </div>
      </div>

      <div class='entries-toolbar flex justify-end mb-2'>
        <div class='entries-actions desktop-entry-actions flex gap-1'>
          <button class='btn btn-secondary entry-action-button' type='button' aria-label='Import from eLabFTW' title='Import from eLabFTW' onclick={() => { showAlert(null); importModalOpen = true; }}>
            <svg class='mobile-import-icon' aria-hidden='true' width='1.25em' height='1.25em' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><path d='M12 16V4m-5 5 5-5 5 5M5 20h14'/></svg>
            <span class='entry-action-label'>Import from eLabFTW</span>
            <span class='mobile-import-label'>Import</span>
          </button>
          <button class='btn btn-primary entry-action-button' onclick={openEditor} disabled={!preferencesReady} aria-label='Create entry' title='Create entry'><span aria-hidden='true'>+</span><span class='entry-action-label'>Create entry</span></button>
        </div>
      </div>

      {#if loading}
        <div class='empty-state'>
          Loading entries...
        </div>
      {:else if entries.length === 0}
        <div class='empty-state'>
          <div class='icon' aria-hidden='true'>✎</div>
          <h3>No entries yet</h3>
          <p>Create your first entry to start writing.</p>
        </div>
      {:else if visibleEntries.length === 0}
        <div class='empty-state'>
          <h3>No matching entries</h3>
          <p>Try another search or filter.</p>
          <button type='button' class='btn btn-secondary' onclick={() => { entrySearch = ''; entryFilter = 'all'; }}>Clear filters</button>
        </div>
      {:else}
        <div class='grid gap-1'>
          {#each visibleEntries as e (e.id)}
            <div class='entry-row flex gap-1'>
              <button type='button' class='entry-card' onclick={() => openEntry(e.id)} disabled={!preferencesReady}>
                <span class='icon-sm' aria-hidden='true'><EntityTypeIcon type={e.remoteType} /></span>
                <span class='entry-card-details grid gap-03'>
                <span class='text-ellipsis text-strong text-big'>
                  {e.title || 'Untitled entry'}
                </span>
                <span class='description'>
                  Last edited {toRelativeTime(e.modifiedAt)}
                </span>
                <EntityCategoryStatus remoteDetails={e.remoteDetails} />
              </span>

                <span class='entry-card-open text-orange text-strong' aria-hidden='true'>
                Open &#8594;
              </span>
              </button>
              <button type='button' class='btn btn-danger entry-delete-button' onclick={() => deleteEntry(e.id, e.title)}
                      aria-label={`Delete ${e.title}`}>
                <span aria-hidden='true'>&#128465;</span>
              </button>
            </div>
          {/each}
        </div>
      {/if}
      <div class='desktop-push-all flex justify-end mt-2 gap-1'>
        {#if entries.length !== 0}
          <button class='btn btn-secondary' onclick={() => openPushModal('all')}>Push all entries to eLabFTW</button>
        {/if}
      </div>
      <div class='mobile-entry-dock'>
        <button class='btn btn-secondary' type='button' onclick={() => { showAlert(null); importModalOpen = true; }}>
          <svg aria-hidden='true' width='20' height='20' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='M12 3v12m-5-5 5 5 5-5M5 20h14'/></svg>
          Import
        </button>
        <button class='btn btn-primary' type='button' onclick={openEditor} disabled={!preferencesReady}>
          <span aria-hidden='true'>+</span> Create entry
        </button>
      </div>
    </section>
    <!-- PROFILE SETTINGS -->
  {:else if view === 'profile'}
    <ProfileSettings
      {profileUuid}
      {preferredEditor}
      onBack={closeProfileSettings}
      onSaved={(editor) => {
        if (currentEntryId === null && entryMainText.trim() === '') {
          entryContentType = editor === 'markdown' ? 2 : 1;
        }
        preferredEditor = editor;
      }}
    />
    <!-- VIEW EDITOR MODE -->
  {:else if view === 'editor'}
    <div>
      <div class='panel entry-editor-toolbar flex justify-between mb-2'>
          <button class='btn btn-secondary' type='button' onclick={openIndex} aria-label='Back' title='Back'><span aria-hidden='true'>←</span><span class='editor-action-label'>Back</span></button>
          <div class='entry-editor-actions flex gap-1'>
              {#if remoteLinks.length > 0}
                <div class='entry-editor-remote-actions flex gap-1'>
                  <button
                    type='button'
                    class='btn btn-secondary btn-external-link'
                    aria-label='Open in eLabFTW'
                    title={remoteLinks.length === 1
                      ? `Open ${remoteLinks[0].type} #${remoteLinks[0].remoteId} on ${remoteLinks[0].siteUrl}`
                      : 'Choose an eLabFTW instance'}
                    onclick={() => openRemoteModal('open')}
                  >
                    <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><path d='M14 3h7v7M10 14 21 3M21 13v7a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h7'/></svg>
                    <span class='editor-action-label'>Open in eLabFTW</span>
                  </button>
                  <button type='button' class='btn btn-secondary' onclick={() => openRemoteModal('pull')} aria-label='Pull' title='Pull'>
                    <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 384 512'><path fill='currentColor' d='M169.4 502.6c12.5 12.5 32.8 12.5 45.3 0l160-160c12.5-12.5 12.5-32.8 0-45.3s-32.8-12.5-45.3 0L224 402.7V32c0-17.7-14.3-32-32-32s-32 14.3-32 32v370.7L54.6 297.3c-12.5-12.5-32.8-12.5-45.3 0s-12.5 32.8 0 45.3l160 160z'/></svg>
                    <span class='editor-action-label'>Pull</span>
                  </button>
                  <button type='button' class='btn btn-secondary btn-quiet'
                          aria-label='Unlink'
                          title='Remove the local sync link without deleting either entry'
                          onclick={() => openRemoteModal('unlink')}>
                    <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><path d='m18.84 12.25 1.72-1.71a4 4 0 0 0-5.66-5.66l-1.72 1.71M5.16 11.75l-1.72 1.71a4 4 0 0 0 5.66 5.66l1.72-1.71M8 2v4M2 8h4m10 14v-4m6-2h-4'/></svg>
                    <span class='editor-action-label'>Unlink</span>
                  </button>
                </div>
              {/if}
            <button class='btn btn-secondary' type='button' disabled={!currentEntryId}
                    aria-label='Push' title='Push' onclick={() => openPushModal('single', currentEntryId)}>
              <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 384 512'><path fill='currentColor' d='M214.6 9.4c-12.5-12.5-32.8-12.5-45.3 0l-160 160c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L160 109.3V480c0 17.7 14.3 32 32 32s32-14.3 32-32V109.3l105.4 105.4c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3l-160-160z'/></svg>
              <span class='editor-action-label'>Push</span>
            </button>
            <button class='btn btn-primary' type='submit' form='entry-editor-form' disabled={editorFormatMismatch} aria-label='Save' title='Save'>
              <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z'/><path d='M17 21v-8H7v8M7 3v5h8'/></svg>
              <span class='editor-action-label'>Save</span>
            </button>
          </div>
      </div>
      <div class='editor-layout'>
        <form id='entry-editor-form' class='panel' onsubmit={handleSubmit}>
        <div>
          <label for='entryTitle'>Title</label>
          <input
            {@attach autofocus}
            required
            id='entryTitle'
            type='text'
            class='input text-strong text-big'
            bind:value={entryTitle}
            placeholder='Title of your entry'
          />
        </div>

        <div class='mt-2 mb-1'>
          <label id='entryMainTextLabel'>Content</label>
        </div>
        {#if editorFormatMismatch}
          <div class='grid gap-1 mb-2'>
            <p class='description'>
              {currentEntryId === null ? 'This draft contains' : 'This entry was saved as'}
              {entryContentType === 1 ? 'HTML (TinyMCE)' : 'Markdown'} content.
              Select that editor in Profile settings to edit it without losing formatting.
            </p>
            <div>
              <button type='button' class='btn btn-secondary' onclick={openProfileSettings}>Open profile settings</button>
            </div>
          </div>
        {:else if preferredEditor === 'tinymce'}
          <TinyMceEditor
            value={entryMainText}
            onChange={(next) => { entryMainText = next; }}
            label='entryMainTextLabel'
          />
        {:else}
          <MarkdownEditor
            value={entryMainText}
            onChange={(next) => { entryMainText = next; }}
            label='entryMainTextLabel'
          />
        {/if}
        </form>
        <aside class='editor-sidebar'>
          <details class='panel entry-info-accordion' bind:open={entryInfoExpanded}>
            <summary class='entry-panel-heading'>
              <h3>Entry information</h3>
              <svg class='entry-chevron' aria-hidden='true' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='m6 9 6 6 6-6'/></svg>
            </summary>
            <div class='entry-detail-content'>
            <div class='entry-info-row'><span>Type</span><span class='flex items-center gap-03'><EntityTypeIcon type={linkedEntityType} /> {linkedEntityType === 'resource' ? 'Resource' : linkedEntityType === 'experiment' ? 'Experiment' : 'Local entry'}</span></div>
            <div class='entry-info-row'><span>Category / Status</span><EntityCategoryStatus remoteDetails={selectedEntry?.remoteDetails} emptyLabel={remoteLinks.length > 0 ? 'Pull to load' : 'Not linked'} /></div>
            <div class='entry-info-row'><span>Created</span><span>{selectedEntry ? new Intl.DateTimeFormat(undefined, {dateStyle: 'medium', timeStyle: 'short'}).format(new Date(selectedEntry.createdAt)) : 'Not saved yet'}</span></div>
            <div class='entry-info-row'><span>Modified</span><span>{selectedEntry ? new Intl.DateTimeFormat(undefined, {dateStyle: 'medium', timeStyle: 'short'}).format(new Date(selectedEntry.modifiedAt)) : 'Not saved yet'}</span></div>
            <div class='entry-info-row'><span>Content type</span><span>{entryContentType === 1 ? 'HTML (TinyMCE)' : 'Markdown'}</span></div>
            </div>
          </details>
          {#key uploadsRefreshKey}
            <UploadsPanel
              {profileUuid}
              entryId={currentEntryId}
              {ensureEntrySaved}
              compact
            />
          {/key}
          <EntryReadonlyPanels
            remoteDetails={selectedEntry?.remoteDetails}
            source={selectedEntry?.remoteSource}
            linked={remoteLinks.length > 0}
          />
        </aside>
      </div>
    </div>
  {/if}
  {#if importModalOpen}
    <EntryImportModal
      {profileUuid}
      onClose={() => { importModalOpen = false; }}
      onImport={confirmImport}
    />
  {/if}
  {#if pushModalOpen}
    <InstancesPushModal
      {profileUuid}
      force={lastFailedPush !== null}
      lockedEntityType={pushMode === 'single' ? linkedEntityType : null}
      onClose={closePushModal}
      onPush={confirmPush}
    />
  {/if}
  {#if remoteModalMode}
    <EntryRemoteModal
      links={remoteLinks}
      mode={remoteModalMode}
      onClose={closeRemoteModal}
      onOpen={(link) => {
        openExternalURL(link.url);
        closeRemoteModal();
      }}
      onPull={confirmPull}
      onUnlink={confirmUnlink}
    />
  {/if}
</div>
