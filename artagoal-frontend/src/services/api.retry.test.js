import axios from 'axios'
import { describe, expect, it } from 'vitest'
import './api.js'

function flakyAdapter(failures, mode = 'http500') {
  let calls = 0
  const adapter = async (config) => {
    calls += 1
    if (calls <= failures) {
      if (mode === 'network') {
        const err = new Error('Network Error')
        err.config = config
        throw err
      }
      const err = new Error('Request failed with status code 500')
      err.config = config
      err.response = { status: 500, data: { error: 'x' }, headers: {}, config }
      err.isAxiosError = true
      throw err
    }
    return {
      data: { ok: true },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    }
  }
  return { adapter, calls: () => calls }
}

describe('axios auto-retry interceptor', () => {
  it('mencoba ulang 1x saat 500 lalu sukses', async () => {
    const f = flakyAdapter(1)
    const res = await axios.get('http://localhost:1/retry-500', {
      adapter: f.adapter,
    })
    expect(res.data).toEqual({ ok: true })
    expect(f.calls()).toBe(2)
  })

  it('mencoba ulang 1x saat network error lalu sukses', async () => {
    const f = flakyAdapter(1, 'network')
    const res = await axios.get('http://localhost:1/retry-net', {
      adapter: f.adapter,
    })
    expect(res.data).toEqual({ ok: true })
    expect(f.calls()).toBe(2)
  })

  it('mencoba ulang hingga sukses pada kegagalan ke-2 (total 3 request)', async () => {
    const f = flakyAdapter(2)
    const res = await axios.get('http://localhost:1/retry-twice', {
      adapter: f.adapter,
    })
    expect(res.data).toEqual({ ok: true })
    expect(f.calls()).toBe(3)
  })

  it('menyerah setelah 3x retry (total 4 request)', async () => {
    const f = flakyAdapter(99)
    await expect(
      axios.get('http://localhost:1/retry-fail', { adapter: f.adapter }),
    ).rejects.toThrow()
    expect(f.calls()).toBe(4)
  })
})
