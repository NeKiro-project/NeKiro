import assert from 'node:assert/strict';
import test from 'node:test';

import {requireConsoleConfiguration} from './consoleConfig';

const validConfiguration = {
  VITE_NEKIRO_API_BASE_URL: 'https://gateway.example.test',
  VITE_NEKIRO_PROVIDER_ID: 'provider.main',
  VITE_NEKIRO_PROVIDER_TOKEN: 'provider-token',
  VITE_NEKIRO_OWNER_TOKEN: 'owner-token',
  VITE_NEKIRO_DEFAULT_WORKSPACE_ID: 'workspace.main',
};

test('console configuration requires all production identities and the Workspace', () => {
  assert.doesNotThrow(() => requireConsoleConfiguration(validConfiguration));
  for (const name of Object.keys(validConfiguration)) {
    const missing = {...validConfiguration};
    delete missing[name as keyof typeof validConfiguration];
    assert.throws(() => requireConsoleConfiguration(missing), new RegExp(name));
  }
});

test('console configuration rejects whitespace-padded values', () => {
  assert.throws(
    () => requireConsoleConfiguration({...validConfiguration, VITE_NEKIRO_OWNER_TOKEN: ' owner-token'}),
    /VITE_NEKIRO_OWNER_TOKEN/,
  );
});
