import React, {useEffect, useMemo, useRef, useState} from 'react';
import {AlertTriangle, CheckCircle2, Copy, ExternalLink, Loader2, RefreshCw, ShieldAlert, ShieldCheck} from 'lucide-react';

import {NekiroApiError, toPlatformErrorView, type AgentRelease, type EndpointBinding, type NekiroApiClient, type VerificationChallenge} from '../api/nekiro';
import {agentKey, canReleaseAction, isCurrentRequest, nextRequestGeneration} from '../consolePolicy';
import type {Agent, PlatformErrorView} from '../types';

interface TrustedPublicationTabProps {
  providerId: string;
  client: NekiroApiClient;
  agents: Agent[];
  draftAgents: Agent[];
  providerCatalogError: PlatformErrorView | null;
  onRefresh: () => void;
}

type ReleaseAction = 'suspend' | 'revoke';

export default function TrustedPublicationTab({providerId, client, agents, draftAgents, providerCatalogError, onRefresh}: TrustedPublicationTabProps) {
  const availableAgents = useMemo(() => dedupeAgents([...draftAgents, ...agents].filter((agent) => agent.ownerId === providerId)), [agents, draftAgents, providerId]);
  const [selectedAgentKey, setSelectedAgentKey] = useState('');
  const [endpoint, setEndpoint] = useState('');
  const [bindingId, setBindingId] = useState('');
  const [binding, setBinding] = useState<EndpointBinding | null>(null);
  const [challenge, setChallenge] = useState<VerificationChallenge | null>(null);
  const [releaseId, setReleaseId] = useState('');
  const [release, setRelease] = useState<AgentRelease | null>(null);
  const [operation, setOperation] = useState<string | null>(null);
  const [error, setError] = useState<PlatformErrorView | null>(null);
  const [confirmAction, setConfirmAction] = useState<ReleaseAction | null>(null);
  const requestGeneration = useRef(0);

  const selectedAgent = availableAgents.find((agent) => agentKey(agent) === selectedAgentKey) ?? availableAgents[0];

  const selectAgent = (value: string) => {
    requestGeneration.current = nextRequestGeneration(requestGeneration.current);
    setOperation(null);
    setSelectedAgentKey(value);
    setBinding(null);
    setBindingId('');
    setChallenge(null);
    setRelease(null);
    setReleaseId('');
    setError(null);
  };

  useEffect(() => {
    if (selectedAgentKey && !availableAgents.some((agent) => agentKey(agent) === selectedAgentKey)) {
      requestGeneration.current = nextRequestGeneration(requestGeneration.current);
      setOperation(null);
      setSelectedAgentKey('');
      setBinding(null);
      setBindingId('');
      setChallenge(null);
      setRelease(null);
      setReleaseId('');
    }
  }, [availableAgents, selectedAgentKey]);

  const run = async (name: string, task: (generation: number) => Promise<void>) => {
    const generation = nextRequestGeneration(requestGeneration.current);
    requestGeneration.current = generation;
    setOperation(name);
    setError(null);
    try {
      await task(generation);
    } catch (value) {
      if (isCurrentRequest(generation, requestGeneration.current)) setError(toPlatformErrorView(value, 'Trusted Publication operation failed.'));
    } finally {
      if (isCurrentRequest(generation, requestGeneration.current)) setOperation(null);
    }
  };

  const createBinding = () => run('create-binding', async (generation) => {
    if (!selectedAgent) throw new Error('Select a registered Agent Card first.');
    const value = await client.createEndpointBinding(providerId, selectedAgent.id, {
      endpoint,
      method: 'http_well_known',
      version: selectedAgent.version,
    });
    if (!isCurrentRequest(generation, requestGeneration.current)) return;
    const authoritative = await client.getEndpointBinding(providerId, value.bindingId);
    if (!isCurrentRequest(generation, requestGeneration.current)) return;
    assertBindingMatches(authoritative, providerId, selectedAgent, value.bindingId);
    setBinding(authoritative);
    setBindingId(authoritative.bindingId);
    setChallenge(null);
    setRelease(null);
    setReleaseId('');
  });

  const readBinding = () => run('read-binding', async (generation) => {
    const value = await client.getEndpointBinding(providerId, bindingId);
    if (!isCurrentRequest(generation, requestGeneration.current)) return;
    assertBindingMatches(value, providerId, selectedAgent, bindingId);
    setBinding(value);
  });

  const issueChallenge = () => run('issue-challenge', async (generation) => {
    if (!binding) throw new Error('Read or create an Endpoint Binding first.');
    const value = await client.createVerificationChallenge(providerId, binding.bindingId);
    if (!isCurrentRequest(generation, requestGeneration.current)) return;
    setChallenge(value);
  });

  const completeChallenge = () => run('complete-challenge', async (generation) => {
    if (!challenge || !binding) throw new Error('Issue a fresh Challenge before completing verification.');
    try {
      await client.completeVerificationChallenge(providerId, binding.bindingId, challenge.challengeId);
    } finally {
      if (isCurrentRequest(generation, requestGeneration.current)) setChallenge(null);
    }
    if (!isCurrentRequest(generation, requestGeneration.current)) return;
    const value = await client.getEndpointBinding(providerId, binding.bindingId);
    if (!isCurrentRequest(generation, requestGeneration.current)) return;
    assertBindingMatches(value, providerId, selectedAgent, binding.bindingId);
    setBinding(value);
  });

  const createRelease = () => run('create-release', async (generation) => {
    if (!selectedAgent || !binding) throw new Error('Select an Agent and create its Endpoint Binding first.');
    if (binding.verificationStatus !== 'pending' && binding.verificationStatus !== 'verified') {
      throw new Error('The Endpoint Binding is not eligible for Release creation.');
    }
    const value = await client.createAgentRelease(providerId, selectedAgent.id, {
      version: selectedAgent.version,
      endpointBindingId: binding.bindingId,
    });
    if (!isCurrentRequest(generation, requestGeneration.current)) return;
    assertReleaseMatches(value, providerId, selectedAgent, binding.bindingId);
    const authoritative = await client.getAgentRelease(value.releaseId);
    if (!isCurrentRequest(generation, requestGeneration.current)) return;
    assertReleaseMatches(authoritative, providerId, selectedAgent, binding.bindingId);
    setRelease(authoritative);
    setReleaseId(authoritative.releaseId);
  });

  const readRelease = () => run('read-release', async (generation) => {
    const value = await client.getAgentRelease(releaseId);
    if (!isCurrentRequest(generation, requestGeneration.current)) return;
    assertReleaseMatches(value, providerId, selectedAgent, binding?.bindingId);
    setRelease(value);
  });

  const refreshRelease = () => run('refresh-release', async (generation) => {
    if (!release) throw new Error('Create or read a Release first.');
    const value = await client.getAgentRelease(release.releaseId);
    if (!isCurrentRequest(generation, requestGeneration.current)) return;
    assertReleaseMatches(value, providerId, selectedAgent, binding?.bindingId ?? value.endpointBindingId);
    setRelease(value);
  });

  const releaseAction = (action: 'verify' | 'publish' | ReleaseAction) => run(action + '-release', async (generation) => {
    if (!release) throw new Error('Create or read a Release first.');
    const value = action === 'verify'
      ? await client.verifyAgentRelease(release.releaseId)
      : action === 'publish'
        ? await client.publishAgentRelease(release.releaseId)
        : action === 'suspend'
          ? await client.suspendAgentRelease(release.releaseId)
          : await client.revokeAgentRelease(release.releaseId);
     if (!isCurrentRequest(generation, requestGeneration.current)) return;
     const authoritative = await client.getAgentRelease(value.releaseId);
     if (!isCurrentRequest(generation, requestGeneration.current)) return;
    assertReleaseMatches(authoritative, providerId, selectedAgent, binding?.bindingId ?? authoritative.endpointBindingId);
    setRelease(authoritative);
    setReleaseId(authoritative.releaseId);
    setConfirmAction(null);
  });

  const canIssueChallenge = binding?.verificationStatus === 'pending';
  const canCompleteChallenge = Boolean(challenge && binding?.verificationStatus === 'pending');
  const canCreateRelease = Boolean(binding && selectedAgent && (binding.verificationStatus === 'pending' || binding.verificationStatus === 'verified'));
  const canVerify = canReleaseAction(release?.state, 'verify');
  const canPublish = canReleaseAction(release?.state, 'publish');
  const canSuspend = canReleaseAction(release?.state, 'suspend');
  const canRevoke = canReleaseAction(release?.state, 'revoke');

  return (
    <div className="h-full flex flex-col gap-5">
      <div className="glass-page-header flex items-start justify-between gap-4">
        <div>
          <div className="font-mono-label text-[10px] uppercase tracking-[0.24em] text-brand-primary mb-2">Trust boundary</div>
          <h2 className="text-2xl font-bold text-brand-on-surface">Trusted Publication</h2>
          <p className="text-sm text-brand-on-surface-variant mt-1 max-w-3xl">Prove endpoint ownership, publish an immutable Release, and hand its exact identity to a Workspace owner.</p>
        </div>
        <button onClick={onRefresh} disabled={Boolean(operation)} className="px-3 py-2 rounded bg-brand-container border border-brand-outline-variant text-xs text-brand-on-surface-variant hover:text-brand-on-surface flex items-center gap-2 disabled:opacity-50">
          <RefreshCw size={14} /> Refresh Cards
        </button>
      </div>

      <ErrorBanner error={error} />
      <ErrorBanner error={providerCatalogError} />

      <div className="grid grid-cols-[minmax(280px,0.7fr)_minmax(520px,1.3fr)] gap-5 min-h-0 flex-1">
        <div className="bg-brand-low border border-brand-outline-variant rounded-xl overflow-hidden min-h-0 flex flex-col">
          <div className="px-4 py-3 border-b border-brand-outline-variant/60">
            <div className="text-xs font-bold text-brand-on-surface">Registered Agent Cards</div>
            <div className="text-[11px] text-brand-on-surface-variant mt-1">Catalog status does not grant trust.</div>
          </div>
          <div className="overflow-y-auto divide-y divide-brand-outline-variant/40">
            {availableAgents.length === 0 ? (
              <div className="p-6 text-center text-sm text-brand-on-surface-variant">Register an Agent Card before creating a Binding.</div>
            ) : availableAgents.map((agent) => (
              <button key={agentKey(agent)} onClick={() => selectAgent(agentKey(agent))} className={'w-full text-left p-4 hover:bg-brand-container ' + (selectedAgent && agentKey(selectedAgent) === agentKey(agent) ? 'bg-brand-primary-container/20' : '')}>
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="font-semibold text-brand-on-surface">{agent.name}</div>
                    <div className="font-mono-code text-[11px] text-brand-on-surface-variant mt-1">{agent.id} @ {agent.version}</div>
                  </div>
                  <span className="text-[10px] uppercase border border-brand-outline-variant rounded px-2 py-0.5 text-brand-on-surface-variant">{agent.status}</span>
                </div>
              </button>
            ))}
          </div>
        </div>

        <div className="min-h-0 overflow-y-auto space-y-5">
          <section className="bg-brand-low border border-brand-outline-variant rounded-xl p-4">
            <SectionTitle icon={<ShieldCheck size={16} />} title="1. Endpoint Binding" detail={selectedAgent ? `${selectedAgent.id} @ ${selectedAgent.version}` : 'Select an Agent Card'} />
            <div className="grid grid-cols-2 gap-3 mt-4">
              <Field label="Provider ID" value={providerId} readOnly />
              <Field label="Agent endpoint" value={endpoint} onChange={setEndpoint} placeholder="https://agent.example/a2a" />
            </div>
            <div className="flex flex-wrap gap-2 mt-4">
              <ActionButton label="Create Binding" onClick={createBinding} disabled={!selectedAgent || !endpoint || Boolean(operation)} />
              <ActionButton label="Read Binding" onClick={readBinding} disabled={!bindingId || Boolean(operation)} />
            </div>
            {binding && <BindingFacts binding={binding} />}
          </section>

          <section className="bg-brand-low border border-brand-outline-variant rounded-xl p-4">
            <SectionTitle icon={<ExternalLink size={16} />} title="2. Endpoint Challenge" detail="The proof is shown once and kept in component memory." />
            <div className="flex flex-wrap gap-2 mt-4">
              <ActionButton label="Issue Challenge" onClick={issueChallenge} disabled={!canIssueChallenge || Boolean(operation)} />
              <ActionButton label="Complete Verification" onClick={completeChallenge} disabled={!canCompleteChallenge || Boolean(operation)} />
            </div>
            {challenge && (
              <div className="mt-4 border border-brand-primary/30 bg-brand-primary-container/10 rounded-lg p-3 text-xs text-brand-on-surface-variant space-y-2">
                <div className="flex items-center gap-2 text-brand-primary font-semibold"><ShieldAlert size={14} /> Serve this exact proof at the challenge URL, then complete once.</div>
                <Fact label="Challenge ID" value={challenge.challengeId} />
                <Fact label="Challenge URL" value={challenge.challengeUrl} />
                <div className="flex items-center gap-2"><span className="font-mono-label text-[10px] uppercase text-brand-on-surface-variant">Proof</span><code className="font-mono-code text-brand-on-surface break-all">{challenge.proof}</code><Copy size={13} /></div>
                <Fact label="Expires" value={challenge.expiresAt} />
              </div>
            )}
          </section>

          <section className="bg-brand-low border border-brand-outline-variant rounded-xl p-4">
            <SectionTitle icon={<CheckCircle2 size={16} />} title="3. Immutable Release" detail="Every action is followed by a server read." />
            <div className="grid grid-cols-[1fr_auto] gap-3 mt-4">
              <Field label="Release ID handoff" value={releaseId} onChange={setReleaseId} placeholder="release-id" />
              <div className="flex items-end"><ActionButton label="Read" onClick={readRelease} disabled={!releaseId || Boolean(operation)} /></div>
            </div>
            <div className="flex flex-wrap gap-2 mt-4">
              <ActionButton label="Create Release" onClick={createRelease} disabled={!canCreateRelease || Boolean(operation)} />
              <ActionButton label="Verify" onClick={() => releaseAction('verify')} disabled={!canVerify || Boolean(operation)} />
              <ActionButton label="Publish" onClick={() => releaseAction('publish')} disabled={!canPublish || Boolean(operation)} />
              <ActionButton label="Refresh" onClick={refreshRelease} disabled={!release || Boolean(operation)} />
              <ActionButton label="Suspend" onClick={() => setConfirmAction('suspend')} disabled={!canSuspend || Boolean(operation)} danger />
              <ActionButton label="Revoke" onClick={() => setConfirmAction('revoke')} disabled={!canRevoke || Boolean(operation)} danger />
            </div>
            {operation && <div className="flex items-center gap-2 mt-3 text-xs text-brand-on-surface-variant"><Loader2 size={13} className="animate-spin" /> {operation} in progress</div>}
            {release && <ReleaseFacts release={release} />}
            {confirmAction && (
              <div className="mt-4 border border-brand-error/30 bg-brand-error-container/10 rounded-lg p-3 text-xs text-brand-on-surface-variant">
                <div className="font-semibold text-brand-error">Confirm {confirmAction} of {release?.releaseId}</div>
                <p className="mt-1">This is a server-owned lifecycle transition. Recovery does not republish or restore this Release in place.</p>
                <div className="flex gap-2 mt-3">
                  <ActionButton label={`Confirm ${confirmAction}`} onClick={() => releaseAction(confirmAction)} disabled={Boolean(operation)} danger />
                  <ActionButton label="Cancel" onClick={() => setConfirmAction(null)} disabled={Boolean(operation)} />
                </div>
              </div>
            )}
          </section>
        </div>
      </div>
    </div>
  );
}

function assertBindingMatches(value: EndpointBinding, providerId: string, agent: Agent | undefined, expectedBindingId: string): void {
  if (value.providerId !== providerId || value.bindingId !== expectedBindingId || (agent && (value.agentId !== agent.id || value.agentCardVersion !== agent.version))) {
    throw new NekiroApiError(200, 'Endpoint Binding does not match the selected provider, Agent, or Card version.', 'INVALID_RESPONSE');
  }
}

function assertReleaseMatches(value: AgentRelease, providerId: string, agent: Agent | undefined, expectedBindingId?: string): void {
  if (value.providerId !== providerId || (agent && (value.agentId !== agent.id || value.agentCardVersion !== agent.version)) || (expectedBindingId && value.endpointBindingId !== expectedBindingId)) {
    throw new NekiroApiError(200, 'Agent Release does not match the selected provider, Agent, or Binding.', 'INVALID_RESPONSE');
  }
}

function dedupeAgents(values: Agent[]): Agent[] {
  const seen = new Set<string>();
  return values.filter((agent) => {
    const key = agentKey(agent);
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function SectionTitle({icon, title, detail}: {icon: React.ReactNode; title: string; detail: string}) {
  return <div className="flex items-start gap-2"><span className="text-brand-primary mt-0.5">{icon}</span><div><div className="text-sm font-bold text-brand-on-surface">{title}</div><div className="text-xs text-brand-on-surface-variant mt-1">{detail}</div></div></div>;
}

function Field({label, value, onChange, placeholder, readOnly}: {label: string; value: string; onChange?: (value: string) => void; placeholder?: string; readOnly?: boolean}) {
  return <label className="flex flex-col gap-1 text-xs text-brand-on-surface-variant">{label}<input value={value} onChange={(event) => onChange?.(event.target.value)} placeholder={placeholder} readOnly={readOnly} className="bg-brand-lowest border border-brand-outline-variant rounded px-3 py-2 text-brand-on-surface outline-none read-only:opacity-70" /></label>;
}

function ActionButton({label, onClick, disabled, danger}: {label: string; onClick: () => void; disabled?: boolean; danger?: boolean}) {
  return <button type="button" onClick={onClick} disabled={disabled} className={'px-3 py-1.5 rounded border text-xs flex items-center gap-1.5 disabled:opacity-40 ' + (danger ? 'border-brand-error/30 text-brand-error bg-brand-error-container/10' : 'border-brand-outline-variant bg-brand-container text-brand-on-surface-variant hover:text-brand-on-surface')}>{label}</button>;
}

function BindingFacts({binding}: {binding: EndpointBinding}) {
  return <div className="grid grid-cols-3 gap-2 mt-4 text-xs"><Fact label="Binding" value={binding.bindingId} /><Fact label="Status" value={binding.verificationStatus} /><Fact label="Endpoint" value={binding.endpoint} /><Fact label="Method" value={binding.verificationMethod} /><Fact label="Evidence" value={binding.verificationEvidenceDigest ?? 'not available'} /><Fact label="Updated" value={binding.updatedAt} /></div>;
}

function ReleaseFacts({release}: {release: AgentRelease}) {
  return <div className="grid grid-cols-3 gap-2 mt-4 text-xs"><Fact label="Release" value={release.releaseId} /><Fact label="State" value={release.state} /><Fact label="Card digest" value={release.cardDigest} /><Fact label="Binding" value={release.endpointBindingId} /><Fact label="Origin" value={release.endpointOrigin} /><Fact label="Path" value={release.endpointPath} /><Fact label="Method" value={release.verificationMethod} /><Fact label="Evidence" value={release.verificationEvidenceDigest ?? 'not available'} /><Fact label="Updated" value={release.updatedAt} /></div>;
}

function Fact({label, value}: {label: string; value: string}) {
  return <div className="bg-brand-lowest border border-brand-outline-variant rounded p-2 min-w-0"><div className="text-[10px] uppercase tracking-wider text-brand-on-surface-variant">{label}</div><div className="font-mono-code text-[11px] text-brand-on-surface mt-1 break-all">{value}</div></div>;
}

function ErrorBanner({error}: {error: PlatformErrorView | null}) {
  if (!error) return null;
  return <div className="border border-brand-error/25 bg-brand-error-container/10 rounded-lg p-3 flex items-start gap-3 text-sm text-brand-error"><AlertTriangle size={16} className="mt-0.5" /><div><div className="font-semibold">{error.code ?? 'ERROR'} / HTTP {error.status}</div><div className="text-brand-on-surface-variant mt-1">{error.message}</div>{error.traceId && <div className="font-mono-code text-[11px] mt-1">traceId: {error.traceId}</div>}</div></div>;
}
