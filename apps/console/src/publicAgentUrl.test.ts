import assert from 'node:assert/strict';
import test from 'node:test';

import {formatPublicAgentUrl, parsePublicAgentUrl} from './publicAgentUrl';

const origin = 'https://agents.nekiro.test';
const id = 'agt_0123456789abcdef0123456789abcdef';

test('public Agent URL round trips the stable identity', () => {
  const url = formatPublicAgentUrl(id, origin);
  assert.equal(url, origin + '/a/' + id);
  assert.equal(parsePublicAgentUrl(url, origin), id);
});

test('public Agent URL parser rejects aliases, normalization, and secrets', () => {
  for (const value of [
    ' ' + origin + '/a/' + id,
    origin + '/a/' + id + '/',
    origin + '/a/' + id + '?x=1',
    origin + '/a/' + id + '#fragment',
    origin + '/agents/' + id,
    'https://other.nekiro.test/a/' + id,
    origin + '/a/' + id.toUpperCase(),
    origin + '/a/agt_0123456789abcdef0123456789abcde%66',
    'https://user:secret@agents.nekiro.test/a/' + id,
  ]) {
    assert.throws(() => parsePublicAgentUrl(value, origin), /Public Agent URL|Public Agent ID/);
  }
});
