import React, {useCallback, useEffect, useMemo, useRef, useState} from 'react';
import {AnimatePresence, motion} from 'motion/react';
import {CheckCircle2, Cpu, HelpCircle, ShieldAlert, X} from 'lucide-react';

import {mapCatalogEntry, NekiroApiClient, NekiroApiError, toPlatformErrorView, validateTrustedInstallation, type AgentCardV02, type AgentRelease} from './api/nekiro';
import {agentKey, isCurrentRequest, matchesPublishedRelease, nextRequestGeneration} from './consolePolicy';
import Header from './components/Header';
import InstallationsTab from './components/InstallationsTab';
import InvocationsTab from './components/InvocationsTab';
import LedgerTab from './components/LedgerTab';
import RegistryTab from './components/RegistryTab';
import Sidebar from './components/Sidebar';
import TrustedPublicationTab from './components/TrustedPublicationTab';
import {requireConsoleConfiguration} from './consoleConfig';
import type {Agent, Installation, InstallationStatus, PlatformErrorView, Workspace} from './types';

export default function App() {
  requireConsoleConfiguration(import.meta.env);
  const [activeTab, setActiveTab] = useState<'registry' | 'trusted' | 'installations' | 'invocations' | 'ledger'>('registry');
  const [searchQuery, setSearchQuery] = useState('');
  const [agents, setAgents] = useState<Agent[]>([]);
  const [providerAgents, setProviderAgents] = useState<Agent[]>([]);
  const [draftAgents, setDraftAgents] = useState<Agent[]>([]);
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [catalogError, setCatalogError] = useState<PlatformErrorView | null>(null);
  const [catalogReady, setCatalogReady] = useState(false);
  const [providerCatalogError, setProviderCatalogError] = useState<PlatformErrorView | null>(null);
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [workspaceDraft, setWorkspaceDraft] = useState(import.meta.env.VITE_NEKIRO_DEFAULT_WORKSPACE_ID ?? '');
  const [workspaceLoading, setWorkspaceLoading] = useState(false);
  const [workspaceError, setWorkspaceError] = useState<PlatformErrorView | null>(null);
  const [installations, setInstallations] = useState<Installation[]>([]);
  const [installationLoading, setInstallationLoading] = useState(false);
  const [installationError, setInstallationError] = useState<PlatformErrorView | null>(null);
  const [showSettings, setShowSettings] = useState(false);
  const [showSupport, setShowSupport] = useState(false);
  const catalogRequestGeneration = useRef(0);
  const providerCatalogRequestGeneration = useRef(0);
  const workspaceRequestGeneration = useRef(0);
  const installationRequestGeneration = useRef(0);

  const providerClient = useMemo(
    () => new NekiroApiClient({
      baseUrl: import.meta.env.VITE_NEKIRO_API_BASE_URL,
      token: import.meta.env.VITE_NEKIRO_PROVIDER_TOKEN,
    }),
    [],
  );
  const ownerClient = useMemo(
    () => new NekiroApiClient({
      baseUrl: import.meta.env.VITE_NEKIRO_API_BASE_URL,
      token: import.meta.env.VITE_NEKIRO_OWNER_TOKEN,
    }),
    [],
  );

  const loadAgents = useCallback(async (query = '') => {
    const generation = nextRequestGeneration(catalogRequestGeneration.current);
    catalogRequestGeneration.current = generation;
    setCatalogLoading(true);
    setCatalogError(null);
    try {
      const response = await ownerClient.searchAgents(query.trim() ? {query: query.trim()} : undefined);
      if (!isCurrentRequest(generation, catalogRequestGeneration.current)) return;
      setAgents(response.items.map(mapCatalogEntry));
      setCatalogReady(true);
    } catch (error) {
      if (!isCurrentRequest(generation, catalogRequestGeneration.current)) return;
      setCatalogError(toPlatformErrorView(error, 'Unable to load the NeKiro Catalog.'));
    } finally {
      if (isCurrentRequest(generation, catalogRequestGeneration.current)) setCatalogLoading(false);
    }
  }, [ownerClient]);

  const loadProviderAgents = useCallback(async (query = '') => {
    const generation = nextRequestGeneration(providerCatalogRequestGeneration.current);
    providerCatalogRequestGeneration.current = generation;
    setProviderCatalogError(null);
    try {
      const providerId = import.meta.env.VITE_NEKIRO_PROVIDER_ID;
      const response = await providerClient.searchAgents({ownerId: providerId, ...(query.trim() ? {query: query.trim()} : {})});
      if (!isCurrentRequest(generation, providerCatalogRequestGeneration.current)) return;
      setProviderAgents(response.items.map(mapCatalogEntry).filter((agent) => agent.ownerId === providerId));
    } catch (error) {
      if (!isCurrentRequest(generation, providerCatalogRequestGeneration.current)) return;
      setProviderCatalogError(toPlatformErrorView(error, 'Unable to load provider-owned Agent Cards.'));
    }
  }, [providerClient]);

  const loadWorkspace = useCallback(async (workspaceId: string) => {
    const generation = nextRequestGeneration(workspaceRequestGeneration.current);
    workspaceRequestGeneration.current = generation;
    setWorkspaceLoading(true);
    setWorkspaceError(null);
    try {
      const value = await ownerClient.getWorkspace(workspaceId);
      if (!isCurrentRequest(generation, workspaceRequestGeneration.current)) return null;
      setWorkspace(value);
      setWorkspaceDraft(value.workspaceId);
      return value;
    } catch (error) {
      if (!isCurrentRequest(generation, workspaceRequestGeneration.current)) return null;
      setWorkspace(null);
      setInstallations([]);
      setWorkspaceError(toPlatformErrorView(error, 'Unable to load Workspace.'));
      return null;
    } finally {
      if (isCurrentRequest(generation, workspaceRequestGeneration.current)) setWorkspaceLoading(false);
    }
  }, [ownerClient]);

  const loadInstallations = useCallback(async (workspaceId = workspace?.workspaceId) => {
    const generation = nextRequestGeneration(installationRequestGeneration.current);
    installationRequestGeneration.current = generation;
    if (!workspaceId) {
      setInstallations([]);
      return;
    }
    setInstallationLoading(true);
    setInstallationError(null);
    try {
      const response = await ownerClient.listInstallations(workspaceId, {limit: 100});
      if (!isCurrentRequest(generation, installationRequestGeneration.current)) return;
      setInstallations(response.items);
    } catch (error) {
      if (!isCurrentRequest(generation, installationRequestGeneration.current)) return;
      setInstallationError(toPlatformErrorView(error, 'Unable to load Workspace Installations.'));
    } finally {
      if (isCurrentRequest(generation, installationRequestGeneration.current)) setInstallationLoading(false);
    }
  }, [ownerClient, workspace?.workspaceId]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadAgents(searchQuery);
      void loadProviderAgents(searchQuery);
    }, 250);
    return () => window.clearTimeout(timer);
  }, [loadAgents, loadProviderAgents, searchQuery]);

  useEffect(() => {
    const defaultWorkspaceId = import.meta.env.VITE_NEKIRO_DEFAULT_WORKSPACE_ID;
    if (defaultWorkspaceId) {
      void loadWorkspace(defaultWorkspaceId).then((value) => {
        if (value) {
          void loadInstallations(value.workspaceId);
        }
      });
    }
  }, [loadInstallations, loadWorkspace]);

  const handleCreateWorkspace = async () => {
    setWorkspaceLoading(true);
    setWorkspaceError(null);
    try {
      const value = await ownerClient.createWorkspace(workspaceDraft);
      setWorkspace(value);
      setWorkspaceDraft(value.workspaceId);
      await loadInstallations(value.workspaceId);
    } catch (error) {
      setWorkspaceError(toPlatformErrorView(error, 'Unable to create Workspace.'));
    } finally {
      setWorkspaceLoading(false);
    }
  };

  const handleReadWorkspace = async () => {
    const value = await loadWorkspace(workspaceDraft);
    if (value) {
      await loadInstallations(value.workspaceId);
    }
  };

  const handleRegisterAgent = async (card: AgentCardV02) => {
    const entry = await providerClient.registerAgent(card);
    const draftAgent = mapCatalogEntry(entry);
    setDraftAgents((current) => upsertAgent(current, draftAgent));
    await Promise.all([loadAgents(searchQuery), loadProviderAgents(searchQuery)]);
    return draftAgent;
  };

  const handlePublishAgent = async (agent: Agent) => {
    await providerClient.publishAgentVersion(agent.id, agent.version);
    setDraftAgents((current) => current.filter((draft) => agentKey(draft) !== agentKey(agent)));
    await Promise.all([loadAgents(searchQuery), loadProviderAgents(searchQuery)]);
  };

  const handleInstallAgent = async (agent: Agent, release: AgentRelease, acceptedPermissions: string[]) => {
    if (!workspace) {
      throw new Error('Select or create a Workspace before installing an Agent.');
    }
    if (!matchesPublishedRelease(release, agent)) {
      throw new NekiroApiError(0, 'The selected Release is not a published match for the selected Agent Card.', 'INVALID_RESPONSE');
    }
    const installation = await ownerClient.installAgent(workspace.workspaceId, {
      agentId: agent.id,
      versionConstraint: release.agentCardVersion,
      acceptedPermissions,
    });
    validateTrustedInstallation(installation, release, agent.id);
    await loadInstallations(workspace.workspaceId);
  };

  const handleUpdateInstallation = async (installation: Installation, status: Exclude<InstallationStatus, 'uninstalled'>) => {
    if (!workspace) {
      return;
    }
    setInstallationError(null);
    try {
      await ownerClient.updateInstallation(workspace.workspaceId, installation.installationId, status);
      await loadInstallations(workspace.workspaceId);
    } catch (error) {
      setInstallationError(toPlatformErrorView(error, 'Unable to update Installation.'));
    }
  };

  const handleUninstall = async (installation: Installation) => {
    if (!workspace) {
      return false;
    }
    setInstallationError(null);
    try {
      await ownerClient.uninstallAgent(workspace.workspaceId, installation.installationId);
      await loadInstallations(workspace.workspaceId);
      return true;
    } catch (error) {
      setInstallationError(toPlatformErrorView(error, 'Unable to uninstall Agent.'));
      return false;
    }
  };

  const getSearchPlaceholder = () => {
    switch (activeTab) {
      case 'registry':
        return 'Search agent name, description, capability...';
      case 'trusted':
        return 'Filter registered Agent Cards...';
      case 'installations':
        return 'Search installation id, agent id, pinned version...';
      case 'invocations':
        return 'Filter active Workspace invocations...';
      case 'ledger':
        return 'Read Invocation or Trace metadata...';
    }
  };

  return (
    <div className="glass-app bg-brand-bg text-brand-on-surface font-sans h-screen w-screen overflow-hidden flex select-none relative">
      <div className="mesh-bg-container">
        <div className="mesh-blob blob1" />
        <div className="mesh-blob blob2" />
        <div className="mesh-blob blob3" />
        <div className="mesh-blob blob4" />
      </div>

      <Sidebar
        activeTab={activeTab}
        setActiveTab={(tab) => {
          setActiveTab(tab);
          setSearchQuery('');
        }}
        onOpenSettings={() => setShowSettings(true)}
        onOpenSupport={() => setShowSupport(true)}
      />

      <Header
        searchQuery={searchQuery}
        setSearchQuery={setSearchQuery}
        searchPlaceholder={getSearchPlaceholder()}
        workspace={workspace}
        workspaceDraft={workspaceDraft}
        setWorkspaceDraft={setWorkspaceDraft}
        workspaceLoading={workspaceLoading}
        workspaceError={workspaceError}
        onReadWorkspace={handleReadWorkspace}
        onCreateWorkspace={handleCreateWorkspace}
        userLabel={workspace?.ownerId ?? 'Workspace owner'}
        apiConfigured={Boolean(import.meta.env.VITE_NEKIRO_API_BASE_URL && import.meta.env.VITE_NEKIRO_PROVIDER_ID && import.meta.env.VITE_NEKIRO_PROVIDER_TOKEN && import.meta.env.VITE_NEKIRO_OWNER_TOKEN && import.meta.env.VITE_NEKIRO_DEFAULT_WORKSPACE_ID)}
      />

      <main className="ml-64 mt-16 w-[calc(100vw-256px)] h-[calc(100vh-64px)] overflow-y-auto bg-brand-bg p-7 relative">
        <AnimatePresence mode="wait" initial={false}>
          {activeTab === 'registry' && (
            <motion.div key="registry" initial={{opacity: 0, y: 10}} animate={{opacity: 1, y: 0}} exit={{opacity: 0, y: -10}} transition={{duration: 0.2}} className="w-full h-full">
              <RegistryTab
                agents={agents}
                draftAgents={draftAgents}
                onRegisterAgent={handleRegisterAgent}
                onPublishAgent={handlePublishAgent}
                catalogLoading={catalogLoading}
                catalogError={catalogError}
                catalogReady={catalogReady}
                defaultOwnerId={import.meta.env.VITE_NEKIRO_PROVIDER_ID ?? ''}
                defaultOwnerName={import.meta.env.VITE_NEKIRO_PROVIDER_NAME ?? ''}
                searchQuery={searchQuery}
              />
            </motion.div>
          )}

          {activeTab === 'trusted' && (
            <motion.div key="trusted" initial={{opacity: 0, y: 10}} animate={{opacity: 1, y: 0}} exit={{opacity: 0, y: -10}} transition={{duration: 0.2}} className="w-full h-full">
              <TrustedPublicationTab
                providerId={import.meta.env.VITE_NEKIRO_PROVIDER_ID ?? ''}
                client={providerClient}
                agents={providerAgents}
                draftAgents={draftAgents}
                providerCatalogError={providerCatalogError}
                onRefresh={() => void loadProviderAgents(searchQuery)}
              />
            </motion.div>
          )}

          {activeTab === 'installations' && (
            <motion.div key="installations" initial={{opacity: 0, y: 10}} animate={{opacity: 1, y: 0}} exit={{opacity: 0, y: -10}} transition={{duration: 0.2}} className="w-full h-full">
              <InstallationsTab
                workspace={workspace}
                agents={agents}
                installations={installations}
                loading={installationLoading}
                error={installationError}
                searchQuery={searchQuery}
                client={ownerClient}
                onInstallAgent={handleInstallAgent}
                onUpdateInstallation={handleUpdateInstallation}
                onUninstall={handleUninstall}
                onRefresh={() => void loadInstallations()}
              />
            </motion.div>
          )}

          {activeTab === 'invocations' && (
            <motion.div key="invocations" initial={{opacity: 0, y: 10}} animate={{opacity: 1, y: 0}} exit={{opacity: 0, y: -10}} transition={{duration: 0.2}} className="w-full h-full">
              <InvocationsTab workspace={workspace} installations={installations} client={ownerClient} />
            </motion.div>
          )}

          {activeTab === 'ledger' && (
            <motion.div key="ledger" initial={{opacity: 0, y: 10}} animate={{opacity: 1, y: 0}} exit={{opacity: 0, y: -10}} transition={{duration: 0.2}} className="w-full h-full">
              <LedgerTab workspace={workspace} client={ownerClient} />
            </motion.div>
          )}
        </AnimatePresence>
      </main>

      {showSettings && (
        <Overlay title="Control Plane Settings" icon={<Cpu size={22} />} onClose={() => setShowSettings(false)}>
          <div className="space-y-3 text-sm text-brand-on-surface-variant">
            <p>Base URL: <span className="font-mono-code text-brand-on-surface">{import.meta.env.VITE_NEKIRO_API_BASE_URL || 'not configured'}</span></p>
            <p>Provider context: <span className="font-mono-code text-brand-on-surface">VITE_NEKIRO_PROVIDER_ID</span> + <span className="font-mono-code text-brand-on-surface">VITE_NEKIRO_PROVIDER_TOKEN</span></p>
            <p>Workspace owner context: <span className="font-mono-code text-brand-on-surface">VITE_NEKIRO_OWNER_TOKEN</span> (credentials are never persisted in local storage)</p>
            <p>Default Workspace: <span className="font-mono-code text-brand-on-surface">{import.meta.env.VITE_NEKIRO_DEFAULT_WORKSPACE_ID || 'manual selection'}</span></p>
          </div>
        </Overlay>
      )}

      {showSupport && (
        <Overlay title="MVP Boundary" icon={<HelpCircle size={22} />} onClose={() => setShowSupport(false)}>
          <div className="space-y-3 text-sm text-brand-on-surface-variant">
            <p>Live surfaces: Registry, Workspace, Installations, Invocation Dispatch, and metadata-only Ledger through public Gateway routes.</p>
            <p>Runtime reads are Owner-authorized and Workspace-scoped. The Console never stores Agent secrets or fabricates Ledger events.</p>
          </div>
        </Overlay>
      )}
    </div>
  );
}

function Overlay({title, icon, children, onClose}: {title: string; icon: React.ReactNode; children: React.ReactNode; onClose: () => void}) {
  return (
    <div className="fixed inset-0 bg-black/55 backdrop-blur-sm z-[100] flex items-center justify-center p-4">
      <div className="bg-brand-low border border-brand-outline-variant rounded-xl shadow-2xl w-full max-w-lg overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b border-brand-outline-variant/60">
          <div className="flex items-center gap-3 text-brand-primary">
            {icon}
            <h2 className="font-headline-md text-sm font-bold text-brand-on-surface">{title}</h2>
          </div>
          <button onClick={onClose} className="p-1 rounded hover:bg-brand-high text-brand-on-surface-variant hover:text-brand-on-surface">
            <X size={18} />
          </button>
        </div>
        <div className="p-5">{children}</div>
        <div className="px-5 py-3 border-t border-brand-outline-variant/40 flex items-center gap-2 text-xs text-brand-on-surface-variant">
          <ShieldAlert size={14} />
          <span>Only public Gateway routes are called from the browser.</span>
          <CheckCircle2 size={14} className="ml-auto text-green-400" />
        </div>
      </div>
    </div>
  );
}

function upsertAgent(agents: Agent[], next: Agent): Agent[] {
  return [next, ...agents.filter((agent) => agentKey(agent) !== agentKey(next))];
}
