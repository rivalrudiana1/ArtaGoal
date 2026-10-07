import { useState } from 'react'
import {
  CalendarDays,
  History,
  Loader2,
  Plus,
  Trash2,
  TrendingUp,
  Trophy,
  X,
} from 'lucide-react'
import { toast } from 'sonner'
import { apiErrorMessage, goalService } from '../services/api.js'
import { formatDate, formatIDR, formatPercent } from '../utils/format.js'
import ContributionHistoryModal from './ContributionHistoryModal.jsx'
import ConfirmModal from './ui/ConfirmModal.jsx'

const STATUS_STYLE = {
  active: 'bg-emerald-100 text-emerald-800',
  achieved: 'bg-amber-100 text-amber-800',
  cancelled: 'bg-slate-200 text-slate-600',
}

export default function GoalCard({ goal, onContributed, onDeleted }) {
  const [modalOpen, setModalOpen] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [amount, setAmount] = useState('')
  const [note, setNote] = useState('')
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const percent = formatPercent(goal.current_amount, goal.target_amount)
  const futureTarget = goal.projection?.future_target_amount
  const monthsLeft = goal.projection?.months_remaining

  async function handleContribute(event) {
    event.preventDefault()
    const value = Number(amount)
    if (!Number.isFinite(value) || value <= 0) {
      toast.error('Nominal setoran harus lebih dari 0.')
      return
    }
    setSaving(true)
    try {
      const { data } = await goalService.addContribution(goal.id, {
        amount: value,
        note: note.trim(),
      })
      onContributed(data.goal)
      setModalOpen(false)
      setAmount('')
      setNote('')
      toast.success('Setoran berhasil ditambahkan! Progres Anda meningkat 🎉')
    } catch (err) {
      toast.error(apiErrorMessage(err, 'Gagal menyimpan setoran.'))
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete() {
    setDeleting(true)
    try {
      await goalService.deleteGoal(goal.id)
      setDeleteOpen(false)
      toast.success('Target keuangan berhasil dihapus!')
      onDeleted(goal.id)
    } catch (err) {
      toast.error(apiErrorMessage(err, 'Gagal menghapus target.'))
    } finally {
      setDeleting(false)
    }
  }

  return (
    <article className="flex flex-col rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:shadow-md">
      <div className="flex items-start justify-between gap-2">
        <div>
          <h3 className="font-bold text-slate-900">{goal.title}</h3>
          {goal.category && (
            <p className="mt-0.5 text-xs font-medium uppercase tracking-wide text-slate-400">
              {goal.category}
            </p>
          )}
        </div>
        <div className="flex items-center gap-1.5">
          {goal.status === 'achieved' && (
            <Trophy className="h-4 w-4 text-amber-500" aria-label="Tercapai" />
          )}
          <span
            className={`rounded-full px-2.5 py-0.5 text-xs font-semibold capitalize ${STATUS_STYLE[goal.status] ?? STATUS_STYLE.active}`}
          >
            {goal.status}
          </span>
        </div>
      </div>

      <div className="mt-4">
        <div className="flex items-end justify-between text-sm">
          <span className="font-bold text-emerald-700">
            {formatIDR(goal.current_amount)}
          </span>
          <span className="text-slate-500">dari {formatIDR(goal.target_amount)}</span>
        </div>
        <div className="mt-2 h-2.5 overflow-hidden rounded-full bg-slate-100">
          <div
            className="h-full rounded-full bg-gradient-to-r from-emerald-500 to-teal-400 transition-all"
            style={{ width: `${percent}%` }}
          />
        </div>
        <p className="mt-1.5 text-right text-xs font-semibold text-slate-600">
          {percent.toFixed(1)}%
        </p>
      </div>

      <dl className="mt-3 space-y-1.5 border-t border-slate-100 pt-3 text-sm">
        <div className="flex items-center justify-between">
          <dt className="flex items-center gap-1.5 text-slate-500">
            <TrendingUp className="h-4 w-4" /> Estimasi akhir (inflasi)
          </dt>
          <dd className="font-semibold text-slate-800">
            {futureTarget ? formatIDR(futureTarget) : '-'}
          </dd>
        </div>
        <div className="flex items-center justify-between">
          <dt className="flex items-center gap-1.5 text-slate-500">
            <CalendarDays className="h-4 w-4" /> Target tanggal
          </dt>
          <dd className="font-semibold text-slate-800">
            {formatDate(goal.target_date)}
            {monthsLeft != null && goal.target_date
              ? ` (${monthsLeft} bln)`
              : ''}
          </dd>
        </div>
      </dl>

      <div className="mt-4 flex gap-2">
        <button
          type="button"
          onClick={() => setModalOpen(true)}
          className="flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-emerald-600 px-3 py-2 text-sm font-semibold text-white transition hover:bg-emerald-700"
        >
          <Plus className="h-4 w-4" /> Setor
        </button>
        <button
          type="button"
          onClick={() => setHistoryOpen(true)}
          className="flex flex-1 items-center justify-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-sm font-semibold text-slate-600 transition hover:border-indigo-200 hover:text-indigo-700"
        >
          <History className="h-4 w-4" /> Riwayat
        </button>
        <button
          type="button"
          onClick={() => setDeleteOpen(true)}
          disabled={deleting}
          className="flex items-center justify-center rounded-lg border border-slate-200 px-3 py-2 text-slate-500 transition hover:border-red-200 hover:text-red-600 disabled:opacity-50"
          aria-label={`Hapus ${goal.title}`}
        >
          <Trash2 className="h-4 w-4" />
        </button>
      </div>

      <ConfirmModal
        isOpen={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={handleDelete}
        title={`Hapus "${goal.title}"?`}
        description="Target dan seluruh riwayat setorannya akan dihapus permanen. Tindakan ini tidak dapat dibatalkan."
        confirmText="Ya, Hapus"
        variant="danger"
        isLoading={deleting}
      />

      <ContributionHistoryModal
        goal={goal}
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
        onUpdated={onContributed}
      />

      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4">
          <div className="w-full max-w-sm rounded-2xl bg-white p-6 shadow-2xl">
            <div className="flex items-center justify-between">
              <h4 className="font-bold text-slate-900">
                Setor ke “{goal.title}”
              </h4>
              <button
                type="button"
                onClick={() => setModalOpen(false)}
                className="text-slate-400 hover:text-slate-600"
                aria-label="Tutup"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <form onSubmit={handleContribute} className="mt-4 space-y-3">
              <label className="block">
                <span className="mb-1 block text-sm font-medium text-slate-700">
                  Nominal (Rp)
                </span>
                <input
                  type="number"
                  min="1"
                  required
                  autoFocus
                  placeholder="500000"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100"
                />
              </label>
              <label className="block">
                <span className="mb-1 block text-sm font-medium text-slate-700">
                  Catatan (opsional)
                </span>
                <input
                  type="text"
                  maxLength={500}
                  placeholder="Gaji bulan ini"
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100"
                />
              </label>
              <button
                type="submit"
                disabled={saving}
                className="flex w-full items-center justify-center gap-2 rounded-lg bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-emerald-700 disabled:opacity-60"
              >
                {saving && <Loader2 className="h-4 w-4 animate-spin" />}
                Simpan setoran
              </button>
            </form>
          </div>
        </div>
      )}
    </article>
  )
}
