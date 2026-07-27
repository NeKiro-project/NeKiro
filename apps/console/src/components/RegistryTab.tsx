import React, {useMemo, useState} from 'react';
import {AlertTriangle, CheckCircle2, Code2, Database, Layers, Loader2, Plus, ShieldCheck, UploadCloud} from 'lucide-react';

import {buildAgentCard, toPlatformErrorView, type AgentCardV02, type AuthenticationType} from '../api/nekiro';
import type {Agent, PlatformErrorView} from '../types';

interface RegistryTabProps {
  agents: Agent[];
  draftAgents: Agent[];
  onRegisterAgent: (card: AgentCardV02) => Promise<Agent>;
  onPublishAgent: (agent: Agent) => Promise<void>;
  catalogLoading: boolean;
  catalogError: PlatformErrorView | null;
  catalogReady: boolean;
  defaultOwnerId: string;
  defaultOwnerName: string;
  searchQuery: string;
}

const defaultCapabilities = JSON.stringify({
  capabilities: [
    {
      id: 'runtime.echo',
      name: 'Runtime Echo',
      description: 'Echoes structured input.',
      inputSchema: {type: 'object'},
      outputSchema: {type: 'object'},
      requiredPermissions: [],
    },
  ],
}, null, 2);

export default function RegistryTab(props: RegistryTabProps) {
  const {
    agents,
    draftAgents,
    onRegisterAgent,
    onPublishAgent,
    catalogLoading,
    catalogError,
    catalogReady,
    defaultOwnerId,
    defaultOwnerName,
    searchQuery,
  } = props;
  const [showForm, setShowForm] = useState(false);
  const [selectedAgentKey, setSelectedAgentKey] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [localError, setLocalError] = useState<PlatformErrorView | null>(null);
  const [agentId, setAgentId] = useState('runtime.echo');
  const [name, setName] = useState('Runtime Echo Agent');
  const [description, setDescription] = useState('Echoes structured input through the A2A JSON-RPC profile.');
  const [ownerId, setOwnerId] = useState(defaultOwnerId);
  const [ownerName, setOwnerName] = useState(defaultOwnerName);
  const [version, setVersion] = useState('1.0.0');
  const [endpoint, setEndpoint] = useState('');
  const [authentication, setAuthentication] = useState<AuthenticationType>('none');
  const [permissionsText, setPermissionsText] = useState('');
  const [capabilitiesJson, setCapabilitiesJson] = useState(defaultCapabilities);

  const allAgents = useMemo(() => [...draftAgents, ...agents], [agents, draftAgents]);
  const selectedAgent = useMemo(() => allAgents.find((agent) => agentKey(agent) === selectedAgentKey) ?? allAgents[0], [allAgents, selectedAgentKey]);
  const filteredDraftAgents = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    if (!query) return draftAgents;
    return draftAgents.filter((agent) => matchesAgentQuery(agent, query));
  }, [draftAgents, searchQuery]);
  const filteredPublishedAgents = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    if (!query) return agents;
    return agents.filter((agent) => matchesAgentQuery(agent, query));
  }, [agents, searchQuery]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setLocalError(null);
    setSubmitting(true);
    try {
      const card = buildAgentCard({
        agentId,
        name,
        ownerId,
        ownerDisplayName: ownerName,
        description,
        version,
        endpoint,
        authentication,
        permissions: parsePermissions(permissionsText),
        capabilitiesJson,
        limits: {timeoutMs: 30000, maxInputBytes: 1048576, maxOutputBytes: 1048576, streaming: true},
      });
      const registeredAgent = await onRegisterAgent(card);
      setShowForm(false);
      setSelectedAgentKey(agentKey(registeredAgent));
    } catch (error) {
      setLocalError(toPlatformErrorView(error, 'Unable to register Agent Card.'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="h-full flex flex-col gap-5">
      <div className="glass-page-header flex items-start justify-between gap-4">
        <div>
          <div className="font-mono-label text-[10px] uppercase tracking-[0.24em] text-brand-primary mb-2">Registry</div>
          <h2 className="text-2xl font-bold text-brand-on-surface">Agent Card Catalog</h2>
          <p className="text-sm text-brand-on-surface-variant mt-1 max-w-3xl">Live v3 Catalog surface for Agent Card v0.2 registration and discovery. Trust and Release lifecycle operations are handled in Trusted Publication.</p>
        </div>
        <button onClick={() => setShowForm((value) => !value)} className="px-4 py-2 rounded-lg bg-brand-primary text-brand-on-primary text-xs font-semibold flex items-center gap-2 hover:opacity-95">
          <Plus size={15} /> Register Agent Card
        </button>
      </div>

      <ErrorBanner error={catalogError ?? localError} />

      <div className="grid grid-cols-4 gap-4 max-[900px]:grid-cols-2">
        <GlassStat icon={<Database size={15} />} label="Visible cards" value={String(agents.length)} detail="from live Catalog" />
        <GlassStat icon={<CheckCircle2 size={15} />} label="Published" value={String(agents.filter((agent) => agent.status === 'published').length)} detail="discoverable versions" accent="emerald" />
        <GlassStat icon={<Layers size={15} />} label="Drafts" value={String(draftAgents.length)} detail="awaiting publish" accent="amber" />
        <GlassStat
          icon={<ShieldCheck size={15} />}
          label="Gateway"
          value={catalogError ? 'ERROR' : catalogLoading ? 'SYNC' : catalogReady ? 'LIVE' : 'WAIT'}
          detail={catalogError ? (catalogError.code ?? 'Catalog request failed') : catalogReady ? 'Northbound /v3' : 'Awaiting first response'}
          accent="violet"
        />
      </div>

      {showForm && (
        <form onSubmit={handleSubmit} className="grid grid-cols-2 gap-4 bg-brand-low border border-brand-outline-variant rounded-xl p-4">
          <Field label="Agent ID" value={agentId} onChange={setAgentId} />
          <Field label="Name" value={name} onChange={setName} />
          <Field label="Owner ID" value={ownerId} onChange={setOwnerId} />
          <Field label="Owner display name" value={ownerName} onChange={setOwnerName} />
          <Field label="Version" value={version} onChange={setVersion} />
          <Field label="A2A endpoint" value={endpoint} onChange={setEndpoint} />
          <label htmlFor="registration-authentication" className="flex flex-col gap-1 text-xs text-brand-on-surface-variant">
            Authentication
            <select id="registration-authentication" aria-label="Authentication" value={authentication} onChange={(event) => setAuthentication(event.target.value as AuthenticationType)} className="bg-brand-lowest border border-brand-outline-variant rounded px-3 py-2 text-brand-on-surface outline-none">
              <option value="none">none</option>
              <option value="api_key">api_key</option>
              <option value="http_bearer">http_bearer</option>
              <option value="oauth2_client_credentials">oauth2_client_credentials</option>
              <option value="mutual_tls">mutual_tls</option>
            </select>
          </label>
          <Field label="Permissions, one per line as ID: description" value={permissionsText} onChange={setPermissionsText} />
          <label className="col-span-2 flex flex-col gap-1 text-xs text-brand-on-surface-variant">
            Description
            <textarea value={description} onChange={(event) => setDescription(event.target.value)} rows={2} className="bg-brand-lowest border border-brand-outline-variant rounded px-3 py-2 text-brand-on-surface outline-none" />
          </label>
          <label className="col-span-2 flex flex-col gap-1 text-xs text-brand-on-surface-variant">
            Capabilities JSON
            <textarea aria-label="Capabilities JSON" value={capabilitiesJson} onChange={(event) => setCapabilitiesJson(event.target.value)} rows={9} className="font-mono-code bg-brand-lowest border border-brand-outline-variant rounded px-3 py-2 text-brand-on-surface outline-none" />
          </label>
          <div className="col-span-2 flex justify-end gap-3">
            <button type="button" onClick={() => setShowForm(false)} className="px-4 py-2 rounded border border-brand-outline-variant text-xs text-brand-on-surface-variant">Cancel</button>
            <button disabled={submitting} className="px-4 py-2 rounded bg-brand-primary text-brand-on-primary text-xs font-semibold disabled:opacity-50 flex items-center gap-2">
              {submitting ? <Loader2 size={14} className="animate-spin" /> : <UploadCloud size={14} />} Submit draft
            </button>
          </div>
        </form>
      )}

      <div className="glass-split-grid grid grid-cols-[minmax(360px,0.95fr)_minmax(420px,1.05fr)] gap-5 min-h-0 flex-1">
        <div className="bg-brand-low border border-brand-outline-variant rounded-xl overflow-hidden min-h-0 flex flex-col">
          <div className="px-4 py-3 border-b border-brand-outline-variant/60 flex items-center justify-between">
            <span className="text-xs font-bold text-brand-on-surface">Registry results</span>
            {catalogLoading && <Loader2 size={14} className="animate-spin text-brand-primary" />}
          </div>
          <div className="overflow-y-auto">
            <AgentListSection
              title="My drafts"
              description="Server-returned drafts from this browser session. Complete Trusted Publication before installation."
              agents={filteredDraftAgents}
              emptyMessage="No draft Agent Cards in this session. Submit draft to stage one for publishing."
              selectedAgent={selectedAgent}
              onSelect={(agent) => setSelectedAgentKey(agentKey(agent))}
            />
            <AgentListSection
              title="Published catalog"
              description="Live discovery returns published Catalog versions. Catalog publication is not Trusted Publication."
              agents={filteredPublishedAgents}
              emptyMessage="No published Catalog entries returned. This is an empty result, not a mock fallback."
              selectedAgent={selectedAgent}
              onSelect={(agent) => setSelectedAgentKey(agentKey(agent))}
            />
          </div>
        </div>

        <div className="bg-brand-low border border-brand-outline-variant rounded-xl overflow-hidden min-h-0 flex flex-col">
          {selectedAgent ? (
            <>
              <div className="px-4 py-3 border-b border-brand-outline-variant/60 flex items-start justify-between gap-3">
                <div>
                  <div className="text-lg font-bold text-brand-on-surface">{selectedAgent.name}</div>
                  <div className="text-xs text-brand-on-surface-variant mt-1">Owner {selectedAgent.owner} ({selectedAgent.ownerId})</div>
                </div>
                <div className="flex gap-2">
                  {selectedAgent.status === 'draft' && <ActionButton icon={<UploadCloud size={13} />} label="Publish to Catalog" onClick={() => onPublishAgent(selectedAgent)} />}
                  {selectedAgent.status === 'published' && <span className="text-[10px] text-brand-on-surface-variant border border-brand-outline-variant rounded px-2 py-1">Review Release in Trusted Publication</span>}
                </div>
              </div>
              <div className="p-4 grid grid-cols-3 gap-3 text-xs border-b border-brand-outline-variant/40">
                <Fact label="Version" value={selectedAgent.version} />
                <Fact label="Registered" value={selectedAgent.registeredAt || 'n/a'} />
                <Fact label="Published" value={selectedAgent.publishedAt || 'not published'} />
              </div>
              <div className="p-4 border-b border-brand-outline-variant/40">
                <div className="flex items-center gap-2 text-xs font-bold text-brand-on-surface mb-2"><CheckCircle2 size={14} className="text-green-400" />Declared permissions</div>
                {selectedAgent.permissions.length === 0 ? (
                  <p className="text-xs text-brand-on-surface-variant">No permissions declared; installation can submit an explicit empty acceptedPermissions array.</p>
                ) : (
                  <div className="space-y-2">
                    {selectedAgent.permissions.map((permission) => (
                      <div key={permission.id} className="bg-brand-lowest border border-brand-outline-variant rounded p-2">
                        <div className="font-mono-code text-[11px] text-brand-primary">{permission.id}</div>
                        <div className="text-xs text-brand-on-surface-variant mt-1">{permission.description}</div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
              <div className="p-4 min-h-0 flex-1 overflow-hidden flex flex-col">
                <div className="flex items-center gap-2 text-xs font-bold text-brand-on-surface mb-2"><Code2 size={14} />Agent Card JSON</div>
                <pre className="flex-1 overflow-auto bg-brand-lowest border border-brand-outline-variant rounded p-3 text-[11px] font-mono-code text-brand-on-surface-variant whitespace-pre-wrap">{selectedAgent.schema}</pre>
              </div>
            </>
          ) : (
            <div className="p-8 text-center text-sm text-brand-on-surface-variant">Select a Catalog entry to inspect the server-returned Agent Card.</div>
          )}
        </div>
      </div>
    </div>
  );
}

function parsePermissions(value: string) {
  return value.split('\n').map((line) => line.trim()).filter(Boolean).map((line) => {
    const [id, ...rest] = line.split(':');
    const permissionID = id.trim();
    const description = rest.join(':').trim();
    if (!permissionID || !description) throw new Error('Each permission must include an ID and description separated by a colon.');
    return {id: permissionID, description};
  });
}

function AgentListSection({
  title,
  description,
  agents,
  emptyMessage,
  selectedAgent,
  onSelect,
}: {
  title: string;
  description: string;
  agents: Agent[];
  emptyMessage: string;
  selectedAgent?: Agent;
  onSelect: (agent: Agent) => void;
}) {
  return (
    <section className="border-b border-brand-outline-variant/60 last:border-b-0">
      <div className="px-4 py-3 bg-brand-lowest/40 border-b border-brand-outline-variant/40">
        <div className="flex items-center justify-between gap-3">
          <span className="text-[11px] font-bold uppercase tracking-[0.18em] text-brand-on-surface">{title}</span>
          <span className="font-mono-code text-[10px] text-brand-on-surface-variant">{agents.length}</span>
        </div>
        <p className="text-[11px] text-brand-on-surface-variant mt-1">{description}</p>
      </div>
      <div className="divide-y divide-brand-outline-variant/40">
        {agents.length === 0 ? (
          <div className="p-6 text-center text-sm text-brand-on-surface-variant">{emptyMessage}</div>
        ) : agents.map((agent) => (
          <div key={agentKey(agent)}>
            <AgentRow
              agent={agent}
              selected={selectedAgent ? agentKey(selectedAgent) === agentKey(agent) : false}
              onSelect={onSelect}
            />
          </div>
        ))}
      </div>
    </section>
  );
}

function AgentRow({agent, selected, onSelect}: {agent: Agent; selected: boolean; onSelect: (agent: Agent) => void}) {
  return (
    <button onClick={() => onSelect(agent)} className={'w-full text-left p-4 hover:bg-brand-container transition-colors ' + (selected ? 'bg-brand-primary-container/20' : '')}>
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="font-semibold text-brand-on-surface">{agent.name}</div>
          <div className="font-mono-code text-[11px] text-brand-on-surface-variant mt-1"><span>{agent.id}</span><span> @ {agent.version}</span></div>
        </div>
        <StatusBadge status={agent.status} />
      </div>
      <p className="text-xs text-brand-on-surface-variant mt-2 line-clamp-2">{agent.description}</p>
      <div className="flex flex-wrap gap-1.5 mt-3">
        {agent.tags.map((tag) => <span key={tag} className="text-[10px] px-2 py-0.5 rounded bg-brand-container border border-brand-outline-variant text-brand-secondary">{tag}</span>)}
      </div>
    </button>
  );
}

function matchesAgentQuery(agent: Agent, query: string): boolean {
  return [agent.id, agent.name, agent.owner, agent.ownerId, agent.description, agent.version, agent.status, agent.tags.join(' ')].join(' ').toLowerCase().includes(query);
}

function agentKey(agent: Agent): string {
  return agent.id + '@' + agent.version;
}

function Field({label, value, onChange}: {label: string; value: string; onChange: (value: string) => void}) {
  return (
    <label className="flex flex-col gap-1 text-xs text-brand-on-surface-variant">
      {label}
      <input value={value} onChange={(event) => onChange(event.target.value)} className="bg-brand-lowest border border-brand-outline-variant rounded px-3 py-2 text-brand-on-surface outline-none" />
    </label>
  );
}

function StatusBadge({status}: {status: Agent['status']}) {
  const cls = status === 'published' ? 'text-green-300 border-green-400/30 bg-green-500/10' : status === 'draft' ? 'text-brand-primary border-brand-primary/30 bg-brand-primary/10' : 'text-brand-error border-brand-error/30 bg-brand-error-container/10';
  return <span className={'glass-status-badge text-[10px] px-2 py-0.5 rounded border uppercase font-mono-label ' + cls}>{status}</span>;
}

function ActionButton({icon, label, onClick}: {icon: React.ReactNode; label: string; onClick: () => void}) {
  return <button onClick={onClick} className="px-3 py-1.5 rounded bg-brand-container border border-brand-outline-variant text-xs text-brand-on-surface-variant hover:text-brand-on-surface flex items-center gap-1.5">{icon}{label}</button>;
}

function Fact({label, value}: {label: string; value: string}) {
  return (
    <div className="bg-brand-lowest border border-brand-outline-variant rounded p-2 min-w-0">
      <div className="text-[10px] uppercase tracking-wider text-brand-on-surface-variant">{label}</div>
      <div className="font-mono-code text-[11px] text-brand-on-surface mt-1 truncate">{value}</div>
    </div>
  );
}

function GlassStat({icon, label, value, detail, accent}: {icon: React.ReactNode; label: string; value: string; detail: string; accent?: 'emerald' | 'amber' | 'violet'}) {
  const color = accent === 'emerald' ? 'text-emerald-300' : accent === 'amber' ? 'text-amber-300' : accent === 'violet' ? 'text-violet-300' : 'text-indigo-300';
  return (
    <div className="rounded-2xl border border-white/[0.07] bg-white/[0.025] backdrop-blur-xl px-5 py-4 transition-all hover:border-white/[0.14] hover:-translate-y-px">
      <div className="flex items-center gap-2 text-[10px] uppercase tracking-[0.18em] text-slate-500"><span className={color}>{icon}</span>{label}</div>
      <div className="mt-2 text-[25px] font-extrabold tracking-tight text-slate-100">{value}</div>
      <div className="mt-0.5 text-[10.5px] text-slate-500">{detail}</div>
    </div>
  );
}

function ErrorBanner({error}: {error: PlatformErrorView | null}) {
  if (!error) return null;
  return (
    <div className="border border-brand-error/25 bg-brand-error-container/10 rounded-lg p-3 flex items-start gap-3 text-sm text-brand-error">
      <AlertTriangle size={16} className="mt-0.5" />
      <div>
        <div className="font-semibold">{error.code ?? 'ERROR'} · HTTP {error.status}</div>
        <div className="text-brand-on-surface-variant mt-1">{error.message}</div>
        {error.traceId && <div className="font-mono-code text-[11px] mt-1">traceId: {error.traceId}</div>}
      </div>
    </div>
  );
}
