import { useEffect, useMemo, useState } from 'react'
import { ActivityCalendar } from 'react-activity-calendar'
import { format, subDays } from 'date-fns'
import { Flame } from 'lucide-react'
import { apiErrorMessage, goalService } from '../services/api.js'
import { formatIDR } from '../utils/format.js'

function levelForCount(count) {
  if (count <= 0) return 0
  if (count === 1) return 1
  if (count === 2) return 2
  if (count === 3) return 3
  return 4
}

function useIsMobile(breakpoint = 640) {
  const [isMobile, setIsMobile] = useState(
    () =>
      typeof window !== 'undefined' && window.innerWidth < breakpoint,
  )
  useEffect(() => {
    if (typeof window === 'undefined') return undefined
    const query = window.matchMedia(`(max-width: ${breakpoint - 1}px)`)
    const onChange = (event) => setIsMobile(event.matches)
    query.addEventListener('change', onChange)
    return () => query.removeEventListener('change', onChange)
  }, [breakpoint])
  return isMobile
}

export default function ContributionHeatmap() {
  const [raw, setRaw] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  // Tanggal "hari ini" dijepret sekali agar kalender stabil antar render.
  const [today] = useState(() => new Date())
  const isMobile = useIsMobile()

  useEffect(() => {
    let cancelled = false
    async function fetchHeatmap() {
      setLoading(true)
      setError('')
      try {
        const { data } = await goalService.getHeatmap()
        if (cancelled) return
        const list = Array.isArray(data) ? data : (data?.data ?? [])
        setRaw(list)
      } catch (err) {
        if (cancelled) return
        setError(apiErrorMessage(err, 'Gagal memuat aktivitas menabung.'))
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    fetchHeatmap()
    return () => {
      cancelled = true
    }
  }, [])

  const amountByDate = useMemo(() => {
    const map = new Map()
    for (const item of raw) {
      if (!item?.date) continue
      map.set(item.date, Number(item.total_amount ?? item.totalAmount ?? 0))
    }
    return map
  }, [raw])

  const { calendarData, totalCount, totalAmount } = useMemo(() => {
    const byDate = new Map()
    for (const item of raw) {
      if (!item?.date) continue
      byDate.set(item.date, Number(item.count ?? 0))
    }
    const days = []
    let countSum = 0
    let amountSum = 0
    for (let i = 364; i >= 0; i -= 1) {
      const date = format(subDays(today, i), 'yyyy-MM-dd')
      const count = byDate.get(date) ?? 0
      countSum += count
      amountSum += amountByDate.get(date) ?? 0
      days.push({ date, count, level: levelForCount(count) })
    }
    return { calendarData: days, totalCount: countSum, totalAmount: amountSum }
  }, [raw, amountByDate, today])

  function retry() {
    setRaw([])
    setLoading(true)
    setError('')
    goalService
      .getHeatmap()
      .then(({ data }) => {
        const list = Array.isArray(data) ? data : (data?.data ?? [])
        setRaw(list)
      })
      .catch((err) => {
        setError(apiErrorMessage(err, 'Gagal memuat aktivitas menabung.'))
      })
      .finally(() => setLoading(false))
  }

  return (
    <section
      aria-label="Aktivitas menabung"
      className="rounded-2xl border border-slate-800 bg-slate-900 p-5 text-slate-100 shadow-lg"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-500/15 text-emerald-400">
            <Flame className="h-5 w-5" />
          </span>
          <div>
            <h2 className="text-base font-extrabold tracking-tight sm:text-lg">
              Aktivitas Menabung (365 Hari Terakhir)
            </h2>
            <p className="text-xs text-slate-400">
              {totalCount} setoran • {formatIDR(totalAmount)} dalam setahun
            </p>
          </div>
        </div>
        <p className="hidden text-xs text-slate-500 sm:block">
          Semakin hijau, semakin rajin menabung
        </p>
      </div>

      <div className="mt-4 overflow-x-auto pb-1">
        {error ? (
          <div className="rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-center">
            <p className="text-sm text-red-300">{error}</p>
            <button
              type="button"
              onClick={retry}
              className="mt-2 rounded-lg bg-emerald-600 px-4 py-1.5 text-sm font-semibold text-white transition hover:bg-emerald-500"
            >
              Coba lagi
            </button>
          </div>
        ) : (
          <ActivityCalendar
            data={calendarData}
            loading={loading}
            colorScheme="dark"
            theme={{
              light: ['#ebedf0', '#9be9a8', '#40c463', '#30a14e', '#216e39'],
              dark: ['#161b22', '#0e4429', '#006d32', '#26a641', '#39d353'],
            }}
            blockSize={isMobile ? 10 : 12}
            blockMargin={3}
            blockRadius={2}
            fontSize={12}
            showMonthLabels={!isMobile}
            showWeekdayLabels={false}
            labels={{
              months: [
                'Jan',
                'Feb',
                'Mar',
                'Apr',
                'Mei',
                'Jun',
                'Jul',
                'Agu',
                'Sep',
                'Okt',
                'Nov',
                'Des',
              ],
              totalCount: '{{count}} setoran dalam setahun terakhir',
              legend: { less: 'Jarang', more: 'Rajin' },
            }}
            tooltips={{
              activity: {
                text: (activity) => {
                  const amount = amountByDate.get(activity.date) ?? 0
                  const dateLabel = new Date(
                    `${activity.date}T00:00:00`,
                  ).toLocaleDateString('id-ID', {
                    day: 'numeric',
                    month: 'short',
                    year: 'numeric',
                  })
                  if (!activity.count) return `${dateLabel}: belum ada setoran`
                  return `${dateLabel}: ${activity.count}x setoran • ${formatIDR(amount)}`
                },
              },
            }}
          />
        )}
      </div>
    </section>
  )
}
