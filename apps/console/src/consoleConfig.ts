const requiredConfigurationNames = [
  'VITE_NEKIRO_API_BASE_URL',
  'VITE_NEKIRO_PROVIDER_ID',
  'VITE_NEKIRO_PROVIDER_TOKEN',
  'VITE_NEKIRO_OWNER_TOKEN',
  'VITE_NEKIRO_DEFAULT_WORKSPACE_ID',
] as const;

export function requireConsoleConfiguration(env: Record<string, unknown>): void {
  for (const name of requiredConfigurationNames) {
    const value = env[name];
    if (typeof value !== 'string' || !value || value !== value.trim()) {
      throw new Error(`${name} is required and must not contain surrounding whitespace`);
    }
  }
}
