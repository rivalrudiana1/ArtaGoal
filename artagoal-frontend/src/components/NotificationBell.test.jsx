import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NotificationBell from './NotificationBell.jsx'
import { goalService } from '../services/api.js'

vi.mock('../services/api.js', () => ({
  goalService: {
    getNotifications: vi.fn(),
    markAsRead: vi.fn(),
  },
  apiErrorMessage: (_err, fallback = 'Terjadi kesalahan.') => fallback,
}))

const NOTIFS = [
  {
    id: 'n1',
    title: 'Peringatan Tenggat Target',
    message: 'Target A tinggal 5 hari',
    is_read: false,
    created_at: '2026-10-06T10:00:00Z',
  },
  {
    id: 'n2',
    title: 'Info',
    message: 'Sudah dibaca',
    is_read: true,
    created_at: '2026-10-05T10:00:00Z',
  },
]

beforeEach(() => {
  vi.clearAllMocks()
  goalService.getNotifications.mockResolvedValue({ data: NOTIFS })
  goalService.markAsRead.mockResolvedValue({ data: { is_read: true } })
})

describe('NotificationBell', () => {
  it('menampilkan badge jumlah belum dibaca', async () => {
    render(<NotificationBell />)
    expect(await screen.findByText('1')).toBeInTheDocument()
  })

  it('membuka dropdown dan menandai dibaca', async () => {
    render(<NotificationBell />)
    fireEvent.click(await screen.findByRole('button', { name: /notifikasi/i }))
    expect(await screen.findByText('Peringatan Tenggat Target')).toBeInTheDocument()

    fireEvent.click(screen.getByText('Tandai Dibaca'))
    await waitFor(() => {
      expect(goalService.markAsRead).toHaveBeenCalledWith('n1')
    })
  })
})
