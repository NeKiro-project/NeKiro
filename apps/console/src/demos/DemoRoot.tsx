import React from 'react';
import {ArrowLeft, LayoutGrid, Layers, TerminalSquare, Sun} from 'lucide-react';

import GlassDemo from './glass/GlassDemo';
import TerminalDemo from './terminal/TerminalDemo';
import SaasDemo from './saas/SaasDemo';
import {type DemoId} from './routing';

export type {DemoId} from './routing';

export function navigateTo(id: DemoId) {
  window.location.hash = id === 'launcher' ? '/demo' : '/demo/' + id;
}

const DEMOS = [
  {
    id: 'glass' as const,
    name: 'Glass 2.0',
    tagline: 'Refined dark glassmorphism',
    description: 'The current style, leveled up: layered glass cards, gradient display type, glowing status, stats row, and hover micro-interactions.',
    icon: Layers,
    accent: 'from-indigo-500 to-violet-500',
  },
  {
    id: 'terminal' as const,
    name: 'Terminal',
    tagline: 'Monospace density, amber accent',
    description: 'Bloomberg-terminal minimalism. No rounded corners, no shadows — table-density rows, keyboard hints, one accent color.',
    icon: TerminalSquare,
    accent: 'from-amber-500 to-orange-500',
  },
  {
    id: 'saas' as const,
    name: 'Light SaaS',
    tagline: 'Vercel / Stripe approachability',
    description: 'Light canvas, white cards, soft shadows, colored badges and an agent card grid. Friendly and demo-ready.',
    icon: Sun,
    accent: 'from-sky-400 to-indigo-400',
  },
];

export default function DemoRoot({demo}: {demo: DemoId}) {
  if (demo === 'glass') return <GlassDemo />;
  if (demo === 'terminal') return <TerminalDemo />;
  if (demo === 'saas') return <SaasDemo />;
  return <Launcher />;
}

/** Floating pill shown inside every demo to hop between directions. */
export function DemoSwitcher({current}: {current: Exclude<DemoId, 'launcher'>; dark?: boolean}) {
  return (
    <div className="fixed bottom-4 left-1/2 -translate-x-1/2 z-[200] flex items-center gap-1 rounded-full border border-white/15 bg-black/70 backdrop-blur-md px-2 py-1.5 shadow-2xl">
      <button onClick={() => navigateTo('launcher')} title="Back to launcher" className="p-1.5 rounded-full text-white/60 hover:text-white hover:bg-white/10">
        <LayoutGrid size={13} />
      </button>
      <span className="w-px h-4 bg-white/15" />
      {DEMOS.map((d) => (
        <button
          key={d.id}
          onClick={() => navigateTo(d.id)}
          className={`px-3 py-1 rounded-full text-[11px] font-medium transition-colors ${
            current === d.id ? 'bg-white text-black' : 'text-white/60 hover:text-white hover:bg-white/10'
          }`}
        >
          {d.name}
        </button>
      ))}
      <span className="w-px h-4 bg-white/15" />
      <button onClick={() => { window.location.hash = ''; }} title="Back to production app" className="p-1.5 rounded-full text-white/60 hover:text-white hover:bg-white/10">
        <ArrowLeft size={13} />
      </button>
    </div>
  );
}

function Launcher() {
  return (
    <div className="fixed inset-0 z-[150] overflow-y-auto bg-[#05070f] text-slate-100">
      <div className="pointer-events-none fixed inset-0 opacity-60" style={{background: 'radial-gradient(60% 50% at 50% 0%, rgba(99,102,241,0.25), transparent 70%)'}} />
      <div className="relative max-w-5xl mx-auto px-8 py-16">
        <div className="text-[11px] uppercase tracking-[0.3em] text-indigo-400 mb-3">NeKiro Console · Redesign demos</div>
        <h1 className="text-4xl font-extrabold tracking-tight">Three directions. Same data. Pick one.</h1>
        <p className="text-slate-400 mt-3 max-w-2xl">
          Each demo renders the identical mock dataset — 6 Agent Cards, 1 Workspace, 4 Installations — with a fully styled
          Registry and Installations surface. Compare, then tell me which direction becomes production.
        </p>

        <div className="grid md:grid-cols-3 gap-5 mt-12">
          {DEMOS.map((d) => {
            const Icon = d.icon;
            return (
              <button
                key={d.id}
                onClick={() => navigateTo(d.id)}
                className="group text-left rounded-2xl border border-white/10 bg-white/[0.03] hover:bg-white/[0.06] hover:border-white/25 transition-all p-6 hover:-translate-y-1"
              >
                <div className={`w-11 h-11 rounded-xl bg-gradient-to-br ${d.accent} flex items-center justify-center text-white shadow-lg mb-5`}>
                  <Icon size={20} />
                </div>
                <div className="text-lg font-bold">{d.name}</div>
                <div className="text-xs text-indigo-300 mt-0.5">{d.tagline}</div>
                <p className="text-sm text-slate-400 mt-3 leading-relaxed">{d.description}</p>
                <div className="mt-5 text-xs font-semibold text-slate-300 opacity-0 group-hover:opacity-100 transition-opacity">Open demo →</div>
              </button>
            );
          })}
        </div>

        <button onClick={() => { window.location.hash = ''; }} className="mt-10 text-xs text-slate-500 hover:text-slate-300 flex items-center gap-1.5">
          <ArrowLeft size={12} /> Back to production app
        </button>
      </div>
    </div>
  );
}
