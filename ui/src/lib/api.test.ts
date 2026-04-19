import test from 'node:test';
import assert from 'node:assert';
import { saveSettings } from './api.ts';

test('saveSettings', async (t) => {
  const originalFetch = global.fetch;

  t.afterEach(() => {
    global.fetch = originalFetch;
  });

  await t.test('resolves on successful save', async () => {
    let fetchCalled = false;
    global.fetch = async (url: any, options: any) => {
      fetchCalled = true;
      assert.ok(url.endsWith('/settings'));
      assert.strictEqual(options?.method, 'POST');
      assert.deepStrictEqual(options?.headers, { "Content-Type": "application/json" });
      assert.strictEqual(options?.body, JSON.stringify({ apiKey: 'test-key', model: 'test-model', apiUrl: 'test-url' }));
      return { ok: true } as Response;
    };

    await saveSettings({ apiKey: 'test-key', model: 'test-model', apiUrl: 'test-url' });
    assert.strictEqual(fetchCalled, true);
  });

  await t.test('throws error on failed save', async () => {
    global.fetch = async () => {
      return { ok: false } as Response;
    };

    await assert.rejects(
      saveSettings({ apiKey: 'test-key', model: 'test-model', apiUrl: 'test-url' }),
      { message: 'Failed to save settings' }
    );
  });
});
