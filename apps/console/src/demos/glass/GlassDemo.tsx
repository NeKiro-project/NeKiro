import React, {useMemo, useState} from 'react';
import {AnimatePresence, motion} from 'motion/react';
import {
  BookOpen, Boxes, CheckCircle2, ChevronRight, Cpu, Database, Layers, LockKeyhole,
  PlayCircle, Rocket, Search, ShieldCheck, ShieldOff, Sparkles, Terminal, Zap,
} from 'lucide-react';

import {DemoSwitcher} from '../DemoRoot';
import {DEMO_AGENTS, DEMO_INSTALLATIONS, DEMO_WORKSPACE, matchesAgent, shortDate} from '../mockData';
import type {Agent, Installation} from '../../types';

type Tab = 'registry' | 'installations' | 'invocations' | 'ledger';

const NAV: {id: Tab; label: string; icon: React.ElementType; hint: string}[] = [
  {id: 'registry', label: 'Registry', icon: Database, hint: '6 cards'},
  {id: 'installations', label: 'Installations', icon: Cpu, hint: '4 pins'},
  {id: 'invocations', label: 'Invocations', icon: PlayCircle, hint: 'v4'},
  {id: 'ledger', label: 'Ledger', icon: BookOpen, hint: 'v4'},
];

export default function GlassDemo() {
  const [tab, setTab] = useState<Tab>('registry');
  const [query, setQuery] = useState('');
  const [selectedKey, setSelectedKey] = useState<string | null>(null);

  const agents = useMemo(() => DEMO_AGENTS.filter((a) => matchesAgent(a, query)), [query]);
  const selected = useMemo(
    () => agents.find((a) => a.id + '@' + a.version === selectedKey) ?? agents[0],
    [agents, selectedKey],
  );

  return (
    <div className="fixed inset-0 z-[150] bg-[#04060d] text-slate-100 overflow-hidden select-none">
      {/* ambient background */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="demo-blob w-[640px] h-[640px] -top-40 -left-40 opacity-25" style={{background: 'radial-gradient(circle, #4f46e5, transparent 70%)'}} />
        <div className="demo-blob w-[560px] h-[560px] top-1/2 -right-52 opacity-20" style={{background: 'radial-gradient(circle, #7c3aed, transparent 70%)', animationDelay: '-8s'}} />
        <div className="demo-blob w-[480px] h-[480px] -bottom-40 left-1/3 opacity-15" style={{background: 'radial-gradient(circle, #0ea5e9, transparent 70%)', animationDelay: '-16s'}} />
        <div className="absolute inset-0" style={{background: 'radial-gradient(120% 90% at 50% 50%, transparent 55%, rgba(2,4,10,0.8) 100%)'}} />
      </div>

      {/* sidebar */}
      <aside className="absolute left-0 top-0 bottom-0 w-64 border-r border-white/[0.07] bg-white/[0.02] backdrop-blur-2xl flex flex-col z-20">
        <div className="px-5 pt-6 pb-7">
          <div className="flex items-center gap-3">
            <div className="relative w-9 h-9 rounded-xl bg-gradient-to-br from-indigo-500 to-violet-600 flex items-center justify-center shadow-[0_0_24px_rgba(99,102,241,0.45)]">
              <Terminal size={17} className="text-white" />
            </div>
            <div>
              <div className="text-sm font-extrabold tracking-tight bg-gradient-to-r from-white to-slate-400 bg-clip-text text-transparent">NeKiro</div>
              <div className="text-[9.5px] uppercase tracking-[0.22em] text-slate-500">Infrastructure Orchestrator</div>
            </div>
          </div>
        </div>

        <nav className="px-3 space-y-1">
          {NAV.map((item) => {
            const Icon = item.icon;
            const active = tab === item.id;
            return (
              <button
                key={item.id}
                onClick={() => { setTab(item.id); setQuery(''); }}
                className={`group w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-[13px] transition-all duration-200 ${
                  active
                    ? 'bg-gradient-to-r from-indigo-500/20 to-violet-500/10 text-white border border-indigo-400/25 shadow-[inset_0_1px_0_rgba(255,255,255,0.06),0_0_20px_rgba(99,102,241,0.15)]'
                    : 'text-slate-400 hover:text-slate-100 hover:bg-white/[0.04] border border-transparent'
                }`}
              >
                <Icon size={16} className={active ? 'text-indigo-300' : 'text-slate-500 group-hover:text-slate-300'} />
                <span className="font-medium">{item.label}</span>
                <span className={`ml-auto text-[9.5px] font-jb px-1.5 py-0.5 rounded ${active ? 'bg-indigo-400/15 text-indigo-200' : 'text-slate-600'}`}>{item.hint}</span>
              </button>
            );
          })}
        </nav>

        <div className="mt-auto p-3">
          <div className="rounded-xl border border-white/[0.07] bg-white/[0.03] p-3.5">
            <div className="flex items-center gap-2 text-[10px] uppercase tracking-[0.18em] text-slate-500">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.9)]" />
              Workspace
            </div>
            <div className="font-jb text-[12.5px] text-slate-200 mt-1.5">{DEMO_WORKSPACE.workspaceId}</div>
            <div className="text-[10.5px] text-slate-500 mt-1">owner {DEMO_WORKSPACE.ownerId}</div>
          </div>
        </div>
      </aside>

      {/* header */}
      <header className="absolute left-64 right-0 top-0 h-16 border-b border-white/[0.06] bg-[#04060d]/60 backdrop-blur-xl flex items-center px-7 gap-4 z-10">
        <div className="relative flex items-center w-80 rounded-xl border border-white/[0.08] bg-white/[0.03] focus-within:border-indigo-400/40 focus-within:shadow-[0_0_0_3px_rgba(99,102,241,0.12)] transition-all">
          <Search size={14} className="ml-3 text-slate-500" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search agents, capabilities, owners…"
            className="w-full bg-transparent outline-none text-[12.5px] px-2.5 py-2 placeholder:text-slate-600"
          />
          <kbd className="mr-2.5 text-[9px] font-jb text-slate-600 border border-white/10 rounded px-1.5 py-0.5">⌘K</kbd>
        </div>
        <div className="ml-auto flex items-center gap-3">
          <span className="flex items-center gap-1.5 text-[10.5px] font-jb text-emerald-300 border border-emerald-400/20 bg-emerald-400/[0.07] rounded-full px-2.5 py-1">
            <Zap size={11} /> Northbound /v3
          </span>
          <div className="w-8 h-8 rounded-full bg-gradient-to-br from-indigo-400 to-violet-500 flex items-center justify-center text-[11px] font-bold text-white shadow-[0_0_16px_rgba(99,102,241,0.4)]">PT</div>
        </div>
      </header>

      {/* main */}
      <main className="absolute left-64 right-0 top-16 bottom-0 overflow-y-auto p-7 z-10">
        <AnimatePresence mode="wait" initial={false}>
          <motion.div key={tab} initial={{opacity: 0, y: 8}} animate={{opacity: 1, y: 0}} exit={{opacity: 0, y: -8}} transition={{duration: 0.18}} className="h-full">
            {tab === 'registry' && <Registry agents={agents} selected={selected} onSelect={(a) => setSelectedKey(a.id + '@' + a.version)} />}
            {tab === 'installations' && <Installations query={query} />}
            {(tab === 'invocations' || tab === 'ledger') && <Gated kind={tab} />}
          </motion.div>
        </AnimatePresence>
      </main>

      <DemoSwitcher current="glass" />
    </div>
  );
}

/* ---------------- Registry ---------------- */

function Registry({agents, selected, onSelect}: {agents: Agent[]; selected?: Agent; onSelect: (a: Agent) => void}) {
  const published = DEMO_AGENTS.filter((a) => a.status === 'published').length;
  const drafts = DEMO_AGENTS.filter((a) => a.status === 'draft').length;
  const installed = DEMO_INSTALLATIONS.filter((i) => i.status === 'enabled').length;

  return (
    <div className="flex flex-col gap-6 h-full min-h-0">
      <div className="flex items-end justify-between">
        <div>
          <div className="flex items-center gap-2 text-[10px] uppercase tracking-[0.28em] text-indigo-300/90 mb-2.5">
            <Sparkles size={11} /> Registry
          </div>
          <h2 className="text-[28px] leading-tight font-extrabold tracking-tight bg-gradient-to-r from-white via-slate-200 to-slate-500 bg-clip-text text-transparent">
            Agent Card Catalog
          </h2>
        </div>
        <button className="px-4 py-2.5 rounded-xl bg-gradient-to-r from-indigo-500 to-violet-500 text-white text-xs font-semibold shadow-[0_4px_24px_rgba(99,102,241,0.4)] hover:shadow-[0_4px_32px_rgba(99,102,241,0.6)] hover:brightness-110 transition-all flex items-center gap-2">
          <Rocket size={14} /> Register Agent Card
        </button>
      </div>

      {/* stats row */}
      <div className="grid grid-cols-4 gap-4">
        <Stat icon={<Boxes size={15} />} label="Total cards" value={String(DEMO_AGENTS.length)} delta="+2 this week" />
        <Stat icon={<CheckCircle2 size={15} />} label="Published" value={String(published)} delta="live in catalog" accent="emerald" />
        <Stat icon={<Layers size={15} />} label="Drafts" value={String(drafts)} delta="awaiting publish" accent="amber" />
        <Stat icon={<ShieldCheck size={15} />} label="Enabled installs" value={String(installed)} delta={'in ' + DEMO_WORKSPACE.workspaceId} accent="violet" />
      </div>

      <div className="grid grid-cols-[minmax(340px,0.9fr)_minmax(460px,1.1fr)] gap-5 flex-1 min-h-0">
        {/* list */}
        <div className="rounded-2xl border border-white/[0.07] bg-white/[0.025] backdrop-blur-xl overflow-hidden flex flex-col min-h-0">
          <div className="px-5 py-3.5 border-b border-white/[0.06] text-[10px] uppercase tracking-[0.2em] text-slate-500 flex justify-between">
            <span>Catalog entries</span>
            <span className="font-jb text-slate-400">{agents.length}</span>
          </div>
          <div className="overflow-y-auto p-2.5 space-y-2">
            {agents.map((agent) => {
              const active = selected === agent;
              return (
                <button
                  key={agent.id + agent.version}
                  onClick={() => onSelect(agent)}
                  className={`w-full text-left rounded-xl border p-4 transition-all duration-200 group ${
                    active
                      ? 'border-indigo-400/30 bg-gradient-to-br from-indigo-500/[0.12] to-violet-500/[0.06] shadow-[0_0_28px_rgba(99,102,241,0.12)]'
                      : 'border-white/[0.05] bg-white/[0.02] hover:bg-white/[0.05] hover:border-white/[0.12] hover:-translate-y-px'
                  }`}
                >
                  <div className="flex items-center justify-between gap-3">
                    <div className="font-semibold text-[14px] text-slate-100">{agent.name}</div>
                    <StatusPill status={agent.status} />
                  </div>
                  <div className="font-jb text-[11px] text-slate-500 mt-1">{agent.id} <span className="text-slate-700">@</span> {agent.version}</div>
                  <p className="text-[12px] text-slate-400 mt-2 leading-relaxed line-clamp-2">{agent.description}</p>
                  <div className="flex items-center gap-1.5 mt-3">
                    {agent.tags.map((t) => (
                      <span key={t} className="font-jb text-[9.5px] px-2 py-0.5 rounded-md bg-white/[0.05] border border-white/[0.07] text-slate-400">{t}</span>
                    ))}
                    <ChevronRight size={13} className={`ml-auto transition-all ${active ? 'text-indigo-300 translate-x-0 opacity-100' : 'text-slate-600 -translate-x-1 opacity-0 group-hover:opacity-100 group-hover:translate-x-0'}`} />
                  </div>
                </button>
              );
            })}
            {agents.length === 0 && <div className="p-10 text-center text-sm text-slate-500">No entries match the current filter.</div>}
          </div>
        </div>

        {/* detail */}
        <div className="rounded-2xl border border-white/[0.07] bg-white/[0.025] backdrop-blur-xl overflow-hidden flex flex-col min-h-0">
          {selected ? (
            <>
              <div className="px-6 py-5 border-b border-white/[0.06] flex items-start justify-between gap-4">
                <div>
                  <div className="text-[19px] font-bold tracking-tight">{selected.name}</div>
                  <div className="text-[12px] text-slate-500 mt-1">by {selected.owner} · <span className="font-jb">{selected.ownerId}</span></div>
                </div>
                <div className="flex gap-2">
                  {selected.status === 'draft' && <DetailAction icon={<Rocket size={13} />} label="Publish" primary />}
                  {selected.status === 'published' && <DetailAction icon={<Database size={13} />} label="Install" primary />}
                  {selected.status === 'published' && <DetailAction icon={<ShieldOff size={13} />} label="Disable" />}
                </div>
              </div>
              <div className="grid grid-cols-3 gap-3 px-6 py-4 border-b border-white/[0.06]">
                <Fact label="Version" value={selected.version} />
                <Fact label="Registered" value={shortDate(selected.registeredAt)} />
                <Fact label="Published" value={selected.publishedAt ? shortDate(selected.publishedAt) : 'not yet'} />
              </div>
              <div className="px-6 py-4 border-b border-white/[0.06]">
                <div className="text-[10px] uppercase tracking-[0.2em] text-slate-500 mb-2.5">Declared permissions</div>
                {selected.permissions.length === 0 ? (
                  <div className="text-[12px] text-slate-500">No permissions declared — installs submit <span className="font-jb text-slate-300">acceptedPermissions: []</span>.</div>
                ) : (
                  <div className="flex flex-wrap gap-2">
                    {selected.permissions.map((p) => (
                      <div key={p.id} className="rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2">
                        <div className="font-jb text-[11px] text-indigo-300">{p.id}</div>
                        <div className="text-[10.5px] text-slate-500 mt-0.5 max-w-56">{p.description}</div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
              <div className="flex-1 min-h-0 px-6 py-4 flex flex-col">
                <div className="text-[10px] uppercase tracking-[0.2em] text-slate-500 mb-2.5">Agent Card · v0.2</div>
                <pre className="flex-1 overflow-auto rounded-xl border border-white/[0.06] bg-black/30 p-4 font-jb text-[11px] leading-relaxed text-slate-400">{selected.schema}</pre>
              </div>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-sm text-slate-500">Select an entry to inspect its Agent Card.</div>
          )}
        </div>
      </div>
    </div>
  );
}

/* ---------------- Installations ---------------- */

function Installations({query}: {query: string}) {
  const items = DEMO_INSTALLATIONS.filter((i) => !query.trim() || [i.installationId, i.agentId, i.installedVersion, i.status].join(' ').toLowerCase().includes(query.toLowerCase()));
  return (
    <div className="flex flex-col gap-6 h-full min-h-0">
      <div>
        <div className="flex items-center gap-2 text-[10px] uppercase tracking-[0.28em] text-indigo-300/90 mb-2.5"><Cpu size={11} /> Installations</div>
        <h2 className="text-[28px] leading-tight font-extrabold tracking-tight bg-gradient-to-r from-white via-slate-200 to-slate-500 bg-clip-text text-transparent">Workspace Agent Pins</h2>
        <p className="text-[13px] text-slate-400 mt-2">Exact version pins inside <span className="font-jb text-slate-200">{DEMO_WORKSPACE.workspaceId}</span> with explicit acceptedPermissions.</p>
      </div>

      <div className="rounded-2xl border border-white/[0.07] bg-white/[0.025] backdrop-blur-xl overflow-hidden flex-1 min-h-0 flex flex-col">
        <div className="px-5 py-3.5 border-b border-white/[0.06] text-[10px] uppercase tracking-[0.2em] text-slate-500">Current & historical pins</div>
        <div className="overflow-y-auto divide-y divide-white/[0.05]">
          {items.map((ins) => (
            <div key={ins.installationId} className="px-6 py-5 hover:bg-white/[0.03] transition-colors">
              <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-4">
                  <div className={`w-9 h-9 rounded-xl flex items-center justify-center border ${
                    ins.status === 'enabled' ? 'bg-emerald-400/10 border-emerald-400/25 text-emerald-300'
                    : ins.status === 'disabled' ? 'bg-amber-400/10 border-amber-400/25 text-amber-300'
                    : 'bg-white/[0.04] border-white/10 text-slate-500'
                  }`}>
                    <Cpu size={16} />
                  </div>
                  <div>
                    <div className="font-semibold text-[14.5px]">{ins.agentId}</div>
                    <div className="font-jb text-[10.5px] text-slate-500 mt-0.5">{ins.installationId}</div>
                  </div>
                </div>
                <InstallPill status={ins.status} />
              </div>
              <div className="grid grid-cols-4 gap-3 mt-4">
                <Fact label="Constraint" value={ins.versionConstraint} />
                <Fact label="Pinned" value={ins.installedVersion} />
                <Fact label="Updated" value={shortDate(ins.updatedAt)} />
                <Fact label="Permissions" value={ins.acceptedPermissions.length === 0 ? '[]' : String(ins.acceptedPermissions.length) + ' accepted'} />
              </div>
            </div>
          ))}
          {items.length === 0 && <div className="p-10 text-center text-sm text-slate-500">No installations match the filter.</div>}
        </div>
      </div>
    </div>
  );
}

/* ---------------- Gated ---------------- */

function Gated({kind}: {kind: 'invocations' | 'ledger'}) {
  const isInv = kind === 'invocations';
  return (
    <div className="h-full flex items-center justify-center">
      <div className="max-w-md text-center rounded-3xl border border-white/[0.07] bg-white/[0.025] backdrop-blur-xl p-10">
        <div className="w-14 h-14 mx-auto rounded-2xl bg-indigo-500/10 border border-indigo-400/25 flex items-center justify-center text-indigo-300 shadow-[0_0_30px_rgba(99,102,241,0.2)]">
          <LockKeyhole size={24} />
        </div>
        <h3 className="text-lg font-bold mt-5">{isInv ? 'Invoke Runtime Gated' : 'Record Runtime Gated'}</h3>
        <p className="text-[13px] text-slate-400 mt-2 leading-relaxed">
          {isInv
            ? 'Invocation Dispatch and the A2A Router are waiting on the backend headless Invoke → Record slice. No mock executions are rendered.'
            : 'The Ledger surface is contract-aware only. No simulated traces or fabricated timelines are shown as platform facts.'}
        </p>
        <div className="mt-6 font-jb text-[10.5px] text-indigo-300/80 border border-indigo-400/20 bg-indigo-400/[0.06] rounded-lg px-3 py-2 inline-block">
          {isInv ? 'POST /v4/workspaces/{id}/invocations' : 'GET /v4/workspaces/{id}/traces/{traceId}'}
        </div>
      </div>
    </div>
  );
}

/* ---------------- bits ---------------- */

function Stat({icon, label, value, delta, accent}: {icon: React.ReactNode; label: string; value: string; delta: string; accent?: 'emerald' | 'amber' | 'violet'}) {
  const color = accent === 'emerald' ? 'text-emerald-300' : accent === 'amber' ? 'text-amber-300' : accent === 'violet' ? 'text-violet-300' : 'text-indigo-300';
  return (
    <div className="rounded-2xl border border-white/[0.07] bg-white/[0.025] backdrop-blur-xl px-5 py-4 hover:border-white/[0.14] hover:-translate-y-px transition-all">
      <div className={`flex items-center gap-2 text-[10px] uppercase tracking-[0.18em] text-slate-500`}>
        <span className={color}>{icon}</span>{label}
      </div>
      <div className="text-[26px] font-extrabold tracking-tight mt-2">{value}</div>
      <div className="text-[10.5px] text-slate-500 mt-0.5">{delta}</div>
    </div>
  );
}

function Fact({label, value}: {label: string; value: string}) {
  return (
    <div className="rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-2 min-w-0">
      <div className="text-[9px] uppercase tracking-[0.16em] text-slate-600">{label}</div>
      <div className="font-jb text-[11.5px] text-slate-200 mt-1 truncate">{value}</div>
    </div>
  );
}

function StatusPill({status}: {status: Agent['status']}) {
  const map = {
    published: 'text-emerald-300 border-emerald-400/25 bg-emerald-400/10',
    draft: 'text-amber-300 border-amber-400/25 bg-amber-400/10',
    disabled: 'text-slate-400 border-white/10 bg-white/[0.04]',
  } as const;
  const dot = {published: 'bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.9)]', draft: 'bg-amber-400', disabled: 'bg-slate-500'} as const;
  return (
    <span className={`flex items-center gap-1.5 text-[9.5px] uppercase tracking-wider font-semibold border rounded-full px-2.5 py-1 ${map[status]}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${dot[status]}`} />{status}
    </span>
  );
}

function InstallPill({status}: {status: Installation['status']}) {
  const map = {
    enabled: 'text-emerald-300 border-emerald-400/25 bg-emerald-400/10',
    disabled: 'text-amber-300 border-amber-400/25 bg-amber-400/10',
    uninstalled: 'text-slate-400 border-white/10 bg-white/[0.04]',
  } as const;
  return <span className={`text-[9.5px] uppercase tracking-wider font-semibold border rounded-full px-2.5 py-1 ${map[status]}`}>{status}</span>;
}

function DetailAction({icon, label, primary}: {icon: React.ReactNode; label: string; primary?: boolean}) {
  return (
    <button className={`px-3.5 py-2 rounded-xl text-[11.5px] font-semibold flex items-center gap-1.5 transition-all ${
      primary
        ? 'bg-gradient-to-r from-indigo-500 to-violet-500 text-white shadow-[0_2px_16px_rgba(99,102,241,0.35)] hover:brightness-110'
        : 'border border-white/10 bg-white/[0.03] text-slate-300 hover:bg-white/[0.07]'
    }`}>
      {icon}{label}
    </button>
  );
}
