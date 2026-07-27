import React, {useEffect, useMemo, useRef, useState} from 'react';
import {AlertTriangle, Database, Loader2, RefreshCw, ShieldCheck, Trash2} from 'lucide-react';

import {NekiroApiError, toPlatformErrorView, type AgentRelease, type NekiroApiClient} from '../api/nekiro';
import {agentKey, isCurrentRequest, matchesPublishedRelease, nextRequestGeneration} from '../consolePolicy';
import type {Agent, Installation, InstallationStatus, PlatformErrorView, Workspace} from '../types';

interface InstallationsTabProps {
  workspace: Workspace | null;
  agents: Agent[];
  installations: Installation[];
  loading: boolean;
  error: PlatformErrorView | null;
  searchQuery: string;
  client: NekiroApiClient;
  onInstallAgent: (agent: Agent, release: AgentRelease, acceptedPermissions: string[]) => Promise<void>;
  onUpdateInstallation: (installation: Installation, status: Exclude<InstallationStatus, 'uninstalled'>) => Promise<void>;
  onUninstall: (installation: Installation) => Promise<boolean>;
  onRefresh: () => void;
}

export default function InstallationsTab({
  workspace,
  agents,
  installations,
  loading,
  error,
  searchQuery,
  client,
  onInstallAgent,
  onUpdateInstallation,
  onUninstall,
  onRefresh,
}: InstallationsTabProps) {
  const publishedAgents = useMemo(() => agents.filter((agent) => agent.status === 'published'), [agents]);
  const [selectedAgentKey, setSelectedAgentKey] = useState('');
  const [versionConstraint, setVersionConstraint] = useState('');
  const [acceptedPermissions, setAcceptedPermissions] = useState<string[]>([]);
  const [releaseId, setReleaseId] = useState('');
  const [preflightRelease, setPreflightRelease] = useState<AgentRelease | null>(null);
  const [preflightLoading, setPreflightLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [localError, setLocalError] = useState<PlatformErrorView | null>(null);
  const [confirmUninstallId, setConfirmUninstallId] = useState<string | null>(null);
  const [busyLifecycle, setBusyLifecycle] = useState(false);
  const preflightGeneration = useRef(0);

  const invalidatePreflight = () => {
    preflightGeneration.current = nextRequestGeneration(preflightGeneration.current);
    setPreflightRelease(null);
  };

  useEffect(() => {
    if (!selectedAgentKey && publishedAgents[0]) {
      const next = publishedAgents[0];
      invalidatePreflight();
      setSelectedAgentKey(agentKey(next));
      setVersionConstraint(next.version);
      setAcceptedPermissions(next.permissions.map((permission) => permission.id).sort());
    }
  }, [publishedAgents, selectedAgentKey]);

  const selectedAgent = publishedAgents.find((agent) => agentKey(agent) === selectedAgentKey);
  const filteredInstallations = installations.filter((installation) => {
    const query = searchQuery.trim().toLowerCase();
    if (!query) return true;
    return [
      installation.installationId,
      installation.workspaceId,
      installation.agentId,
      installation.versionConstraint,
      installation.installedVersion,
      installation.status,
      installation.acceptedPermissions.join(' '),
    ].join(' ').toLowerCase().includes(query);
  });

  const handleSelectAgent = (selectedKey: string) => {
    const agent = publishedAgents.find((item) => agentKey(item) === selectedKey);
    invalidatePreflight();
    setSelectedAgentKey(selectedKey);
    setVersionConstraint(agent?.version ?? '');
    setAcceptedPermissions(agent?.permissions.map((permission) => permission.id).sort() ?? []);
    setReleaseId('');
    setPreflightRelease(null);
    setLocalError(null);
  };

  const handlePreflight = async () => {
    if (!selectedAgent) return;
    const generation = nextRequestGeneration(preflightGeneration.current);
    preflightGeneration.current = generation;
    const requestedAgentKey = selectedAgentKey;
    const requestedReleaseId = releaseId;
    setPreflightLoading(true);
    setLocalError(null);
    try {
      const value = await client.getAgentRelease(requestedReleaseId);
      if (!isCurrentRequest(generation, preflightGeneration.current)) return;
      if (!matchesPublishedRelease(value, selectedAgent)) throw new NekiroApiError(200, 'The selected Release is not a published match for the selected Agent Card.', 'INVALID_RESPONSE');
      if (requestedAgentKey !== selectedAgentKey || requestedReleaseId !== releaseId) return;
      setPreflightRelease(value);
      setVersionConstraint(value.agentCardVersion);
    } catch (value) {
      if (!isCurrentRequest(generation, preflightGeneration.current)) return;
      setPreflightRelease(null);
      setLocalError(toPlatformErrorView(value, 'Unable to preflight the trusted Release.'));
    } finally {
      if (isCurrentRequest(generation, preflightGeneration.current)) setPreflightLoading(false);
    }
  };

  const handleInstall = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!selectedAgent || !preflightRelease) return;
    setSubmitting(true);
    setLocalError(null);
    try {
      await onInstallAgent(selectedAgent, preflightRelease, acceptedPermissions);
    } catch (installError) {
      setLocalError(toPlatformErrorView(installError, 'Unable to install Agent.'));
    } finally {
      setSubmitting(false);
    }
  };

  const togglePermission = (permissionId: string) => {
    setAcceptedPermissions((current) => current.includes(permissionId)
      ? current.filter((item) => item !== permissionId)
      : [...current, permissionId].sort());
  };

  return (
    <div className="h-full flex flex-col gap-5">
      <div className="glass-page-header flex items-start justify-between gap-4">
        <div>
          <div className="font-mono-label text-[10px] uppercase tracking-[0.24em] text-brand-primary mb-2">Installations</div>
          <h2 className="text-2xl font-bold text-brand-on-surface">Workspace Agent Pins</h2>
          <p className="text-sm text-brand-on-surface-variant mt-1 max-w-3xl">
            Preflight an immutable published Release before installing its exact Card version into the current Workspace.
          </p>
        </div>
         <button onClick={onRefresh} disabled={loading || busyLifecycle} className="px-3 py-2 rounded bg-brand-container border border-brand-outline-variant text-xs text-brand-on-surface-variant hover:text-brand-on-surface flex items-center gap-2 disabled:opacity-40">
          <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>

      {!workspace && (
        <div className="border border-brand-primary/25 bg-brand-primary-container/10 rounded-lg p-4 text-sm text-brand-on-surface-variant">
          Select or create a Workspace in the header before installing Agents. The Console will not create mock Workspace state.
        </div>
      )}

      <ErrorBanner error={error ?? localError} />

      <div className="glass-split-grid grid grid-cols-[minmax(360px,0.85fr)_minmax(520px,1.15fr)] gap-5 min-h-0 flex-1">
        <form onSubmit={handleInstall} className="bg-brand-low border border-brand-outline-variant rounded-xl p-4 h-fit">
          <div className="flex items-center gap-2 mb-4">
            <Database size={16} className="text-brand-primary" />
            <div>
              <div className="text-sm font-bold text-brand-on-surface">Install trusted Release</div>
              <div className="text-xs text-brand-on-surface-variant">The Release ID is an explicit provider handoff; Catalog publication alone is not trust.</div>
            </div>
          </div>

          <div className="flex flex-col gap-1 text-xs text-brand-on-surface-variant mb-3">
            <label htmlFor="installation-agent">Published Agent</label>
            <select id="installation-agent" value={selectedAgentKey} onChange={(event) => handleSelectAgent(event.target.value)} disabled={!workspace || publishedAgents.length === 0 || busyLifecycle} className="bg-brand-lowest border border-brand-outline-variant rounded px-3 py-2 text-brand-on-surface outline-none disabled:opacity-50">
              {publishedAgents.length === 0 && <option value="">No published agents returned</option>}
              {publishedAgents.map((agent) => <option key={agentKey(agent)} value={agentKey(agent)}>{agent.name} ({agent.version})</option>)}
            </select>
          </div>

          <div className="flex flex-col gap-1 text-xs text-brand-on-surface-variant mb-3">
            <label htmlFor="trusted-release-id">Trusted Release ID</label>
            <div className="flex gap-2"><input id="trusted-release-id" value={releaseId} onChange={(event) => { invalidatePreflight(); setReleaseId(event.target.value); setLocalError(null); }} disabled={!workspace || preflightLoading || busyLifecycle} placeholder="release-id" className="flex-1 bg-brand-lowest border border-brand-outline-variant rounded px-3 py-2 text-brand-on-surface outline-none disabled:opacity-50" /><button type="button" onClick={handlePreflight} disabled={!workspace || !selectedAgent || !releaseId || preflightLoading || busyLifecycle} className="px-3 py-2 rounded border border-brand-outline-variant bg-brand-container text-xs text-brand-on-surface-variant disabled:opacity-40">{preflightLoading ? 'Reading...' : 'Preflight'}</button></div>
          </div>

          {preflightRelease && <div className="border border-green-400/30 bg-green-500/10 rounded-lg p-3 mb-3 text-xs text-brand-on-surface-variant"><div className="flex items-center gap-2 text-green-300 font-semibold"><ShieldCheck size={14} /> Published Release preflight passed</div><div className="grid grid-cols-2 gap-2 mt-3"><Fact label="Release" value={preflightRelease.releaseId} /><Fact label="Version" value={preflightRelease.agentCardVersion} /><Fact label="Card digest" value={preflightRelease.cardDigest} /><Fact label="Binding" value={preflightRelease.endpointBindingId} /><Fact label="Origin" value={preflightRelease.endpointOrigin} /><Fact label="Path" value={preflightRelease.endpointPath} /></div></div>}

          <label className="flex flex-col gap-1 text-xs text-brand-on-surface-variant mb-3">
            Exact version constraint
            <input value={versionConstraint} readOnly className="bg-brand-lowest border border-brand-outline-variant rounded px-3 py-2 text-brand-on-surface outline-none opacity-70" />
          </label>

          <div className="text-xs font-bold text-brand-on-surface mb-2">Declared permissions</div>
          <div className="space-y-2 mb-4">
            {!selectedAgent && <div className="text-xs text-brand-on-surface-variant">Select a published Agent to review permissions.</div>}
            {selectedAgent && selectedAgent.permissions.length === 0 && (
              <div className="border border-brand-outline-variant rounded p-3 text-xs text-brand-on-surface-variant">This Agent declares no permissions. Submitting installs with acceptedPermissions: [].</div>
            )}
            {selectedAgent?.permissions.map((permission) => (
              <label key={permission.id} className="flex items-start gap-3 bg-brand-lowest border border-brand-outline-variant rounded p-3 cursor-pointer">
                <input type="checkbox" checked={acceptedPermissions.includes(permission.id)} onChange={() => togglePermission(permission.id)} disabled={busyLifecycle} className="mt-1" />
                <span>
                  <span className="font-mono-code text-[11px] text-brand-primary block">{permission.id}</span>
                  <span className="text-xs text-brand-on-surface-variant">{permission.description}</span>
                </span>
              </label>
            ))}
          </div>

          <button disabled={!workspace || !selectedAgent || !preflightRelease || submitting || busyLifecycle} className="w-full px-4 py-2 rounded bg-brand-primary text-brand-on-primary text-xs font-semibold disabled:opacity-50 flex items-center justify-center gap-2">
            {submitting ? <Loader2 size={14} className="animate-spin" /> : <ShieldCheck size={14} />}
            Install exact pin
          </button>
        </form>

        <div className="bg-brand-low border border-brand-outline-variant rounded-xl overflow-hidden min-h-0 flex flex-col">
          <div className="px-4 py-3 border-b border-brand-outline-variant/60 flex items-center justify-between">
            <span className="text-xs font-bold text-brand-on-surface">Current and historical Installations</span>
            {loading && <Loader2 size={14} className="animate-spin text-brand-primary" />}
          </div>
          <div className="overflow-y-auto divide-y divide-brand-outline-variant/40">
            {filteredInstallations.length === 0 ? (
              <div className="p-8 text-center text-sm text-brand-on-surface-variant">No Installation facts returned for this Workspace.</div>
            ) : filteredInstallations.map((installation) => (
              <div key={installation.installationId} className="p-4">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="font-semibold text-brand-on-surface">{installation.agentId}</div>
                    <div className="font-mono-code text-[11px] text-brand-on-surface-variant mt-1">{installation.installationId}</div>
                  </div>
                  <StatusBadge status={installation.status} />
                </div>
                <div className="grid grid-cols-3 gap-2 mt-3 text-xs">
                  <Fact label="Constraint" value={installation.versionConstraint} />
                  <Fact label="Pinned" value={installation.installedVersion} />
                  <Fact label="Updated" value={installation.updatedAt} />
                </div>
                <div className="grid grid-cols-1 gap-2 mt-2 text-xs"><Fact label="Installed Release" value={installation.installedReleaseId ?? 'not returned (legacy/untrusted record)'} /></div>
                <div className="mt-3 flex flex-wrap gap-1.5">
                  {installation.acceptedPermissions.length === 0
                    ? <span className="text-[10px] px-2 py-0.5 rounded bg-brand-container border border-brand-outline-variant text-brand-on-surface-variant">acceptedPermissions: []</span>
                    : installation.acceptedPermissions.map((permission) => <span key={permission} className="text-[10px] px-2 py-0.5 rounded bg-brand-container border border-brand-outline-variant text-brand-secondary">{permission}</span>)}
                </div>
                <div className="mt-3 flex gap-2">
                  {installation.status === 'enabled' && <button disabled={busyLifecycle} onClick={() => runInstallationAction(installation, 'disabled')} className="px-3 py-1.5 rounded border border-brand-outline-variant text-xs text-brand-on-surface-variant hover:text-brand-on-surface disabled:opacity-40">Disable</button>}
                  {installation.status === 'disabled' && <button disabled={busyLifecycle} onClick={() => runInstallationAction(installation, 'enabled')} className="px-3 py-1.5 rounded border border-brand-outline-variant text-xs text-brand-on-surface-variant hover:text-brand-on-surface disabled:opacity-40">Enable</button>}
                  {installation.status === 'disabled' && confirmUninstallId !== installation.installationId && <button disabled={busyLifecycle} onClick={() => setConfirmUninstallId(installation.installationId)} className="px-3 py-1.5 rounded border border-brand-error/30 text-xs text-brand-error flex items-center gap-1.5 disabled:opacity-40"><Trash2 size={12} />Uninstall</button>}
                  {installation.status === 'disabled' && confirmUninstallId === installation.installationId && <button disabled={busyLifecycle} onClick={() => runUninstall(installation)} className="px-3 py-1.5 rounded bg-brand-error-container text-xs text-brand-error disabled:opacity-40">Confirm uninstall</button>}
                  {installation.status === 'uninstalled' && <span className="text-xs text-brand-on-surface-variant">Uninstalled at {installation.uninstalledAt}</span>}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );

  async function runInstallationAction(installation: Installation, status: Exclude<InstallationStatus, 'uninstalled'>) {
    setBusyLifecycle(true);
    try {
      await onUpdateInstallation(installation, status);
    } finally {
      setBusyLifecycle(false);
    }
  }

  async function runUninstall(installation: Installation) {
    setBusyLifecycle(true);
    try {
      if (await onUninstall(installation)) setConfirmUninstallId(null);
    } finally {
      setBusyLifecycle(false);
    }
  }
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

function StatusBadge({status}: {status: InstallationStatus}) {
  const cls = status === 'enabled' ? 'text-green-300 border-green-400/30 bg-green-500/10' : status === 'disabled' ? 'text-brand-primary border-brand-primary/30 bg-brand-primary/10' : 'text-brand-error border-brand-error/30 bg-brand-error-container/10';
  return <span className={'glass-status-badge text-[10px] px-2 py-0.5 rounded border uppercase font-mono-label ' + cls}>{status}</span>;
}

function Fact({label, value}: {label: string; value: string}) {
  return (
    <div className="bg-brand-lowest border border-brand-outline-variant rounded p-2 min-w-0">
      <div className="text-[10px] uppercase tracking-wider text-brand-on-surface-variant">{label}</div>
      <div className="font-mono-code text-[11px] text-brand-on-surface mt-1 truncate">{value}</div>
    </div>
  );
}
