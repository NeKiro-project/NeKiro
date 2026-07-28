export type DemoId = 'launcher' | 'glass' | 'terminal' | 'saas';

export function demoFromHash(hash: string): DemoId | null {
  switch (hash) {
    case '#/demo':
      return 'launcher';
    case '#/demo/glass':
      return 'glass';
    case '#/demo/terminal':
      return 'terminal';
    case '#/demo/saas':
      return 'saas';
    default:
      return null;
  }
}
