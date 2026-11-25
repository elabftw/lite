
<script lang='ts'>
  type Props = { remoteDetails?: string; source?: string; linked: boolean };
  let {remoteDetails, source, linked}: Props = $props();

  type RemoteLink = {
    title?: string;
    name?: string;
    entityid?: number;
    item_id?: number;
    experiment_id?: number;
    id?: number;
  };

  type Metadata = {
    extra_fields?: Record<string, {type?: string; value?: unknown; unit?: string; units?: string | string[]}>;
  };

  type RemoteComment = { comment?: string; fullname?: string; created_at?: string };
  type RemoteStep = { body?: string; finished?: number | string | boolean };
  type RemoteCompound = { id?: number; name?: string; iupac_name?: string; cas_number?: string };
  type RemoteList<T> = string | T[] | {data: T[]};

  type RemoteDetails = {
    tags?: string;
    metadata?: string | Metadata;
    itemsLinks?: string | RemoteLink[] | {data: RemoteLink[]};
    experimentsLinks?: string | RemoteLink[] | {data: RemoteLink[]};
    metadataLabels?: Record<string, string>;
    comments?: RemoteList<RemoteComment>;
    steps?: RemoteList<RemoteStep>;
    compoundsLinks?: RemoteList<RemoteCompound>;
  };

  // Parse JSON strings while allowing already decoded values
  function parseJSON<T>(value: string | T | null | undefined, fallback: T): T {
    if (value == null) return fallback;
    if (typeof value !== 'string') return value;
    try { return JSON.parse(value) ?? fallback; } catch { return fallback; }
  }
  // Extract arrays or responses wrapped in data
  function remoteList<T>(value: RemoteList<T> | undefined): T[] {
    const parsed = parseJSON<T[] | {data: T[]}>(value, []);
    return Array.isArray(parsed) ? parsed : Array.isArray(parsed.data) ? parsed.data : [];
  }

  // Parse the saved remote information
  const data = $derived(parseJSON<RemoteDetails>(remoteDetails, {}));
  // Split pipe-separated tags into a list
  const tags = $derived(data.tags?.split('|').filter(Boolean) ?? []);
  const comments = $derived(remoteList(data.comments));
  const steps = $derived(remoteList(data.steps));
  const compounds = $derived(remoteList(data.compoundsLinks));

  // The encrypted snapshot stores labels separately from the original metadata IDs.
  function metadataValue(type: string | undefined, value: unknown): string {
    const values = Array.isArray(value) ? value : [value];
    return values.map(v => {
      const raw = String(v ?? '');
      return data.metadataLabels?.[`${type}:${raw.trim()}`] || raw;
    }).join(',\n');
  }

  // Metadata is itself a JSON string in the eLabFTW API response
  const metadata = $derived(
    Object.entries(parseJSON<Metadata>(data.metadata, {}).extra_fields ?? {}).map(([name, field]) => ({
      name,
      value: metadataValue(field.type, field.value) + (field.unit ? ` ${field.unit}` : typeof field.units === 'string' ? ` ${field.units}` : ''),
    }))
  );
  // Combine linked resources and experiments
  const links = $derived([
    ...remoteList(data.itemsLinks).map(item => ({
      type: 'Resource',
      name: String(item.title ?? item.name ?? item.entityid ?? item.item_id ?? item.id ?? ''),
    })),
    ...remoteList(data.experimentsLinks).map(item => ({
      type: 'Experiment',
      name: String(item.title ?? item.name ?? item.entityid ?? item.experiment_id ?? item.id ?? ''),
    })),
  ]);

  const sections = $derived([
    {name: 'Tags', count: tags.length, available: !!remoteDetails},
    {name: 'Metadata', count: metadata.length, available: !!remoteDetails},
    {name: 'Links', count: links.length, available: !!remoteDetails},
    {name: 'Comments', count: comments.length, available: data.comments != null},
    {name: 'Steps', count: steps.length, available: data.steps != null},
    {name: 'Compounds', count: compounds.length, available: data.compoundsLinks != null},
  ]);

  const hint = $derived(linked ? 'Pull from eLabFTW to load this information.' : 'Not linked to an eLabFTW instance.');
</script>

<!-- Read-only panels for the saved remote snapshot -->
{#each sections as section (section.name)}
  <details class='panel'>
    <summary class='entry-panel-heading'>
      <h3>{section.name}{#if section.available} ({section.count}){/if}</h3>
      <svg class='entry-chevron' aria-hidden='true' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='m6 9 6 6 6-6'/></svg>
    </summary>
    <div class='entry-detail-content'>
      {#if !section.available}
        <p class='description'>{hint}</p>
      {:else if section.name === 'Tags'}
        {#if tags.length}
          <div class='entry-tag-list'>{#each tags as tag, i (i)}<span class='entry-tag'>{tag}</span>{/each}</div>
        {:else}<p class='description'>No tags.</p>{/if}
      {:else if section.name === 'Metadata'}
        {#if metadata.length}
          {#each metadata as field (field.name)}
            <div class='entry-readonly-row entry-metadata-row'><span>{field.name}</span><span>{field.value || '—'}</span></div>
          {/each}
        {:else}<p class='description'>No custom fields.</p>{/if}
      {:else if section.name === 'Links'}
        {#if links.length}
          {#each links as link, i (i)}
            <div class='entry-readonly-row'><span>{link.name || 'Untitled'}</span><span>{link.type}</span></div>
          {/each}
        {:else}<p class='description'>No linked entries.</p>{/if}
      {:else if section.name === 'Comments'}
        {#each comments as comment, i (i)}
          <div class='entry-readonly-row entry-metadata-row'>
            <span>{comment.fullname || 'Unknown user'}{comment.created_at ? ` · ${comment.created_at}` : ''}</span>
            <span>{comment.comment || ''}</span>
          </div>
        {:else}<p class='description'>No comments.</p>{/each}
      {:else if section.name === 'Steps'}
        {#each steps as step, i (i)}
          <div class='entry-readonly-row entry-metadata-row'>
            <span>{step.body || 'Untitled step'}</span>
            <span>{Number(step.finished) === 1 ? 'Completed' : 'Pending'}</span>
          </div>
        {:else}<p class='description'>No steps.</p>{/each}
      {:else if section.name === 'Compounds'}
        {#each compounds as compound, i (i)}
          <div class='entry-readonly-row entry-metadata-row'>
            <span>{compound.name || compound.iupac_name || `#${compound.id ?? '?'}`}</span>
            <span>{compound.cas_number || ''}</span>
          </div>
        {:else}<p class='description'>No compounds.</p>{/each}
      {/if}
      {#if section.available && source}<p class='description'>Last pulled from {source}</p>{/if}
    </div>
  </details>
{/each}
