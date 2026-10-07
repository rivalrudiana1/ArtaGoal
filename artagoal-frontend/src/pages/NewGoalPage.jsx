import { useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, Calculator, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { apiErrorMessage, goalService } from '../services/api.js'
import { SubmitButton } from '../components/AuthLayout.jsx'
import { formatIDR } from '../utils/format.js'

const CATEGORIES = [
  'Dana Darurat',
  'Pendidikan',
  'Pensiun',
  'Rumah',
  'Kendaraan',
  'Liburan',
  'Modal Usaha',
  'Lainnya',
]

const inputClass =
  'w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100'

export default function NewGoalPage() {
  const navigate = useNavigate()
  const [title, setTitle] = useState('')
  const [category, setCategory] = useState(CATEGORIES[0])
  const [targetAmount, setTargetAmount] = useState('')
  const [currentAmount, setCurrentAmount] = useState('')
  const [targetDate, setTargetDate] = useState('')
  const [inflation, setInflation] = useState('3')
  const [loading, setLoading] = useState(false)

  const simulation = useMemo(() => {
    const pv = Number(targetAmount)
    const rate = Number(inflation)
    if (!Number.isFinite(pv) || pv <= 0) {
      return { pv: 0, rate: 0, years: 0, fv: 0, diff: 0, valid: false }
    }
    const safeRate = Number.isFinite(rate)
      ? Math.min(Math.max(rate, 0), 15)
      : 0
    let years = 0
    if (targetDate) {
      const end = new Date(targetDate).getTime()
      // eslint-disable-next-line react/purity -- simulasi butuh acuan waktu sekarang saat render
      const now = Date.now()
      if (Number.isFinite(end) && end > now) {
        years = (end - now) / (1000 * 60 * 60 * 24 * 365)
      }
    }
    const fv = pv * (1 + safeRate / 100) ** years
    return { pv, rate: safeRate, years, fv, diff: fv - pv, valid: true }
  }, [targetAmount, inflation, targetDate])

  async function handleSubmit(event) {
    event.preventDefault()
    const target = Number(targetAmount)
    if (!Number.isFinite(target) || target <= 0) {
      toast.error('Target dana harus lebih dari 0.')
      return
    }
    setLoading(true)
    try {
      await goalService.createGoal({
        title: title.trim(),
        category,
        target_amount: target,
        current_amount: Number(currentAmount) || 0,
        target_date: targetDate || undefined,
        expected_inflation_rate: Number(inflation) || 0,
      })
      toast.success('Target baru berhasil disimpan! Mari konsisten menabung.')
      navigate('/dashboard', { replace: true })
    } catch (err) {
      toast.error(apiErrorMessage(err, 'Gagal membuat target.'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
      <main className="mx-auto max-w-xl px-4 py-10">
        <Link
          to="/dashboard"
          className="inline-flex items-center gap-1.5 text-sm font-medium text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-100"
        >
          <ArrowLeft className="h-4 w-4" /> Kembali ke dashboard
        </Link>
        <div className="mt-4 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm sm:p-8 dark:border-slate-800 dark:bg-slate-900">
          <h1 className="text-xl font-extrabold tracking-tight text-slate-900 dark:text-white">
            Target Baru
          </h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            Tentukan impian finansialmu beserta estimasi inflasinya.
          </p>
          <form onSubmit={handleSubmit} className="mt-6 space-y-4">
            <label className="block">
              <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                Nama target
              </span>
              <input
                type="text"
                required
                maxLength={100}
                placeholder="DP Rumah"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className={inputClass}
              />
            </label>
            <div className="grid gap-4 sm:grid-cols-2">
              <label className="block">
                <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                  Kategori
                </span>
                <select
                  value={category}
                  onChange={(e) => setCategory(e.target.value)}
                  className={inputClass}
                >
                  {CATEGORIES.map((c) => (
                    <option key={c} value={c}>
                      {c}
                    </option>
                  ))}
                </select>
              </label>
              <label className="block">
                <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                  Target tanggal (opsional)
                </span>
                <input
                  type="date"
                  value={targetDate}
                  onChange={(e) => setTargetDate(e.target.value)}
                  className={inputClass}
                />
              </label>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <label className="block">
                <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                  Target dana (Rp)
                </span>
                <input
                  type="number"
                  min="1"
                  required
                  placeholder="100000000"
                  value={targetAmount}
                  onChange={(e) => setTargetAmount(e.target.value)}
                  className={inputClass}
                />
              </label>
              <label className="block">
                <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                  Tabungan awal (Rp)
                </span>
                <input
                  type="number"
                  min="0"
                  placeholder="0"
                  value={currentAmount}
                  onChange={(e) => setCurrentAmount(e.target.value)}
                  className={inputClass}
                />
              </label>
            </div>
            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60">
              <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                Estimasi inflasi{' '}
                <span className="font-bold text-indigo-700 dark:text-indigo-400">
                  {Number(inflation) || 0}% per tahun
                </span>
              </span>
              <input
                type="range"
                min="0"
                max="15"
                step="0.5"
                value={Math.min(Math.max(Number(inflation) || 0, 0), 15)}
                onChange={(e) => setInflation(e.target.value)}
                className="w-full accent-emerald-600"
                aria-label="Geser untuk mengatur estimasi inflasi tahunan"
              />
              <div className="flex justify-between text-xs text-slate-400">
                <span>0%</span>
                <span>7,5%</span>
                <span>15%</span>
              </div>
              <div className="mt-2 grid grid-cols-2 gap-2">
                <label className="block">
                  <span className="sr-only">
                    Estimasi inflasi angka (% per tahun)
                  </span>
                  <input
                    type="number"
                    min="0"
                    max="15"
                    step="0.1"
                    value={inflation}
                    onChange={(e) => setInflation(e.target.value)}
                    className={inputClass}
                  />
                </label>
                <p className="flex items-center text-xs text-slate-500 dark:text-slate-400">
                  Geser slider atau ketik manual (0–15%).
                </p>
              </div>
            </div>
            <div className="rounded-xl border border-indigo-100 bg-indigo-50/70 p-4 dark:border-indigo-500/20 dark:bg-indigo-500/10">
              <p className="flex items-center gap-1.5 text-sm font-bold text-indigo-900 dark:text-indigo-300">
                <Calculator className="h-4 w-4" /> Simulasi Proyeksi Real-time
              </p>
              {simulation.valid ? (
                <dl className="mt-2.5 space-y-1.5 text-sm">
                  <div className="flex items-center justify-between">
                    <dt className="text-slate-600 dark:text-slate-400">Target awal (PV)</dt>
                    <dd className="font-semibold text-slate-900 dark:text-white">
                      {formatIDR(simulation.pv)}
                    </dd>
                  </div>
                  <div className="flex items-center justify-between">
                    <dt className="text-slate-600 dark:text-slate-400">
                      Future Value ({simulation.rate}% ×{' '}
                      {simulation.years.toFixed(1)} thn)
                    </dt>
                    <dd className="font-extrabold text-indigo-700 dark:text-indigo-400">
                      {formatIDR(simulation.fv)}
                    </dd>
                  </div>
                  <div className="flex items-center justify-between border-t border-indigo-100 pt-1.5 dark:border-indigo-500/20">
                    <dt className="text-slate-600 dark:text-slate-400">Selisih akibat inflasi</dt>
                    <dd className="font-semibold text-amber-700 dark:text-amber-400">
                      +{formatIDR(simulation.diff)}
                    </dd>
                  </div>
                </dl>
              ) : (
                <p className="mt-2 text-sm text-slate-500 dark:text-slate-400">
                  Isi nominal target untuk melihat estimasi nilai masa depan.
                </p>
              )}
              <p className="mt-2.5 border-t border-indigo-100 pt-2 text-xs leading-relaxed text-slate-500 dark:border-indigo-500/20 dark:text-slate-400">
                Simulasi frontend: FV = PV × (1 + i/100)
                <sup>tahun</sup>.{!targetDate
                  ? ' Pilih tanggal target agar durasi tahun terhitung akurat.'
                  : ' Nilai ini estimasi kasar sebelum disimpan — angka resmi dihitung backend saat target dibuat.'}
              </p>
            </div>
            <SubmitButton loading={loading}>
              {loading && <Loader2 className="h-4 w-4 animate-spin" />}
              Simpan Target
            </SubmitButton>
          </form>
        </div>
      </main>
    </div>
  )
}
