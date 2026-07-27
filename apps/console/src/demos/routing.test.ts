import assert from 'node:assert/strict';
import test from 'node:test';

import {demoFromHash} from './routing';

test('demo routing accepts only the four comparison hashes', () => {
  assert.equal(demoFromHash('#/demo'), 'launcher');
  assert.equal(demoFromHash('#/demo/glass'), 'glass');
  assert.equal(demoFromHash('#/demo/terminal'), 'terminal');
  assert.equal(demoFromHash('#/demo/saas'), 'saas');
  assert.equal(demoFromHash('#/demoglass'), null);
  assert.equal(demoFromHash('#/demo/unknown'), null);
  assert.equal(demoFromHash(''), null);
});
