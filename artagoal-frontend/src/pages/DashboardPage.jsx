import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Download,
  Loader2,
  LogOut,
  PiggyBank,
  Plus,
  Search,
  SearchX,
  Target,
  Wallet,
} from 'lucide-react'
import GoalCard from '../components/GoalCard.jsx'
import GoalProjectionChart from '../components/GoalProjectionChart.jsx'
import { useAuth } from '../context/AuthContext.jsx'
import { apiErrorMessage, goalService } from '../services/api.js'
import { formatIDR, formatPercent, greeting } from '../utils/format.js'
import { exportGoalsToCsv } from '../utils/exportCsv.js'

const CATEGORY_OPTIONS = [
  'Semua',
  'Dana Darurat',
  'Pendidikan',
  'Investasi',
  'Gadget',
  'Kendaraan',
  'Lainnya',
]

const STATUS_OPTIONS = [
  { value: 'all', label: 'Semua' },
  { value: 'active', label: 'ACTIVE' },
  { value: 'achieved', label: 'ACHIEVED' },
]

const SORT_OPTIONS = [
  { value: 'deadline', label: 'Tenggat Waktu Terdekat' },
  { value: 'progress', label: 'Progres Terbanyak' },
  { value: 'nominal', label: 'Target Nominal Terbesar' },
  { value: 'newest', label: 'Terbaru Dibuat' },
]

function progressRatio(goal) {
  const target = Number(goal?.target_amount) || 0
  if (target <= 0) return 0
  return Number(goal?.current_amount || 0) / target
}

function deadlineTime(goal) {
  if (!goal?.target_date) return Number.POSITIVE_INFINITY
  const t = new Date(goal.target_date).getTime()
  return Number.isNaN(t) ? Number.POSITIVE_INFINITY : t
}

function createdTime(goal) {
  if (!goal?.created_at) return 0
  const t = new Date(goal.created_at).getTime()
  return Number.isNaN(t) ? 0 : t
}

function SummaryCard({ icon, label, value, accent }) {
  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex items-center gap-2 text-sm text-slate-500">
        {icon}
        <span>{label}</span>
      </div>
      <p className={`mt-2 text-2xl font-extrabold tracking-tight ${accent}`}>
        {value}
      </p>
    </div>
  )
}

const selectClass =
  'w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 outline-none transition focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100'

export default function DashboardPage() {
  const { user, logout } = useAuth()
  const [goals, setGoals] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [categoryFilter, setCategoryFilter] = useState('Semua')
  const [statusFilter, setStatusFilter] = useState('all')
  const [sortBy, setSortBy] = useState('deadline')

  const fetchGoals = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const { data } = await goalService.listMyGoals()
      setGoals(Array.isArray(data) ? data : [])
    } catch (err) {
      setError(apiErrorMessage(err, 'Gagal memuat daftar target.'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    // eslint-disable-next-line react/set-state-in-effect -- fetch awal data dashboard
    fetchGoals()
  }, [fetchGoals])

  const summary = useMemo(() => {
    const totalTarget = goals.reduce((s, g) => s + Number(g.target_amount || 0), 0)
    const totalSaved = goals.reduce((s, g) => s + Number(g.current_amount || 0), 0)
    return { totalTarget, totalSaved, percent: formatPercent(totalSaved, totalTarget) }
  }, [goals])

  const filteredGoals = useMemo(() => {
    const keyword = search.trim().toLowerCase()
    const filtered = goals.filter((goal) => {
      if (
        keyword &&
        !String(goal?.title ?? '').toLowerCase().includes(keyword)
      ) {
        return false
      }
      if (categoryFilter !== 'Semua' && goal?.category !== categoryFilter) {
        return false
      }
      if (
        statusFilter !== 'all' &&
        String(goal?.status ?? '').toLowerCase() !== statusFilter
      ) {
        return false
      }
      return true
    })
    const sorted = [...filtered]
    switch (sortBy) {
      case 'progress':
        sorted.sort((a, b) => progressRatio(b) - progressRatio(a))
        break
      case 'nominal':
        sorted.sort(
          (a, b) => Number(b.target_amount || 0) - Number(a.target_amount || 0),
        )
        break
      case 'newest':
        sorted.sort((a, b) => createdTime(b) - createdTime(a))
        break
      case 'deadline':
      default:
        sorted.sort((a, b) => deadlineTime(a) - deadlineTime(b))
        break
    }
    return sorted
  }, [goals, search, categoryFilter, statusFilter, sortBy])

  const hasActiveFilter =
    search.trim() !== '' || categoryFilter !== 'Semua' || statusFilter !== 'all'

  function resetFilters() {
    setSearch('')
    setCategoryFilter('Semua')
    setStatusFilter('all')
    setSortBy('deadline')
  }

  function handleContributed(updatedGoal) {
    setGoals((prev) => prev.map((g) => (g.id === updatedGoal.id ? updatedGoal : g)))
  }

  function handleDeleted(id) {
    setGoals((prev) => prev.filter((g) => g.id !== id))
  }

  function handleExport() {
    exportGoalsToCsv(goals, 'artagoal-laporan.csv')
  }

  return (
    <div className="min-h-screen">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-4">
          <div className="flex items-center gap-2">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-600 text-white">
              <PiggyBank className="h-5 w-5" />
            </span>
            <span className="text-lg font-extrabold tracking-tight text-slate-900">
              ArtaGoal
            </span>
          </div>
          <div className="flex items-center gap-3">
            <p className="hidden text-sm text-slate-600 sm:block">
              {greeting()},{' '}
              <span className="font-semibold text-slate-900">
                {user?.name ?? 'Sobat'}
              </span>
            </p>
            <button
              type="button"
              onClick={logout}
              className="flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-600 transition hover:border-red-200 hover:text-red-600"
            >
              <LogOut className="h-4 w-4" /> Keluar
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl space-y-6 px-4 py-8">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h1 className="text-2xl font-extrabold tracking-tight text-slate-900">
              {greeting()}, {user?.name ?? 'Sobat'}!
            </h1>
            <p className="mt-1 text-sm text-slate-500">
              Pantau progres menuju semua target keuanganmu.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={handleExport}
              disabled={loading || goals.length === 0}
              className="flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:border-emerald-300 hover:text-emerald-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <Download className="h-4 w-4" /> Export Laporan (.csv)
            </button>
            <Link
              to="/goals/new"
              className="flex items-center gap-1.5 rounded-lg bg-emerald-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-emerald-700"
            >
              <Plus className="h-4 w-4" /> Target Baru
            </Link>
          </div>
        </div>

        <section className="grid gap-4 sm:grid-cols-3">
          <SummaryCard
            icon={<Target className="h-4 w-4" />}
            label="Total target dana"
            value={formatIDR(summary.totalTarget)}
            accent="text-slate-900"
          />
          <SummaryCard
            icon={<Wallet className="h-4 w-4" />}
            label="Total tabungan terkumpul"
            value={formatIDR(summary.totalSaved)}
            accent="text-emerald-700"
          />
          <SummaryCard
            icon={<PiggyBank className="h-4 w-4" />}
            label={`${goals.length} target • ${summary.percent.toFixed(1)}% tercapai`}
            value={formatIDR(Math.max(summary.totalTarget - summary.totalSaved, 0))}
            accent="text-slate-900"
          />
        </section>

        <GoalProjectionChart goals={goals} />

        <section
          aria-label="Pencarian dan filter target"
          className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm"
        >
          <div className="grid gap-3 lg:grid-cols-[1.4fr_1fr_1fr_1fr]">
            <label className="relative block">
              <span className="sr-only">Cari target berdasarkan judul</span>
              <Search className="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Cari target berdasarkan judul..."
                className="w-full rounded-lg border border-slate-300 bg-white py-2 pr-3 pl-9 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100"
              />
            </label>
            <label className="block">
              <span className="sr-only">Filter kategori</span>
              <select
                value={categoryFilter}
                onChange={(e) => setCategoryFilter(e.target.value)}
                className={selectClass}
              >
                {CATEGORY_OPTIONS.map((option) => (
                  <option key={option} value={option}>
                    {option === 'Semua' ? 'Semua Kategori' : option}
                  </option>
                ))}
              </select>
            </label>
            <label className="block">
              <span className="sr-only">Filter status</span>
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className={selectClass}
              >
                {STATUS_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.value === 'all'
                      ? 'Semua Status'
                      : `Status: ${option.label}`}
                  </option>
                ))}
              </select>
            </label>
            <label className="block">
              <span className="sr-only">Urutkan target</span>
              <select
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value)}
                className={selectClass}
              >
                {SORT_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="mt-2.5 flex flex-wrap items-center justify-between gap-2 text-xs text-slate-500">
            <p>
              Menampilkan{' '}
              <span className="font-bold text-slate-800">
                {filteredGoals.length}
              </span>{' '}
              dari {goals.length} target
              {hasActiveFilter ? ' sesuai kriteria.' : '.'}
            </p>
            {hasActiveFilter && (
              <button
                type="button"
                onClick={resetFilters}
                className="font-semibold text-emerald-700 hover:text-emerald-800 hover:underline"
              >
                Reset filter
              </button>
            )}
          </div>
        </section>

        {loading ? (
          <div className="flex items-center justify-center gap-2 py-16 text-slate-500">
            <Loader2 className="h-5 w-5 animate-spin" /> Memuat target...
          </div>
        ) : error ? (
          <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
            <p className="text-sm text-red-700">{error}</p>
            <button
              type="button"
              onClick={fetchGoals}
              className="mt-3 rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white hover:bg-red-700"
            >
              Coba lagi
            </button>
          </div>
        ) : goals.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-slate-300 bg-white p-12 text-center">
            <Target className="mx-auto h-10 w-10 text-slate-300" />
            <p className="mt-3 font-semibold text-slate-800">Belum ada target</p>
            <p className="mt-1 text-sm text-slate-500">
              Buat target pertamamu dan mulai menabung dengan konsisten.
            </p>
            <Link
              to="/goals/new"
              className="mt-4 inline-flex items-center gap-1.5 rounded-lg bg-emerald-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-emerald-700"
            >
              <Plus className="h-4 w-4" /> Buat Target
            </Link>
          </div>
        ) : filteredGoals.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-slate-300 bg-white p-12 text-center">
            <SearchX className="mx-auto h-10 w-10 text-slate-300" />
            <p className="mt-3 font-semibold text-slate-800">
              Target tidak ditemukan
            </p>
            <p className="mx-auto mt-1 max-w-md text-sm text-slate-500">
              Tidak ada target yang cocok dengan pencarian
              {search.trim() ? ` “${search.trim()}”` : ''} dan kombinasi
              filter saat ini. Coba kata kunci lain atau atur ulang filter.
            </p>
            <button
              type="button"
              onClick={resetFilters}
              className="mt-4 inline-flex items-center gap-1.5 rounded-lg bg-emerald-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-emerald-700"
            >
              Reset pencarian & filter
            </button>
          </div>
        ) : (
          <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {filteredGoals.map((goal) => (
              <GoalCard
                key={goal.id}
                goal={goal}
                onContributed={handleContributed}
                onDeleted={handleDeleted}
              />
            ))}
          </section>
        )}
      </main>
    </div>
  )
}
