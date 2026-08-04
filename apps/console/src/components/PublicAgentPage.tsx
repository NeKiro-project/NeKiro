import React, {useEffect, useMemo, useState} from 'react';

import {NekiroApiClient, toPlatformErrorView} from '../api/nekiro';
import type {PlatformErrorView, Workspace} from '../types';
import PublicAgentInstallPanel from './PublicAgentInstallPanel';

export default function PublicAgentPage() {
  const ownerToken = import.meta.env.VITE_NEKIRO_OWNER_TOKEN;
  const defaultWorkspaceID = import.meta.env.VITE_NEKIRO_DEFAULT_WORKSPACE_ID;
  const hasOwnerContext = typeof ownerToken === 'string' && ownerToken !== '' && typeof defaultWorkspaceID === 'string' && defaultWorkspaceID !== '';
  const client = useMemo(() => new NekiroApiClient({
    baseUrl: import.meta.env.VITE_NEKIRO_API_BASE_URL,
    publicAgentOrigin: import.meta.env.VITE_NEKIRO_PUBLIC_AGENT_ORIGIN,
    ...(hasOwnerContext ? {token: ownerToken} : {anonymousOnly: true}),
  }), [hasOwnerContext, ownerToken]);
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [workspaceError, setWorkspaceError] = useState<PlatformErrorView | null>(null);
  const initialURL = import.meta.env.VITE_NEKIRO_PUBLIC_AGENT_ORIGIN
    + window.location.pathname
    + window.location.search
    + window.location.hash;

  useEffect(() => {
    if (!hasOwnerContext) return;
    let active = true;
    void client.getWorkspace(defaultWorkspaceID).then((value) => {
      if (active) setWorkspace(value);
    }).catch((value) => {
      if (active) setWorkspaceError(toPlatformErrorView(value, 'Unable to load the configured Workspace owner context.'));
    });
    return () => { active = false; };
  }, [client, defaultWorkspaceID, hasOwnerContext]);

  return <main className="min-h-screen bg-brand-bg text-brand-on-surface p-6 md:p-12">
    <div className="max-w-4xl mx-auto space-y-5">
      <header>
        <div className="font-mono-label text-xs uppercase tracking-[0.25em] text-brand-primary">NeKiro Public Agent</div>
        <h1 className="text-3xl font-bold mt-2">Review a shared Agent</h1>
        <p className="text-sm text-brand-on-surface-variant mt-2">Resolve public trusted publication facts, then explicitly select one exact Release and its permissions.</p>
      </header>
      {workspaceError && <div className="border border-brand-error/25 rounded-lg p-3 text-sm text-brand-error">{workspaceError.message}</div>}
      <PublicAgentInstallPanel client={client} workspace={workspace} initialUrl={initialURL} />
      {!hasOwnerContext && <a href="/" className="inline-flex text-sm text-brand-primary underline">Open the authenticated Workspace Console</a>}
    </div>
  </main>;
}
