import {useMemo, useState} from 'react';
import {Activity, ArrowUpRight, Database, Filter, GitBranch, Play, Search, TerminalSquare} from 'lucide-react';

import {DemoSwitcher} from '../DemoRoot';
import {DEMO_AGENTS, DEMO_INSTALLATIONS, DEMO_WORKSPACE, matchesAgent, shortDate} from '../mockData';

type Tab = 'registry' | 'installations' | 'invocations' | 'ledger';

export default function TerminalDemo() {
  const [tab, setTab] = useState<Tab>('registry');
  const [query, setQuery] = useState('');
  const agents = useMemo(() => DEMO_AGENTS.filter((agent) => matchesAgent(agent, query)), [query]);

  return (
    <div className="fixed inset-0 z-[150] overflow-hidden bg-[#0b0c0d] text-[#e7e5df] font-jb selection:bg-amber-300 selection:text-black">
      <div className="demo-scanlines pointer-events-none absolute inset-0 opacity-40" />
      <header className="h-14 border-b border-[#3b3830] bg-[#111211] flex items-center px-5 gap-6">
        <div className="flex items-center gap-2 text-amber-300"><TerminalSquare size={17} /><span className="font-bold tracking-[0.16em] text-sm">NEKIRO//OPS</span></div>
        <div className="text-[10px] text-[#7f7b70]">CONTROL PLANE / CATALOG</div>
        <div className="ml-auto flex items-center gap-4 text-[10px] text-[#aaa59a]"><span>WS {DEMO_WORKSPACE.workspaceId}</span><span className="text-emerald-400">● OWNER ONLINE</span></div>
      </header>
      <div className="flex h-[calc(100%-56px)]">
        <aside className="w-56 border-r border-[#3b3830] bg-[#0f100f] p-3">
          <div className="text-[10px] text-[#6f6b62] tracking-[0.2em] mb-3">NAVIGATION</div>
          {([
            ['registry', 'REGISTRY', Database], ['installations', 'INSTALLATIONS', GitBranch], ['invocations', 'INVOCATIONS', Play], ['ledger', 'LEDGER', Activity],
          ] as const).map(([id, label, Icon]) => (
            <button key={id} onClick={() => setTab(id)} className={`w-full flex items-center gap-2 px-3 py-2 mb-1 text-left text-[11px] border-l-2 ${tab === id ? 'border-amber-300 bg-amber-300/10 text-amber-200' : 'border-transparent text-[#8e897e] hover:text-[#ddd8cc]'}`}>
              <Icon size={13} /> {label}<span className="ml-auto text-[9px] text-[#5e5a51]">{id === 'registry' ? '06' : id === 'installations' ? '04' : '—'}</span>
            </button>
          ))}
          <div className="mt-8 border-t border-[#2e2c27] pt-4 text-[10px] text-[#6f6b62] leading-relaxed">STRICT MODE<br /><span className="text-emerald-400">NO FABRICATED EVENTS</span><br />NORTHBOUND API v4</div>
        </aside>
        <main className="flex-1 overflow-y-auto p-6">
          <div className="flex items-start justify-between mb-6">
            <div><div className="text-[10px] tracking-[0.25em] text-amber-300 mb-2">{tab.toUpperCase()} / LIVE VIEW</div><h1 className="text-2xl font-bold tracking-tight">{tab === 'registry' ? 'Agent Card Catalog' : tab === 'installations' ? 'Workspace Pins' : tab === 'invocations' ? 'Invocation Dispatch' : 'Invocation Ledger'}</h1></div>
            <div className="text-right text-[10px] text-[#777268]">OWNER: {DEMO_WORKSPACE.ownerId}<br />UPDATED {shortDate(DEMO_WORKSPACE.updatedAt)}</div>
          </div>
          {tab === 'registry' && <Registry agents={agents} query={query} setQuery={setQuery} />}
          {tab === 'installations' && <Installations query={query} setQuery={setQuery} />}
          {tab === 'invocations' && <RuntimePanel kind="invocations" />}
          {tab === 'ledger' && <RuntimePanel kind="ledger" />}
        </main>
      </div>
      <DemoSwitcher current="terminal" />
    </div>
  );
}

function SearchBox({query, setQuery}: {query: string; setQuery: (value: string) => void}) {
  return <div className="flex items-center border border-[#49443a] bg-[#121311] px-2 py-1.5 mb-4 max-w-lg"><Search size={13} className="text-[#777268]" /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="filter by id, capability, owner" className="ml-2 flex-1 bg-transparent outline-none text-[11px] placeholder:text-[#5e5a51]" /><Filter size={12} className="text-[#777268]" /></div>;
}

function Registry({agents, query, setQuery}: {agents: typeof DEMO_AGENTS; query: string; setQuery: (value: string) => void}) {
  return <><SearchBox query={query} setQuery={setQuery} /><div className="border border-[#3b3830] bg-[#0f100f]"><div className="grid grid-cols-[1.6fr_1fr_1fr_0.7fr_0.8fr] border-b border-[#3b3830] px-4 py-2 text-[9px] tracking-[0.18em] text-[#777268]"><span>AGENT CARD</span><span>CAPABILITY</span><span>OWNER</span><span>VERSION</span><span>STATE</span></div>{agents.map((agent) => <div key={agent.id + agent.version} className="grid grid-cols-[1.6fr_1fr_1fr_0.7fr_0.8fr] items-center px-4 py-3 border-b border-[#292823] text-[11px] hover:bg-amber-300/[0.04]"><div><div className="text-[#eee9dc]">{agent.name}</div><div className="text-[9px] text-[#777268]">{agent.id}</div></div><span className="text-[#b2ada1]">{agent.tags.join(', ')}</span><span className="text-[#b2ada1]">{agent.ownerId}</span><span className="text-amber-200">{agent.version}</span><span className={agent.status === 'published' ? 'text-emerald-400' : agent.status === 'draft' ? 'text-amber-300' : 'text-[#777268]'}>{agent.status.toUpperCase()}</span></div>)}{agents.length === 0 && <div className="p-8 text-center text-[11px] text-[#777268]">NO MATCHING CARDS</div>}</div></>;
}

function Installations({query, setQuery}: {query: string; setQuery: (value: string) => void}) {
  const items = DEMO_INSTALLATIONS.filter((item) => !query || [item.installationId, item.agentId, item.installedVersion, item.status].join(' ').toLowerCase().includes(query.toLowerCase()));
  return <><SearchBox query={query} setQuery={setQuery} /><div className="border border-[#3b3830] bg-[#0f100f]">{items.map((item) => <div key={item.installationId} className="grid grid-cols-[1.4fr_0.8fr_0.8fr_0.8fr] items-center px-4 py-4 border-b border-[#292823] text-[11px]"><div><div>{item.agentId}</div><div className="text-[9px] text-[#777268]">{item.installationId}</div></div><span>constraint {item.versionConstraint}</span><span className="text-amber-200">pin {item.installedVersion}</span><span className={item.status === 'enabled' ? 'text-emerald-400' : 'text-[#aaa59a]'}>{item.status.toUpperCase()}</span></div>)}</div></>;
}

function RuntimePanel({kind}: {kind: 'invocations' | 'ledger'}) {
  return <div className="max-w-3xl border border-[#3b3830] bg-[#0f100f] p-6"><div className="flex items-center gap-3 text-amber-200"><ArrowUpRight size={16} /><span className="text-sm">{kind === 'invocations' ? 'POST /v4/workspaces/{workspaceId}/invocations' : 'GET /v4/workspaces/{workspaceId}/traces/{traceId}'}</span></div><p className="mt-5 text-[11px] leading-relaxed text-[#aaa59a]">Live Owner-only surface. Runtime facts are read from the Gateway; the demo intentionally shows the contract boundary without inventing an execution.</p><div className="mt-6 grid grid-cols-3 gap-3 text-[10px]"><div className="border border-[#34312b] p-3"><div className="text-[#777268]">AUTH</div><div className="text-emerald-400 mt-1">OWNER</div></div><div className="border border-[#34312b] p-3"><div className="text-[#777268]">MODE</div><div className="text-amber-200 mt-1">JSON / SSE</div></div><div className="border border-[#34312b] p-3"><div className="text-[#777268]">STORAGE</div><div className="text-[#aaa59a] mt-1">METADATA ONLY</div></div></div></div>;
}
