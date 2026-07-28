import React from 'react';
import {AlertTriangle, CheckCircle2, Database, Plus, Search, User} from 'lucide-react';

import type {PlatformErrorView, Workspace} from '../types';

interface HeaderProps {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  searchPlaceholder?: string;
  workspace: Workspace | null;
  workspaceDraft: string;
  setWorkspaceDraft: (workspaceId: string) => void;
  workspaceLoading: boolean;
  workspaceError: PlatformErrorView | null;
  onReadWorkspace: () => void;
  onCreateWorkspace: () => void;
  userLabel: string;
  apiConfigured: boolean;
}

export default function Header({
  searchQuery,
  setSearchQuery,
  searchPlaceholder = 'Search resource, agent, traces...',
  workspace,
  workspaceDraft,
  setWorkspaceDraft,
  workspaceLoading,
  workspaceError,
  onReadWorkspace,
  onCreateWorkspace,
  userLabel,
  apiConfigured,
}: HeaderProps) {
  return (
    <header className="fixed top-0 right-0 left-64 z-40 flex justify-between items-center px-7 bg-brand-bg/60 backdrop-blur-xl border-b border-brand-outline-variant h-16">
      <div className="glass-header-controls flex items-center gap-3 2xl:gap-4 min-w-0">
        <div className="glass-search-control relative flex items-center bg-brand-container border border-brand-outline-variant/60 rounded-xl px-2.5 py-1 h-9 transition-colors focus-within:border-brand-primary/50 focus-within:shadow-[0_0_0_3px_rgba(99,102,241,0.12)]">
          <Search size={14} className="text-brand-on-surface-variant mr-2" />
          <input
            type="text"
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            placeholder={searchPlaceholder}
            className="bg-transparent border-none outline-none text-brand-on-surface font-mono-code text-[11px] w-48 xl:w-64 2xl:w-80 p-0 m-0 placeholder-brand-on-surface-variant/40 focus:ring-0"
          />
        </div>

        <div className="glass-workspace-control flex items-center gap-2 bg-brand-container border border-brand-outline-variant/60 rounded-xl h-9 px-2">
          <Database size={13} className={workspace ? 'text-green-400' : 'text-brand-on-surface-variant'} />
          <input
            value={workspaceDraft}
            onChange={(event) => setWorkspaceDraft(event.target.value)}
            placeholder="workspace id"
            className="bg-transparent outline-none text-[10.5px] font-mono-code text-brand-on-surface w-24 xl:w-32 2xl:w-40 placeholder-brand-on-surface-variant/40"
          />
          <button
            onClick={onReadWorkspace}
            disabled={workspaceLoading || !workspaceDraft.trim()}
            className="text-[9.5px] px-2 py-0.5 rounded bg-brand-high text-brand-on-surface-variant hover:text-brand-on-surface disabled:opacity-40"
          >
            Load
          </button>
          <button
            onClick={onCreateWorkspace}
            disabled={workspaceLoading || !workspaceDraft.trim()}
            aria-label="Create workspace"
            title="Create workspace"
            className="glass-create-workspace text-[9.5px] px-2 py-0.5 rounded bg-brand-primary/20 text-brand-primary hover:bg-brand-primary/30 disabled:opacity-40 flex items-center gap-1"
          >
            <Plus size={10} />
            <span className="glass-create-label">Create</span>
          </button>
        </div>
      </div>

      <div className="glass-header-status pointer-events-none flex items-center gap-2 xl:gap-3">
        {workspaceError && (
          <div className="flex items-center gap-1.5 text-[10px] font-mono-code text-brand-error border border-brand-error/25 bg-brand-error-container/10 rounded px-2 py-1 max-w-[18rem] truncate" title={workspaceError.message}>
            <AlertTriangle size={12} />
            <span>{workspaceError.code ?? 'WORKSPACE_ERROR'}</span>
            {workspaceError.traceId && <span className="text-brand-on-surface-variant">trace {workspaceError.traceId}</span>}
          </div>
        )}

        <div className="glass-status-strip flex items-center gap-2 xl:gap-3 font-mono-label text-[10px]">
          <span className="px-2.5 py-1 bg-brand-container rounded border border-brand-outline-variant/50 text-brand-on-surface-variant flex items-center gap-1.5">
            <span className={'glass-status-dot w-1.5 h-1.5 rounded-full inline-block ' + (apiConfigured ? 'bg-green-400 text-green-400' : 'bg-brand-error text-brand-error')} />
            API: {apiConfigured ? 'configured' : 'missing'}
          </span>
          <span className="px-2.5 py-1 bg-brand-container rounded border border-brand-outline-variant/50 text-brand-on-surface-variant">
            Workspace: {workspace?.workspaceId ?? 'not selected'}
          </span>
          <span className="px-2.5 py-1 bg-brand-container rounded border border-brand-outline-variant/50 text-brand-on-surface-variant flex items-center gap-1.5">
            <User size={12} />
            {userLabel}
          </span>
          {workspace && <CheckCircle2 size={15} className="text-green-400" />}
        </div>
      </div>
    </header>
  );
}
