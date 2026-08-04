const publicAgentIDPattern = /^agt_[0-9a-f]{32}$/;

export function parsePublicAgentUrl(value: string, expectedOrigin: string): string {
  if (typeof value !== 'string' || value === '' || value !== value.trim()) {
    throw new Error('Public Agent URL must be the exact canonical URL.');
  }
  const origin = validateOrigin(expectedOrigin);
  const parsed = new URL(value);
  if (parsed.href !== value || parsed.origin !== origin || parsed.username || parsed.password || parsed.search || parsed.hash) {
    throw new Error('Public Agent URL origin or syntax is unsupported.');
  }
  const prefix = origin + '/a/';
  if (!value.startsWith(prefix)) throw new Error('Public Agent URL path is unsupported.');
  const publicAgentID = value.slice(prefix.length);
  if (!publicAgentIDPattern.test(publicAgentID)) throw new Error('Public Agent ID is invalid.');
  if (value !== formatPublicAgentUrl(publicAgentID, origin)) throw new Error('Public Agent URL is not canonical.');
  return publicAgentID;
}

export function formatPublicAgentUrl(publicAgentID: string, expectedOrigin: string): string {
  const origin = validateOrigin(expectedOrigin);
  if (!publicAgentIDPattern.test(publicAgentID)) throw new Error('Public Agent ID is invalid.');
  return origin + '/a/' + publicAgentID;
}

function validateOrigin(value: string): string {
  if (typeof value !== 'string' || value === '' || value !== value.trim()) throw new Error('Public Agent origin is required.');
  const parsed = new URL(value);
  if (!['http:', 'https:'].includes(parsed.protocol) || parsed.username || parsed.password || parsed.pathname !== '/' || value.endsWith('/') || parsed.search || parsed.hash || parsed.origin !== value) {
    throw new Error('Public Agent origin is invalid.');
  }
  return value;
}
