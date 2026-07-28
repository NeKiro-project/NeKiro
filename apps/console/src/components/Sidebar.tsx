import React from 'react';
import { Database, Cpu, PlayCircle, BookOpen, Settings, HelpCircle, Terminal, ShieldCheck } from 'lucide-react';

interface SidebarProps {
  activeTab: 'registry' | 'trusted' | 'installations' | 'invocations' | 'ledger';
  setActiveTab: (tab: 'registry' | 'trusted' | 'installations' | 'invocations' | 'ledger') => void;
  onOpenSettings?: () => void;
  onOpenSupport?: () => void;
}

export default function Sidebar({ activeTab, setActiveTab, onOpenSettings, onOpenSupport }: SidebarProps) {
  const navItems = [
    { id: 'registry', name: 'Registry', icon: Database },
    { id: 'trusted', name: 'Trusted Publication', icon: ShieldCheck },
    { id: 'installations', name: 'Installations', icon: Cpu },
    { id: 'invocations', name: 'Invocations', icon: PlayCircle },
    { id: 'ledger', name: 'Ledger', icon: BookOpen },
  ] as const;

  return (
    <aside
      id="sidebar"
      className="w-64 h-screen fixed left-0 top-0 bg-brand-lowest border-r border-brand-outline-variant flex flex-col py-5 overflow-y-auto z-50 shadow-lg"
    >
      {/* Branding Header */}
      <div className="px-5 mb-9 flex items-center gap-3">
        <div className="relative w-9 h-9 rounded-xl bg-gradient-to-br from-indigo-500 to-violet-600 flex items-center justify-center text-white shadow-[0_0_24px_rgba(99,102,241,0.45)]">
          <Terminal size={17} />
        </div>
        <div>
          <h1 className="font-headline-md text-sm font-extrabold text-brand-on-surface leading-none tracking-tight">
            NeKiro
          </h1>
          <p className="font-mono-label text-[10px] text-brand-on-surface-variant mt-1 uppercase tracking-wider">
            Infrastructure Orchestrator
          </p>
        </div>
      </div>

      {/* Navigation Items */}
      <nav className="flex-1 px-3 space-y-1">
        {navItems.map((item) => {
          const IconComponent = item.icon;
          const isActive = activeTab === item.id;
          return (
            <button
              key={item.id}
              onClick={() => setActiveTab(item.id)}
              aria-label={item.name}
              title={item.name}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl font-mono-label text-xs transition-all duration-200 cursor-pointer active:scale-98 text-left border ${
                isActive
                  ? 'text-white bg-gradient-to-r from-indigo-500/20 to-violet-500/10 border-indigo-400/25 shadow-[inset_0_1px_0_rgba(255,255,255,0.06),0_0_20px_rgba(99,102,241,0.15)]'
                  : 'text-brand-on-surface-variant border-transparent hover:text-brand-on-surface hover:bg-brand-high'
              }`}
            >
              <IconComponent size={16} className={isActive ? 'text-brand-primary' : 'text-brand-on-surface-variant'} />
              <span>{item.name}</span>
            </button>
          );
        })}
      </nav>

      {/* Footer Settings & Support */}
      <div className="px-3 mt-auto space-y-1 pt-4 border-t border-brand-outline-variant/30">
        <button
          onClick={onOpenSettings}
          aria-label="Settings"
          title="Settings"
          className="w-full flex items-center gap-3 px-3 py-2.5 rounded font-mono-label text-xs text-brand-on-surface-variant hover:text-brand-on-surface hover:bg-brand-high transition-all duration-150 text-left"
        >
          <Settings size={16} />
          <span>Settings</span>
        </button>
        <button
          onClick={onOpenSupport}
          aria-label="Support"
          title="Support"
          className="w-full flex items-center gap-3 px-3 py-2.5 rounded font-mono-label text-xs text-brand-on-surface-variant hover:text-brand-on-surface hover:bg-brand-high transition-all duration-150 text-left"
        >
          <HelpCircle size={16} />
          <span>Support</span>
        </button>
      </div>
    </aside>
  );
}
