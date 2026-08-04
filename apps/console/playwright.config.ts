import {defineConfig} from '@playwright/test';

const baseURL = process.env.NEKIRO_E2E_BASE_URL;
const previewOrigin = 'http://127.0.0.1:4173';
const browserExecutablePath = process.env.NEKIRO_E2E_BROWSER_EXECUTABLE_PATH;
if (!baseURL || baseURL !== previewOrigin) {
  throw new Error(`NEKIRO_E2E_BASE_URL must equal the production preview origin ${previewOrigin}`);
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
    ...(browserExecutablePath ? {launchOptions: {executablePath: browserExecutablePath}} : {}),
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
