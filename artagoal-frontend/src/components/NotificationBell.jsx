import { useEffect, useState } from 'react'
import { Bell, Check, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { apiErrorMessage, goalService } from '../services/api.js'
import { formatDate } from '../utils/format.js'

const POLL_INTERVAL_MS = 60_000

export default function NotificationBell() {
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [markingId, setMarkingId] = useState('')

  useEffect(() => {
    let cancelled = false
    async function fetchNotifications() {
      try {
        const { data } = await goalService.getNotifications()
        if (!cancelled) setItems(Array.isArray(data) ? data : [])
      } catch {
        if (!cancelled) setItems([])
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    fetchNotifications()
    const timer = setInterval(fetchNotifications, POLL_INTERVAL_MS)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [])

  const unreadCount = items.filter((n) => !n.is_read).length

  async function handleMarkAsRead(id) {
    setMarkingId(id)
    try {
      const { data } = await goalService.markAsRead(id)
      setItems((prev) =>
        prev.map((n) =>
          n.id === id ? { ...n, is_read: data?.is_read ?? true } : n,
        ),
      )
    } catch (err) {
      toast.error(apiErrorMessage(err, 'Gagal menandai notifikasi.'))
    } finally {
      setMarkingId('')
    }
  }

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-label="Notifikasi"
        aria-expanded={open}
        className="relative flex h-9 w-9 items-center justify-center rounded-full border border-slate-200 text-slate-500 transition hover:border-emerald-300 hover:text-emerald-600"
      >
        <Bell className="h-4 w-4" />
        {unreadCount > 0 && (
          <span className="absolute -top-1 -right-1 flex h-5 min-w-5 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white">
            {unreadCount > 9 ? '9+' : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <>
          <button
            type="button"
            aria-label="Tutup notifikasi"
            onClick={() => setOpen(false)}
            className="fixed inset-0 z-10 cursor-default bg-transparent"
          />
          <div className="absolute right-0 z-20 mt-2 max-h-96 w-80 overflow-y-auto rounded-xl border border-slate-200 bg-white shadow-xl">
            <p className="sticky top-0 border-b border-slate-100 bg-white px-4 py-2.5 text-sm font-bold text-slate-800">
              Notifikasi
              {unreadCount > 0 && (
                <span className="ml-2 rounded-full bg-red-100 px-2 py-0.5 text-xs font-semibold text-red-600">
                  {unreadCount} baru
                </span>
              )}
            </p>
            {loading ? (
              <div className="flex items-center justify-center gap-2 px-4 py-8 text-sm text-slate-500">
                <Loader2 className="h-4 w-4 animate-spin" /> Memuat...
              </div>
            ) : items.length === 0 ? (
              <p className="px-4 py-8 text-center text-sm text-slate-500">
                Belum ada notifikasi. Santai, targetmu aman.
              </p>
            ) : (
              <ul className="divide-y divide-slate-100">
                {items.map((n) => (
                  <li
                    key={n.id}
                    className={`px-4 py-3 ${n.is_read ? '' : 'bg-emerald-50/60'}`}
                  >
                    <div className="flex items-start gap-2">
                      {!n.is_read && (
                        <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-red-500" />
                      )}
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-semibold text-slate-800">
                          {n.title}
                        </p>
                        <p className="mt-0.5 text-xs break-words text-slate-600">
                          {n.message}
                        </p>
                        <p className="mt-1 text-[11px] text-slate-400">
                          {formatDate(n.created_at)}
                        </p>
                        {!n.is_read && (
                          <button
                            type="button"
                            disabled={markingId === n.id}
                            onClick={() => handleMarkAsRead(n.id)}
                            className="mt-1.5 inline-flex items-center gap-1 rounded-lg border border-emerald-200 px-2 py-1 text-xs font-semibold text-emerald-700 transition hover:bg-emerald-50 disabled:opacity-60"
                          >
                            {markingId === n.id ? (
                              <Loader2 className="h-3 w-3 animate-spin" />
                            ) : (
                              <Check className="h-3 w-3" />
                            )}
                            Tandai Dibaca
                          </button>
                        )}
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </>
      )}
    </div>
  )
}
