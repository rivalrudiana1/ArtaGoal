import { describe, expect, it } from 'vitest'
import { apiErrorMessage } from './api.js'

describe('apiErrorMessage', () => {
  it('mengambil pesan backend {error}', () => {
    const err = { response: { data: { error: 'Target tidak ditemukan' } }, message: 'x' }
    expect(apiErrorMessage(err)).toBe('Target tidak ditemukan')
  })

  it('fallback ke pesan error lalu default', () => {
    expect(apiErrorMessage({ message: 'Network Error' })).toBe('Network Error')
    expect(apiErrorMessage({}, 'Gagal.')).toBe('Gagal.')
    expect(apiErrorMessage(null)).toBe('Terjadi kesalahan.')
  })
})
