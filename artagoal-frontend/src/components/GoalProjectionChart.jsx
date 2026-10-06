import { useMemo } from 'react'
import { Target, TrendingUp } from 'lucide-react'
import {
  Area,
  CartesianGrid,
  ComposedChart,
  Legend,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { formatIDR } from '../utils/format.js'

function toNumber(value, fallback = 0) {
  const n = Number(value)
  return Number.isFinite(n) ? n : fallback
}

/** Fallback proyeksi di sisi klien jika backend belum menyertakan `projection`. */
function resolveFutureTarget(goal) {
  const fromApi = goal?.projection?.future_target_amount
  if (Number.isFinite(Number(fromApi)) && Number(fromApi) > 0) {
    return Number(fromApi)
  }
  const target = toNumber(goal?.target_amount)
  const rate = toNumber(goal?.expected_inflation_rate)
  if (goal?.target_date && rate > 0 && target > 0) {
    const now = new Date()
    const end = new Date(goal.target_date)
    const days = (end.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)
    if (Number.isFinite(days) && days > 0) {
      const years = days / 365
      return target * (1 + rate / 100) ** years
    }
  }
  return target
}

function truncateTitle(value, max = 12) {
  const text = String(value ?? '-')
  return text.length > max ? `${text.slice(0, max)}…` : text
}

/** Format ringkas untuk sumbu Y: 2,5 jt / 1,2 M agar muat di layar kecil. */
function formatCompactIDR(value) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '0'
  if (Math.abs(n) >= 1_000_000_000) {
    const v = n / 1_000_000_000
    return `${v.toLocaleString('id-ID', { maximumFractionDigits: 1 })} M`
  }
  if (Math.abs(n) >= 1_000_000) {
    const v = n / 1_000_000
    return `${v.toLocaleString('id-ID', { maximumFractionDigits: 1 })} jt`
  }
  if (Math.abs(n) >= 1_000) {
    const v = n / 1_000
    return `${v.toLocaleString('id-ID', { maximumFractionDigits: 0 })} rb`
  }
  return String(Math.round(n))
}

function ChartTooltip({ active, payload, label }) {
  if (!active || !payload || payload.length === 0) return null
  const fullTitle = payload[0]?.payload?.fullTitle ?? label
  return (
    <div className="rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm shadow-lg">
      <p className="mb-1.5 max-w-56 truncate font-bold text-slate-900">
        {fullTitle}
      </p>
      <div className="space-y-1">
        {payload.map((entry) => (
          <p
            key={entry.dataKey}
            className="flex items-center gap-2 text-slate-600"
          >
            <span
              className="h-2.5 w-2.5 rounded-full"
              style={{ backgroundColor: entry.color ?? entry.stroke }}
            />
            <span className="min-w-28">{entry.name}</span>
            <span className="font-semibold text-slate-900">
              {formatIDR(entry.value)}
            </span>
          </p>
        ))}
      </div>
    </div>
  )
}

export default function GoalProjectionChart({ goals = [] }) {
  const chartData = useMemo(
    () =>
      (Array.isArray(goals) ? goals : []).map((goal) => ({
        title: truncateTitle(goal?.title),
        fullTitle: goal?.title ?? '-',
        Terkumpul: toNumber(goal?.current_amount),
        'Proyeksi Inflasi': Math.round(resolveFutureTarget(goal)),
      })),
    [goals],
  )

  if (!Array.isArray(goals) || goals.length === 0) {
    return (
      <section
        aria-label="Grafik proyeksi target"
        className="rounded-2xl border border-dashed border-slate-300 bg-white p-8 text-center shadow-sm"
      >
        <Target className="mx-auto h-10 w-10 text-slate-300" />
        <p className="mt-3 font-semibold text-slate-800">
          Belum ada data proyeksi
        </p>
        <p className="mx-auto mt-1 max-w-md text-sm text-slate-500">
          Buat target keuangan pertamamu untuk melihat perbandingan dana
          terkumpul dan nilai proyeksi terinflasi di grafik ini.
        </p>
      </section>
    )
  }

  return (
    <section
      aria-label="Grafik proyeksi target"
      className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
    >
      <div className="mb-4 flex items-start justify-between gap-3">
        <div className="flex items-center gap-2.5">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-indigo-100 text-indigo-700">
            <TrendingUp className="h-5 w-5" />
          </span>
          <div>
            <h2 className="font-bold tracking-tight text-slate-900">
              Proyeksi vs Terkumpul
            </h2>
            <p className="text-xs text-slate-500">
              Perbandingan dana terkumpul dan nilai target terinflasi per
              target.
            </p>
          </div>
        </div>
        <span className="hidden rounded-full bg-emerald-50 px-3 py-1 text-xs font-semibold text-emerald-700 sm:block">
          {goals.length} target
        </span>
      </div>

      <div className="h-72 w-full sm:h-80">
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart
            data={chartData}
            margin={{ top: 8, right: 8, bottom: 0, left: 0 }}
          >
            <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
            <XAxis
              dataKey="title"
              tick={{ fontSize: 12, fill: '#64748b' }}
              tickLine={false}
              axisLine={{ stroke: '#e2e8f0' }}
              minTickGap={8}
            />
            <YAxis
              tick={{ fontSize: 12, fill: '#64748b' }}
              tickLine={false}
              axisLine={false}
              width={64}
              tickFormatter={formatCompactIDR}
            />
            <Tooltip content={<ChartTooltip />} />
            <Legend
              wrapperStyle={{ fontSize: 12 }}
              iconType="circle"
              iconSize={8}
            />
            <Area
              type="monotone"
              dataKey="Terkumpul"
              stroke="#059669"
              strokeWidth={2.5}
              fill="#6ee7b7"
              fillOpacity={0.35}
              dot={{ r: 3, fill: '#059669', strokeWidth: 0 }}
              activeDot={{ r: 5 }}
              name="Terkumpul"
            />
            <Line
              type="monotone"
              dataKey="Proyeksi Inflasi"
              stroke="#d97706"
              strokeWidth={2.5}
              strokeDasharray="7 4"
              dot={{ r: 3, fill: '#d97706', strokeWidth: 0 }}
              activeDot={{ r: 5 }}
              name="Proyeksi Inflasi"
            />
          </ComposedChart>
        </ResponsiveContainer>
      </div>

      <p className="mt-3 border-t border-slate-100 pt-3 text-xs text-slate-400">
        Nilai proyeksi memakai kalkulasi anti-inflasi backend
        (FV&nbsp;=&nbsp;PV&nbsp;×&nbsp;(1&nbsp;+&nbsp;i)ⁿ). Grafik ter-update
        otomatis setiap tambah target / setor tabungan.
      </p>
    </section>
  )
}
