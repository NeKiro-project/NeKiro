import {defineConfig} from '@playwright/test';

const baseURL = process.env.NEKIRO_E2E_BASE_URL;
if (!baseURL || baseURL !== baseURL.trim() || !/^https?:\/\/[^/]+(?::\d+)?$/.test(baseURL)) {
  throw new Error('NEKIRO_E2E_BASE_URL must be an explicit browser server origin');
}

if (!process.env.NEKIRO_E2E_COMPOSE_FILE || !process.env.NEKIRO_E2E_COMPOSE_PROJECT) {
  throw new Error('NEKIRO_E2E_COMPOSE_FILE and NEKIRO_E2E_COMPOSE_PROJECT are required');
}

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  timeout: 120_000,
  expect: {timeout: 15_000},
  reporter: 'line',
  use: {
    baseURL,
    browserName: 'chromium',
    trace: 'off',
    screenshot: 'off',
    video: 'off',
  },
  webServer: {
    command: 'npm run preview -- --host 127.0.0.1 --port 4173',
    url: baseURL,
    reuseExistingServer: false,
    timeout: 120_000,
  },
});
