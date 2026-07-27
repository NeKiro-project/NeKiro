import {useEffect, useMemo, useRef, useState} from 'react';
import {Activity, CheckCircle2, LoaderCircle, Play, Radio, ShieldAlert} from 'lucide-react';

import {NekiroApiClient, toPlatformErrorView, type InvocationResultStreamEventV2} from '../api/nekiro';
import {isCurrentRequest, isTrustedEnabledInstallation, nextRequestGeneration} from '../consolePolicy';
import type {Installation, PlatformErrorView, Workspace} from '../types';

interface InvocationsTabProps {
  workspace: Workspace | null;
  installations: Installation[];
  client: NekiroApiClient;
}

export default function InvocationsTab({workspace, installations, client}: InvocationsTabProps) {
  const enabled = useMemo(() => installations.filter(isTrustedEnabledInstallation), [installations]);
  const [installationId, setInstallationId] = useState('');
  const [capability, setCapability] = useState('');
  const [input, setInput] = useState('{\n  "message": "hello"\n}');
  const [stream, setStream] = useState(false);
  const [events, setEvents] = useState<InvocationResultStreamEventV2[]>([]);
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<PlatformErrorView | null>(null);
  const [loading, setLoading] = useState(false);
  const requestGeneration = useRef(0);

  useEffect(() => {
    requestGeneration.current = nextRequestGeneration(requestGeneration.current);
    setLoading(false);
    setResult(null);
    setEvents([]);
    setError(null);
    if (installationId && !enabled.some((item) => item.installationId === installationId)) {
      setInstallationId('');
      setCapability('');
    }
  }, [workspace?.workspaceId]);

  useEffect(() => {
    if (installationId && !enabled.some((item) => item.installationId === installationId)) {
      setInstallationId('');
      setCapability('');
    }
  }, [enabled, installationId]);

  const run = async () => {
    if (!workspace) {
      setError({status: 0, code: 'CONFIGURATION_ERROR', message: 'Select the active Workspace first.'});
      return;
    }
    const installation = enabled.find((item) => item.installationId === installationId);
    if (!installation) {
      setError({status: 0, code: 'INSTALLATION_DISABLED', message: 'Select an enabled trusted Installation before invoking.'});
      return;
    }
    let parsed: unknown;
    try {
      parsed = JSON.parse(input);
    } catch {
      setError({status: 0, code: 'VALIDATION_ERROR', message: 'Input must be valid JSON.'});
      return;
    }
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
      setError({status: 0, code: 'VALIDATION_ERROR', message: 'Input must be a JSON object.'});
      return;
    }
    const generation = nextRequestGeneration(requestGeneration.current);
    requestGeneration.current = generation;
    const workspaceId = workspace.workspaceId;
    setLoading(true);
    setError(null);
    setResult(null);
    setEvents([]);
    try {
      if (stream) {
        await client.invokeStream(workspaceId, {agentId: installation.agentId, capability, input: parsed as Record<string, unknown>}, (event) => {
          if (isCurrentRequest(generation, requestGeneration.current)) setEvents((current) => [...current, event]);
        });
      } else {
        const value = await client.invoke(workspaceId, {agentId: installation.agentId, capability, input: parsed as Record<string, unknown>, stream: false});
        if (isCurrentRequest(generation, requestGeneration.current)) setResult(value);
      }
    } catch (value) {
      if (isCurrentRequest(generation, requestGeneration.current)) setError(toPlatformErrorView(value, 'Invocation failed.'));
    } finally {
      if (isCurrentRequest(generation, requestGeneration.current)) setLoading(false);
    }
  };

  return (
    <div className="h-full flex flex-col gap-5">
      <div>
        <div className="font-mono-label text-[10px] uppercase tracking-[0.24em] text-brand-primary mb-2">Invocations / Owner</div>
        <h2 className="text-2xl font-bold text-brand-on-surface">Invoke an installed Agent</h2>
        <p className="text-sm text-brand-on-surface-variant mt-1 max-w-3xl">Requests use Gateway v4. JSON and SSE responses are validated for correlation and terminal semantics before display.</p>
      </div>
      <div className="glass-split-grid grid grid-cols-[minmax(340px,0.85fr)_minmax(420px,1.15fr)] gap-5 flex-1 min-h-0">
        <DispatchForm
          workspace={workspace}
          enabled={enabled}
          installationId={installationId}
          setInstallationId={(value) => { setInstallationId(value); setCapability(''); }}
          capability={capability}
          setCapability={setCapability}
          input={input}
          setInput={setInput}
          stream={stream}
          setStream={setStream}
          loading={loading}
          onSubmit={() => void run()}
        />
        <ResponsePanel stream={stream} loading={loading} result={result} events={events} error={error} />
      </div>
    </div>
  );
}

function DispatchForm({workspace, enabled, installationId, setInstallationId, capability, setCapability, input, setInput, stream, setStream, loading, onSubmit}: {
  workspace: Workspace | null;
  enabled: Installation[];
  installationId: string;
  setInstallationId: (value: string) => void;
  capability: string;
  setCapability: (value: string) => void;
  input: string;
  setInput: (value: string) => void;
  stream: boolean;
  setStream: (value: boolean) => void;
  loading: boolean;
  onSubmit: () => void;
}) {
  return (
    <section className="bg-brand-low border border-brand-outline-variant rounded-xl p-5 h-fit">
      <div className="flex items-center gap-2 text-sm font-bold mb-4"><Play size={16} className="text-brand-primary" /> Dispatch request</div>
      <label htmlFor="invocation-installation" className="block text-xs text-brand-on-surface-variant mb-1">Installed Agent</label>
      <select id="invocation-installation" value={installationId} onChange={(event) => setInstallationId(event.target.value)} disabled={loading} className="w-full rounded-lg border border-brand-outline-variant bg-brand-lowest px-3 py-2 text-sm text-brand-on-surface outline-none disabled:opacity-40">
        <option value="">Select enabled installation</option>
        {enabled.map((item) => <option key={item.installationId} value={item.installationId}>{item.agentId} @ {item.installedVersion} / {item.installedReleaseId}</option>)}
      </select>
      <label htmlFor="invocation-capability" className="block text-xs text-brand-on-surface-variant mt-4 mb-1">Capability</label>
      <input id="invocation-capability" value={capability} onChange={(event) => setCapability(event.target.value)} disabled={loading} placeholder="Enter declared capability" className="w-full rounded-lg border border-brand-outline-variant bg-brand-lowest px-3 py-2 text-sm text-brand-on-surface outline-none disabled:opacity-40" />
      <label htmlFor="invocation-input" className="block text-xs text-brand-on-surface-variant mt-4 mb-1">Input JSON</label>
      <textarea id="invocation-input" value={input} onChange={(event) => setInput(event.target.value)} disabled={loading} rows={7} className="w-full rounded-lg border border-brand-outline-variant bg-brand-lowest px-3 py-2 font-mono-code text-xs text-brand-on-surface outline-none resize-y disabled:opacity-40" />
      <label htmlFor="invocation-stream" className="flex items-center gap-2 mt-3 text-xs text-brand-on-surface-variant"><input id="invocation-stream" type="checkbox" checked={stream} onChange={(event) => setStream(event.target.checked)} disabled={loading} /> Stream result over SSE</label>
      <button disabled={loading || !workspace || !installationId || !capability} onClick={onSubmit} className="mt-4 w-full rounded-lg bg-brand-primary px-4 py-2.5 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:opacity-40">{loading ? 'Invoking...' : 'Invoke'}</button>
      {!workspace && <p className="mt-3 text-xs text-amber-300">Select the active Workspace before invoking.</p>}
    </section>
  );
}

function ResponsePanel({stream, loading, result, events, error}: {stream: boolean; loading: boolean; result: unknown; events: InvocationResultStreamEventV2[]; error: PlatformErrorView | null}) {
  return (
    <section className="bg-brand-low border border-brand-outline-variant rounded-xl overflow-hidden min-h-0 flex flex-col">
      <div className="px-4 py-3 border-b border-brand-outline-variant/60 flex items-center gap-2"><Activity size={15} className="text-brand-primary" /><span className="text-xs font-bold">Gateway response</span>{stream && <span className="ml-auto flex items-center gap-1 text-[10px] text-amber-300"><Radio size={12} /> SSE</span>}</div>
      <div className="p-4 overflow-auto flex-1">
        {error && <div className="mb-4 rounded-lg border border-red-400/25 bg-red-400/10 p-3 text-xs text-red-200"><ShieldAlert size={14} className="inline mr-2" />{error.message}{error.traceId && <span className="block mt-1 font-mono-code text-[10px]">trace {error.traceId}</span>}{error.invocationId && <span className="block mt-1 font-mono-code text-[10px]">invocation {error.invocationId}</span>}{error.rootTaskId && <span className="block mt-1 font-mono-code text-[10px]">root task {error.rootTaskId}</span>}</div>}
        {loading && <div className="flex items-center gap-2 text-xs text-brand-on-surface-variant"><LoaderCircle size={14} className="animate-spin" /> Waiting for Router...</div>}
        {result !== null && <JsonBlock value={result} />}
        {events.length > 0 && <div className="space-y-2">{events.map((event) => <div key={event.sequence} className="rounded-lg border border-brand-outline-variant bg-brand-lowest p-3 text-xs"><div className="flex items-center gap-2"><CheckCircle2 size={13} className={event.type === 'completed' ? 'text-emerald-300' : 'text-brand-primary'} /><span className="font-mono-code text-brand-primary">#{event.sequence} {event.type}</span><span className="ml-auto text-brand-on-surface-variant">{event.status}</span></div>{event.type === 'chunk' && <pre className="mt-2 whitespace-pre-wrap font-mono-code text-[11px] text-brand-on-surface-variant">{JSON.stringify(event.chunk, null, 2)}</pre>}{event.error && <div className="mt-2 text-red-200">{event.error.code}: {event.error.message}</div>}</div>)}</div>}
        {!loading && !result && events.length === 0 && !error && <div className="h-full flex items-center justify-center text-sm text-brand-on-surface-variant">No invocation submitted.</div>}
      </div>
    </section>
  );
}

function JsonBlock({value}: {value: unknown}) {
  return <pre className="rounded-lg border border-brand-outline-variant bg-brand-lowest p-4 whitespace-pre-wrap font-mono-code text-xs text-brand-on-surface-variant">{JSON.stringify(value, null, 2)}</pre>;
}
