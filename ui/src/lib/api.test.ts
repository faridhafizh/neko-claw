import { describe, it, mock, beforeEach, afterEach } from 'node:test';
import * as assert from 'node:assert';
import { approveCommand } from './api';

describe('approveCommand', () => {
  const originalFetch = global.fetch;

  beforeEach(() => {
    global.fetch = mock.fn();
  });

  afterEach(() => {
    global.fetch = originalFetch;
    mock.restoreAll();
  });

  it('should resolve with data on success', async () => {
    const mockData = { status: 'success', output: 'ok', reply: 'done' };
    global.fetch = mock.fn(async () => {
      return {
        ok: true,
        json: async () => mockData
      } as Response;
    });

    const result = await approveCommand('123');
    assert.deepStrictEqual(result, mockData);

    assert.strictEqual((global.fetch as any).mock.calls.length, 1);
    const callArgs = (global.fetch as any).mock.calls[0].arguments;
    assert.strictEqual(callArgs[0], 'http://localhost:8080/api/command/approve');
    assert.strictEqual(callArgs[1].method, 'POST');
    assert.strictEqual(callArgs[1].body, JSON.stringify({ id: '123' }));
  });

  it('should throw error with text on failure', async () => {
    global.fetch = mock.fn(async () => {
      return {
        ok: false,
        text: async () => 'Custom error message'
      } as Response;
    });

    await assert.rejects(
      async () => { await approveCommand('123'); },
      { message: 'Custom error message' }
    );
  });

  it('should throw default error on failure without text', async () => {
    global.fetch = mock.fn(async () => {
      return {
        ok: false,
        text: async () => ''
      } as Response;
    });

    await assert.rejects(
      async () => { await approveCommand('123'); },
      { message: 'Failed to approve command' }
    );
  });
});
