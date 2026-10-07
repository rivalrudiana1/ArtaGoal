import { useEffect, useState } from 'react'
import { Flame, Medal, PiggyBank } from 'lucide-react'
import { goalService } from '../services/api.js'

function StatItem({ icon, value, label }) {
  return (
    <div className="flex items-center gap-3">
      <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-emerald-500/15 text-emerald-400">
        {icon}
      </span>
      <div className="min-w-0">
        <p className="truncate text-lg leading-tight font-extrabold text-white">
          {value}
        </p>
        <p className="truncate text-xs text-slate-400">{label}</p>
      </div>
    </div>
  )
}

export default function StatsStrip({ refreshKey = 0 }) {
  const [stats, setStats] = useState(null)

  useEffect(() => {
    let cancelled = false
    async function fetchStats() {
      try {
        const { data } = await goalService.getStats()
        if (!cancelled) setStats(data)
      } catch {
        if (!cancelled) setStats(null)
      }
    }
    fetchStats()
    return () => {
      cancelled = true
    }
  }, [refreshKey])

  if (!stats || stats.total_contributions === 0) return null

  const progress = Math.min(100, Math.max(0, Number(stats.level_progress) || 0))

  return (
    <section
      aria-label="Streak dan level menabung"
      className="rounded-2xl border border-slate-800 bg-slate-900 p-5 text-slate-100 shadow-lg"
    >
      <div className="grid gap-4 sm:grid-cols-3">
        <StatItem
          icon={<Flame className="h-5 w-5" />}
          value={`${stats.current_streak_days} hari`}
          label={`Beruntun • rekor ${stats.longest_streak_days} hari`}
        />
        <StatItem
          icon={<Medal className="h-5 w-5" />}
          value={stats.level}
          label={
            stats.next_level_at != null
              ? `${stats.total_contributions}/${stats.next_level_at} ke level berikut`
              : 'Level maksimal!'
          }
        />
        <StatItem
          icon={<PiggyBank className="h-5 w-5" />}
          value={`${stats.total_contributions}x`}
          label={`Total setoran • ${stats.active_days_365} hari aktif`}
        />
      </div>
      {stats.next_level_at != null && (
        <div
          className="mt-4 h-2 overflow-hidden rounded-full bg-slate-800"
          role="progressbar"
          aria-valuenow={Math.round(progress)}
          aria-valuemin={0}
          aria-valuemax={100}
        >
          <div
            className="h-full rounded-full bg-gradient-to-r from-amber-500 to-orange-400 transition-all"
            style={{ width: `${progress}%` }}
          />
        </div>
      )}
    </section>
  )
}
