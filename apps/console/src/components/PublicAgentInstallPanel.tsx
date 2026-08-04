import React, {useEffect, useState} from 'react';
import {AlertTriangle, Loader2, ShieldCheck} from 'lucide-react';

import {toPlatformErrorView, validatePublicInstallation, type NekiroApiClient, type PublicAgentRelease, type PublicAgentShare} from '../api/nekiro';
import {parsePublicAgentUrl} from '../publicAgentUrl';
import type {PlatformErrorView, Workspace} from '../types';

interface PublicAgentInstallPanelProps {
  client: NekiroApiClient;
  workspace: Workspace | null;
  onInstalled?: () => Promise<void> | void;
  initialUrl?: string;
}

export default function PublicAgentInstallPanel({client, workspace, onInstalled, initialUrl = ''}: PublicAgentInstallPanelProps) {
  const [url, setUrl] = useState(initialUrl);
  const [share, setShare] = useState<PublicAgentShare | null>(null);
  const [selectedReleaseID, setSelectedReleaseID] = useState('');
  const [acceptedPermissions, setAcceptedPermissions] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [installedReleaseID, setInstalledReleaseID] = useState('');
  const [error, setError] = useState<PlatformErrorView | null>(null);
  const selectedRelease = share?.releases.find((release) => release.releaseId === selectedReleaseID);

  const resolveURL = async (candidate: string) => {
    setLoading(true);
    setError(null);
    setShare(null);
    setSelectedReleaseID('');
    setAcceptedPermissions([]);
    setInstalledReleaseID('');
    try {
      const publicAgentID = parsePublicAgentUrl(candidate, import.meta.env.VITE_NEKIRO_PUBLIC_AGENT_ORIGIN);
      const value = await client.resolvePublicAgent(publicAgentID);
      setShare(value);
    } catch (value) {
      setError(toPlatformErrorView(value, 'Unable to resolve this canonical Public Agent URL.'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (initialUrl) void resolveURL(initialUrl);
  // The initial direct-route URL is resolved exactly once for each route value.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialUrl]);

  const resolve = (event: React.FormEvent) => {
    event.preventDefault();
    void resolveURL(url);
  };

  const install = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!workspace || !selectedRelease) return;
    setSubmitting(true);
    setError(null);
    try {
      const installation = await client.installAgent(workspace.workspaceId, {
        agentId: selectedRelease.agentId,
        versionConstraint: selectedRelease.agentCardVersion,
        acceptedPermissions,
      });
      validatePublicInstallation(installation, selectedRelease, workspace.workspaceId, acceptedPermissions);
      setInstalledReleaseID(selectedRelease.releaseId);
      await onInstalled?.();
    } catch (value) {
      setError(toPlatformErrorView(value, 'Unable to install the selected exact trusted Release.'));
    } finally {
      setSubmitting(false);
    }
  };

  const togglePermission = (permissionID: string) => setAcceptedPermissions((current) => current.includes(permissionID)
    ? current.filter((item) => item !== permissionID)
    : [...current, permissionID].sort());

  return <section className="bg-brand-low border border-brand-outline-variant rounded-xl p-4 space-y-4">
    <div>
      <div className="font-mono-label text-[10px] uppercase tracking-[0.2em] text-brand-primary">Public Share</div>
      <h3 className="text-lg font-bold text-brand-on-surface mt-1">Install from a canonical NeKiro URL</h3>
      <p className="text-xs text-brand-on-surface-variant mt-1">This public URL is discovery metadata, never an Agent endpoint or invitation secret.</p>
    </div>
    <form onSubmit={resolve} className="flex gap-2">
      <input aria-label="Public Agent URL" value={url} onChange={(event) => setUrl(event.target.value)} placeholder="https://agents.nekiro.dev/a/agt_..." className="flex-1 bg-brand-lowest border border-brand-outline-variant rounded px-3 py-2 text-sm text-brand-on-surface" />
      <button disabled={loading || !url} className="px-3 py-2 rounded bg-brand-container border border-brand-outline-variant text-xs text-brand-on-surface disabled:opacity-40">{loading ? 'Resolving…' : 'Resolve'}</button>
    </form>
    {error && <ErrorBanner error={error} />}
    {share && <div className="space-y-4">
      <div className="border border-brand-outline-variant rounded-lg p-3 text-xs text-brand-on-surface-variant">
        <div className="font-semibold text-brand-on-surface">{share.availability === 'installable' ? 'Installable trusted Releases' : 'Not currently installable'}</div>
        <div className="font-mono-code mt-1">{share.publicAgentId}</div>
        <div className="mt-1">Registered {share.registeredAt}</div>
      </div>
      {share.availability === 'not_installable' ? <div className="text-sm text-brand-on-surface-variant">No published trusted Release is available. Draft Card details remain private.</div> : <form onSubmit={install} className="space-y-3">
        <label className="flex flex-col gap-1 text-xs text-brand-on-surface-variant">Select one exact Release (required)
          <select aria-label="Exact public Release" value={selectedReleaseID} onChange={(event) => { setSelectedReleaseID(event.target.value); setAcceptedPermissions([]); setInstalledReleaseID(''); }} className="bg-brand-lowest border border-brand-outline-variant rounded px-3 py-2 text-brand-on-surface">
            <option value="">Select a Release</option>
            {share.releases.map((release) => <option key={release.releaseId} value={release.releaseId}>{release.name} · {release.agentCardVersion} · {release.releaseId}</option>)}
          </select>
        </label>
        {selectedRelease && <ReleaseReview release={selectedRelease} acceptedPermissions={acceptedPermissions} togglePermission={togglePermission} />}
        {!workspace && <div className="text-xs text-brand-on-surface-variant border border-brand-primary/25 rounded p-3">Authenticate and select an authorized Workspace before installation.</div>}
        <button disabled={!workspace || !selectedRelease || submitting} className="w-full px-4 py-2 rounded bg-brand-primary text-brand-on-primary text-xs font-semibold disabled:opacity-50 flex justify-center gap-2">{submitting ? <Loader2 size={14} className="animate-spin" /> : <ShieldCheck size={14} />}Install exact Release</button>
        {installedReleaseID && <div className="border border-green-400/30 bg-green-500/10 rounded-lg p-3 text-xs text-green-300">Installed exact Release <span className="font-mono-code">{installedReleaseID}</span>.</div>}
      </form>}
    </div>}
  </section>;
}

function ReleaseReview({release, acceptedPermissions, togglePermission}: {release: PublicAgentRelease; acceptedPermissions: string[]; togglePermission: (id: string) => void}) {
  return <div className="border border-brand-outline-variant rounded-lg p-3 text-xs text-brand-on-surface-variant space-y-2">
    <div className="grid grid-cols-2 gap-2"><Fact label="Agent" value={release.name} /><Fact label="Version" value={release.agentCardVersion} /><Fact label="Release" value={release.releaseId} /><Fact label="Card digest" value={release.cardDigest} /><Fact label="Owner" value={`${release.owner.displayName} (${release.owner.id})`} /><Fact label="Published" value={release.publishedAt} /></div>
    <div className="font-semibold text-brand-on-surface pt-2">Declared permissions</div>
    {release.permissions.length === 0 ? <div>No permissions declared.</div> : release.permissions.map((permission) => <label key={permission.id} className="flex items-start gap-2"><input type="checkbox" checked={acceptedPermissions.includes(permission.id)} onChange={() => togglePermission(permission.id)} /><span><span className="font-mono-code text-brand-primary">{permission.id}</span> — {permission.description}</span></label>)}
  </div>;
}

function Fact({label, value}: {label: string; value: string}) { return <div><div className="text-[10px] uppercase tracking-wider">{label}</div><div className="font-mono-code truncate text-brand-on-surface mt-0.5">{value}</div></div>; }
function ErrorBanner({error}: {error: PlatformErrorView}) { return <div className="border border-brand-error/25 bg-brand-error-container/10 rounded-lg p-3 flex gap-2 text-sm text-brand-error"><AlertTriangle size={15} /><div><div>{error.code ?? 'ERROR'} · {error.message}</div>{error.traceId && <div className="font-mono-code text-[10px] mt-1">traceId: {error.traceId}</div>}</div></div>; }
