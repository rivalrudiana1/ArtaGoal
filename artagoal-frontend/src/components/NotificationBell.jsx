import { useEffect, useState } from 'react'
import { Bell, Check, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { apiErrorMessage, goalService } from '../services/api.js'
import { formatDate } from '../utils/format.js'

const POLL_INTERVAL_MS = 60_000

function urlBase64ToUint8Array(base64) {
  const padding = '='.repeat((4 - (base64.length % 4)) % 4)
  const raw = atob(base64.replace(/-/g, '+').replace(/_/g, '/') + padding)
  return Uint8Array.from([...raw].map((c) => c.charCodeAt(0)))
}

function pushSupported() {
  return (
    typeof window !== 'undefined' &&
    'serviceWorker' in navigator &&
    'PushManager' in window &&
    'Notification' in window
  )
}

export default function NotificationBell() {
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [markingId, setMarkingId] = useState('')

  useEffect(() => {
    let cancelled = false
    async function fetchNotifications() {
      try {
        const { data } = await goalService.getNotifications({ limit: 50 })
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
  const [pushState, setPushState] = useState(
    pushSupported() ? 'unknown' : 'unsupported',
  )

  async function refreshPushState() {
    if (!pushSupported()) {
      setPushState('unsupported')
      return
    }
    if (Notification.permission === 'denied') {
      setPushState('denied')
      return
    }
    try {
      const reg = await navigator.serviceWorker.ready
      const sub = await reg.pushManager.getSubscription()
      setPushState(sub ? 'on' : 'off')
    } catch {
      setPushState('off')
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react/set-state-in-effect -- sinkronisasi status langganan dari service worker eksternal
    refreshPushState()
  }, [open])

  async function handleEnablePush() {
    setPushState('loading')
    try {
      const permission = await Notification.requestPermission()
      if (permission !== 'granted') {
        setPushState(permission === 'denied' ? 'denied' : 'off')
        return
      }
      const { data } = await goalService.getVapidPublicKey()
      const reg = await navigator.serviceWorker.ready
      const sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(data.public_key),
      })
      const json = sub.toJSON()
      await goalService.savePushSubscription({
        endpoint: sub.endpoint,
        keys: { p256dh: json.keys.p256dh, auth: json.keys.auth },
      })
      setPushState('on')
      toast.success('Push notification aktif di perangkat ini!')
    } catch (err) {
      setPushState('off')
      toast.error(apiErrorMessage(err, 'Gagal mengaktifkan push.'))
    }
  }

  async function handleDisablePush() {
    setPushState('loading')
    try {
      const reg = await navigator.serviceWorker.ready
      const sub = await reg.pushManager.getSubscription()
      if (sub) {
        await goalService.deletePushSubscription(sub.endpoint).catch(() => {})
        await sub.unsubscribe()
      }
      setPushState('off')
      toast.success('Push notification dimatikan.')
    } catch (err) {
      toast.error(apiErrorMessage(err, 'Gagal mematikan push.'))
      refreshPushState()
    }
  }

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
        className="relative flex h-9 w-9 items-center justify-center rounded-full border border-slate-200 text-slate-500 transition hover:border-emerald-300 hover:text-emerald-600 dark:border-slate-700 dark:text-slate-300 dark:hover:border-emerald-500 dark:hover:text-emerald-400"
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
          <div className="absolute right-0 z-20 mt-2 max-h-96 w-80 overflow-y-auto rounded-xl border border-slate-200 bg-white shadow-xl dark:border-slate-700 dark:bg-slate-900">
            <p className="sticky top-0 border-b border-slate-100 bg-white px-4 py-2.5 text-sm font-bold text-slate-800 dark:border-slate-800 dark:bg-slate-900 dark:text-white">
              Notifikasi
              {unreadCount > 0 && (
                <span className="ml-2 rounded-full bg-red-100 px-2 py-0.5 text-xs font-semibold text-red-600 dark:bg-red-500/15 dark:text-red-400">
                  {unreadCount} baru
                </span>
              )}
            </p>
            {pushState !== 'unsupported' && (
              <div className="border-b border-slate-100 bg-slate-50 px-4 py-2 dark:border-slate-800 dark:bg-slate-800/50">
                {pushState === 'on' ? (
                  <button
                    type="button"
                    onClick={handleDisablePush}
                    className="text-xs font-semibold text-slate-500 hover:text-red-600 hover:underline dark:text-slate-400"
                  >
                    Push aktif di perangkat ini — matikan?
                  </button>
                ) : pushState === 'denied' ? (
                  <p className="text-xs text-slate-400 dark:text-slate-500">
                    Izin push diblokir browser. Aktifkan via pengaturan situs.
                  </p>
                ) : (
                  <button
                    type="button"
                    disabled={pushState === 'loading'}
                    onClick={handleEnablePush}
                    className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-700 hover:text-emerald-800 hover:underline disabled:opacity-60 dark:text-emerald-400 dark:hover:text-emerald-300"
                  >
                    {pushState === 'loading' && (
                      <Loader2 className="h-3 w-3 animate-spin" />
                    )}
                    Aktifkan push di perangkat ini
                  </button>
                )}
              </div>
            )}
            {loading ? (
              <div className="flex items-center justify-center gap-2 px-4 py-8 text-sm text-slate-500 dark:text-slate-400">
                <Loader2 className="h-4 w-4 animate-spin" /> Memuat...
              </div>
            ) : items.length === 0 ? (
              <p className="px-4 py-8 text-center text-sm text-slate-500 dark:text-slate-400">
                Belum ada notifikasi. Santai, targetmu aman.
              </p>
            ) : (
              <ul className="divide-y divide-slate-100 dark:divide-slate-800">
                {items.map((n) => (
                  <li
                    key={n.id}
                    className={`px-4 py-3 ${n.is_read ? '' : 'bg-emerald-50/60 dark:bg-emerald-500/10'}`}
                  >
                    <div className="flex items-start gap-2">
                      {!n.is_read && (
                        <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-red-500" />
                      )}
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-semibold text-slate-800 dark:text-slate-100">
                          {n.title}
                        </p>
                        <p className="mt-0.5 text-xs break-words text-slate-600 dark:text-slate-400">
                          {n.message}
                        </p>
                        <p className="mt-1 text-[11px] text-slate-400 dark:text-slate-500">
                          {formatDate(n.created_at)}
                        </p>
                        {!n.is_read && (
                          <button
                            type="button"
                            disabled={markingId === n.id}
                            onClick={() => handleMarkAsRead(n.id)}
                            className="mt-1.5 inline-flex items-center gap-1 rounded-lg border border-emerald-200 px-2 py-1 text-xs font-semibold text-emerald-700 transition hover:bg-emerald-50 disabled:opacity-60 dark:border-emerald-500/40 dark:text-emerald-400 dark:hover:bg-emerald-500/10"
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
