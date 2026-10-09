import { useEffect, useState } from 'react'
import { NavLink } from 'react-router-dom'
import { Bell, Home, Target, User } from 'lucide-react'
import { goalService } from '../services/api.js'
import { useAuth } from '../context/AuthContext.jsx'
import { authFileUrl } from '../services/api.js'

function initials(name) {
  const parts = String(name ?? '').trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

const linkBase =
  'relative flex min-h-[56px] min-w-[64px] flex-1 flex-col items-center justify-center gap-1 px-2 py-2 text-[11px] font-semibold transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400'

function linkClass({ isActive }) {
  return `${linkBase} ${
    isActive
      ? 'text-emerald-600 dark:text-emerald-400'
      : 'text-slate-500 hover:text-slate-800 active:text-slate-900 dark:text-slate-400 dark:hover:text-slate-200 dark:active:text-slate-100'
  }`
}

export default function BottomNav() {
  const { user } = useAuth()
  const [unread, setUnread] = useState(0)

  useEffect(() => {
    let cancelled = false
    async function fetchUnread() {
      try {
        const { data } = await goalService.getNotifications({ limit: 50 })
        if (cancelled) return
        const list = Array.isArray(data) ? data : []
        setUnread(list.filter((n) => !n.is_read).length)
      } catch {
        if (!cancelled) setUnread(0)
      }
    }
    fetchUnread()
    const timer = setInterval(fetchUnread, 60_000)
    window.addEventListener('focus', fetchUnread)
    return () => {
      cancelled = true
      clearInterval(timer)
      window.removeEventListener('focus', fetchUnread)
    }
  }, [])

  const avatar = authFileUrl(user?.avatar_url)

  return (
    <nav
      aria-label="Navigasi utama"
      className="fixed right-0 bottom-0 left-0 z-50 border-t border-slate-200 bg-white/90 backdrop-blur-md md:hidden dark:border-slate-800 dark:bg-slate-900/90"
      style={{ paddingBottom: 'env(safe-area-inset-bottom)' }}
    >
      <div className="mx-auto flex max-w-lg items-stretch justify-between gap-1 px-2">
        <NavLink to="/dashboard" className={linkClass} aria-label="Dashboard">
          {({ isActive }) => (
            <>
              <Home
                className="h-5 w-5"
                aria-hidden="true"
                strokeWidth={isActive ? 2.5 : 2}
              />
              <span>Dashboard</span>
              {isActive && (
                <span
                  aria-hidden="true"
                  className="absolute bottom-1 h-1 w-8 rounded-full bg-emerald-500 dark:bg-emerald-400"
                />
              )}
            </>
          )}
        </NavLink>

        <NavLink to="/goals/new" className={linkClass} aria-label="Target keuangan">
          {({ isActive }) => (
            <>
              <Target
                className="h-5 w-5"
                aria-hidden="true"
                strokeWidth={isActive ? 2.5 : 2}
              />
              <span>Target</span>
              {isActive && (
                <span
                  aria-hidden="true"
                  className="absolute bottom-1 h-1 w-8 rounded-full bg-emerald-500 dark:bg-emerald-400"
                />
              )}
            </>
          )}
        </NavLink>

        <NavLink to="/dashboard" className={linkClass} aria-label="Notifikasi">
          {() => (
            <>
              <span className="relative">
                <Bell className="h-5 w-5" aria-hidden="true" />
                {unread > 0 && (
                  <span
                    aria-label={`${unread} notifikasi belum dibaca`}
                    className="absolute -top-2 -right-2 flex h-5 min-w-5 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white"
                  >
                    {unread > 9 ? '9+' : unread}
                  </span>
                )}
              </span>
              <span>Notifikasi</span>
            </>
          )}
        </NavLink>

        <NavLink to="/profile" className={linkClass} aria-label="Profil akun">
          {({ isActive }) => (
            <>
              {avatar ? (
                <img
                  src={avatar}
                  alt=""
                  aria-hidden="true"
                  className={`h-6 w-6 rounded-full object-cover ring-2 ${
                    isActive
                      ? 'ring-emerald-500 dark:ring-emerald-400'
                      : 'ring-slate-200 dark:ring-slate-700'
                  }`}
                />
              ) : user?.name ? (
                <span
                  aria-hidden="true"
                  className={`flex h-6 w-6 items-center justify-center rounded-full text-[10px] font-extrabold ring-2 ${
                    isActive
                      ? 'bg-emerald-100 text-emerald-700 ring-emerald-500 dark:bg-emerald-500/20 dark:text-emerald-300 dark:ring-emerald-400'
                      : 'bg-slate-200 text-slate-600 ring-slate-300 dark:bg-slate-800 dark:text-slate-300 dark:ring-slate-700'
                  }`}
                >
                  {initials(user.name)}
                </span>
              ) : (
                <User
                  className="h-5 w-5"
                  aria-hidden="true"
                  strokeWidth={isActive ? 2.5 : 2}
                />
              )}
              <span>Profil</span>
              {isActive && (
                <span
                  aria-hidden="true"
                  className="absolute bottom-1 h-1 w-8 rounded-full bg-emerald-500 dark:bg-emerald-400"
                />
              )}
            </>
          )}
        </NavLink>
      </div>
    </nav>
  )
}
