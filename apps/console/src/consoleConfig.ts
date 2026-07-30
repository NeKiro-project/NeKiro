const requiredConfigurationNames = [
  'VITE_NEKIRO_API_BASE_URL',
  'VITE_NEKIRO_PROVIDER_ID',
  'VITE_NEKIRO_PROVIDER_TOKEN',
  'VITE_NEKIRO_OWNER_TOKEN',
  'VITE_NEKIRO_DEFAULT_WORKSPACE_ID',
  'VITE_NEKIRO_PUBLIC_AGENT_ORIGIN',
] as const;

export function requireConsoleConfiguration(env: Record<string, unknown>): void {
  for (const name of requiredConfigurationNames) {
    const value = env[name];
    if (typeof value !== 'string' || !value || value !== value.trim()) {
      throw new Error(`${name} is required and must not contain surrounding whitespace`);
    }
  }

  const publicOrigin = new URL(env.VITE_NEKIRO_PUBLIC_AGENT_ORIGIN as string);
  if (!['http:', 'https:'].includes(publicOrigin.protocol)
    || publicOrigin.username
    || publicOrigin.password
    || publicOrigin.pathname !== '/'
    || publicOrigin.search
    || publicOrigin.hash
    || publicOrigin.origin !== env.VITE_NEKIRO_PUBLIC_AGENT_ORIGIN) {
    throw new Error('VITE_NEKIRO_PUBLIC_AGENT_ORIGIN must be an exact HTTP or HTTPS origin');
  }
}

export function requirePublicConsoleConfiguration(env: Record<string, unknown>): void {
  for (const name of ['VITE_NEKIRO_API_BASE_URL', 'VITE_NEKIRO_PUBLIC_AGENT_ORIGIN']) {
    const value = env[name];
    if (typeof value !== 'string' || !value || value !== value.trim()) {
      throw new Error(`${name} is required and must not contain surrounding whitespace`);
    }
  }
}
