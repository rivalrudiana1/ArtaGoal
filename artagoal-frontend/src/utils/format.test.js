import { describe, expect, it, vi } from 'vitest'
import { formatDate, formatIDR, formatPercent, greeting } from './format.js'

describe('formatIDR', () => {
  it('memformat rupiah tanpa desimal', () => {
    expect(formatIDR(500000)).toContain('500.000')
    expect(formatIDR(500000)).toContain('Rp')
  })

  it('aman untuk input tak valid', () => {
    expect(formatIDR(undefined)).toContain('0')
    expect(formatIDR('abc')).toContain('0')
  })
})

describe('formatPercent', () => {
  it('menghitung persen progres', () => {
    expect(formatPercent(300, 1200)).toBe(25)
  })

  it('membatasi 100 dan aman untuk target nol', () => {
    expect(formatPercent(1500, 1000)).toBe(100)
    expect(formatPercent(100, 0)).toBe(0)
  })
})

describe('formatDate', () => {
  it('memformat tanggal id-ID dan aman untuk input kosong', () => {
    expect(formatDate('2026-10-07')).toContain('2026')
    expect(formatDate('')).toBe('-')
    expect(formatDate('bukan-tanggal')).toBe('-')
  })
})

describe('greeting', () => {
  it('menyapa sesuai jam', () => {
    vi.spyOn(Date.prototype, 'getHours').mockReturnValue(8)
    expect(greeting()).toBe('Selamat pagi')
    vi.spyOn(Date.prototype, 'getHours').mockReturnValue(20)
    expect(greeting()).toBe('Selamat malam')
    vi.restoreAllMocks()
  })
})
