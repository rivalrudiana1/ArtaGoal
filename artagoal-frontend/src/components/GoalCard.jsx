import { useState } from 'react'
import {
  CalendarDays,
  History,
  Loader2,
  Pencil,
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

const EDIT_CATEGORIES = [
  'Dana Darurat',
  'Pendidikan',
  'Investasi',
  'Gadget',
  'Kendaraan',
  'Pensiun',
  'Rumah',
  'Liburan',
  'Modal Usaha',
  'Lainnya',
]

const EDIT_STATUSES = ['active', 'cancelled']

function toDateInput(value) {
  if (!value) return ''
  const t = new Date(value).getTime()
  if (Number.isNaN(t)) return ''
  return new Date(t).toISOString().slice(0, 10)
}

const editInputClass =
  'w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100'

export default function GoalCard({ goal, onContributed, onDeleted }) {
  const [modalOpen, setModalOpen] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [amount, setAmount] = useState('')
  const [note, setNote] = useState('')
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [editSaving, setEditSaving] = useState(false)
  const [editTitle, setEditTitle] = useState('')
  const [editCategory, setEditCategory] = useState('')
  const [editTarget, setEditTarget] = useState('')
  const [editDate, setEditDate] = useState('')
  const [editInflation, setEditInflation] = useState('')
  const [editStatus, setEditStatus] = useState('active')

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

  function openEditModal() {
    setEditTitle(goal.title ?? '')
    setEditCategory(goal.category ?? '')
    setEditTarget(String(goal.target_amount ?? ''))
    setEditDate(toDateInput(goal.target_date))
    setEditInflation(String(goal.expected_inflation_rate ?? ''))
    setEditStatus(goal.status ?? 'active')
    setEditOpen(true)
  }

  async function handleEdit(event) {
    event.preventDefault()
    const target = Number(editTarget)
    if (!editTitle.trim()) {
      toast.error('Nama target wajib diisi.')
      return
    }
    if (!Number.isFinite(target) || target <= 0) {
      toast.error('Target dana harus lebih dari 0.')
      return
    }
    setEditSaving(true)
    try {
      const { data } = await goalService.updateGoal(goal.id, {
        title: editTitle.trim(),
        category: editCategory,
        target_amount: target,
        target_date: editDate || undefined,
        expected_inflation_rate: Number(editInflation) || 0,
        status: editStatus,
      })
      onContributed(data)
      setEditOpen(false)
      toast.success('Target berhasil diperbarui!')
    } catch (err) {
      toast.error(apiErrorMessage(err, 'Gagal memperbarui target.'))
    } finally {
      setEditSaving(false)
    }
  }

  return (
    <article className="flex flex-col rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:shadow-md dark:border-slate-800 dark:bg-slate-900">
      <div className="flex items-start justify-between gap-2">
        <div>
          <h3 className="font-bold text-slate-900 dark:text-white">{goal.title}</h3>
          {goal.category && (
            <p className="mt-0.5 text-xs font-medium uppercase tracking-wide text-slate-400 dark:text-slate-500">
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
          <span className="font-bold text-emerald-700 dark:text-emerald-400">
            {formatIDR(goal.current_amount)}
          </span>
          <span className="text-slate-500 dark:text-slate-400">dari {formatIDR(goal.target_amount)}</span>
        </div>
        <div className="mt-2 h-2.5 overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
          <div
            className="h-full rounded-full bg-gradient-to-r from-emerald-500 to-teal-400 transition-all"
            style={{ width: `${percent}%` }}
          />
        </div>
        <p className="mt-1.5 text-right text-xs font-semibold text-slate-600 dark:text-slate-400">
          {percent.toFixed(1)}%
        </p>
      </div>

      <dl className="mt-3 space-y-1.5 border-t border-slate-100 pt-3 text-sm dark:border-slate-800">
        <div className="flex items-center justify-between">
          <dt className="flex items-center gap-1.5 text-slate-500 dark:text-slate-400">
            <TrendingUp className="h-4 w-4" /> Estimasi akhir (inflasi)
          </dt>
          <dd className="font-semibold text-slate-800 dark:text-slate-100">
            {futureTarget ? formatIDR(futureTarget) : '-'}
          </dd>
        </div>
        <div className="flex items-center justify-between">
          <dt className="flex items-center gap-1.5 text-slate-500 dark:text-slate-400">
            <CalendarDays className="h-4 w-4" /> Target tanggal
          </dt>
          <dd className="font-semibold text-slate-800 dark:text-slate-100">
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
          className="flex flex-1 items-center justify-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-sm font-semibold text-slate-600 transition hover:border-indigo-200 hover:text-indigo-700 dark:border-slate-700 dark:text-slate-300 dark:hover:border-indigo-500 dark:hover:text-indigo-400"
        >
          <History className="h-4 w-4" /> Riwayat
        </button>
        <button
          type="button"
          onClick={openEditModal}
          className="flex items-center justify-center rounded-lg border border-slate-200 px-3 py-2 text-slate-500 transition hover:border-emerald-200 hover:text-emerald-600 dark:border-slate-700 dark:text-slate-400 dark:hover:border-emerald-500 dark:hover:text-emerald-400"
          aria-label={`Ubah ${goal.title}`}
        >
          <Pencil className="h-4 w-4" />
        </button>
        <button
          type="button"
          onClick={() => setDeleteOpen(true)}
          disabled={deleting}
          className="flex items-center justify-center rounded-lg border border-slate-200 px-3 py-2 text-slate-500 transition hover:border-red-200 hover:text-red-600 disabled:opacity-50 dark:border-slate-700 dark:text-slate-400 dark:hover:border-red-500 dark:hover:text-red-400"
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

      {editOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 dark:bg-black/60">
          <div className="max-h-[90vh] w-full max-w-sm overflow-y-auto rounded-2xl bg-white p-6 shadow-2xl dark:bg-slate-900">
            <div className="flex items-center justify-between">
              <h4 className="font-bold text-slate-900 dark:text-white">Ubah target</h4>
              <button
                type="button"
                onClick={() => setEditOpen(false)}
                className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                aria-label="Tutup"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <form onSubmit={handleEdit} className="mt-4 space-y-3">
              <label className="block">
                <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                  Nama target
                </span>
                <input
                  type="text"
                  required
                  maxLength={100}
                  value={editTitle}
                  onChange={(e) => setEditTitle(e.target.value)}
                  className={editInputClass}
                />
              </label>
              <div className="grid grid-cols-2 gap-3">
                <label className="block">
                  <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                    Kategori
                  </span>
                  <select
                    value={editCategory}
                    onChange={(e) => setEditCategory(e.target.value)}
                    className={editInputClass}
                  >
                    {!EDIT_CATEGORIES.includes(editCategory) && (
                      <option value={editCategory}>{editCategory}</option>
                    )}
                    {EDIT_CATEGORIES.map((c) => (
                      <option key={c} value={c}>
                        {c}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="block">
                  <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                    Status
                  </span>
                  <select
                    value={editStatus}
                    onChange={(e) => setEditStatus(e.target.value)}
                    className={editInputClass}
                  >
                    {!EDIT_STATUSES.includes(editStatus) && (
                      <option value={editStatus}>{editStatus}</option>
                    )}
                    {EDIT_STATUSES.map((s) => (
                      <option key={s} value={s}>
                        {s}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
              <label className="block">
                <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                  Target dana (Rp)
                </span>
                <input
                  type="number"
                  min="1"
                  required
                  value={editTarget}
                  onChange={(e) => setEditTarget(e.target.value)}
                  className={editInputClass}
                />
              </label>
              <div className="grid grid-cols-2 gap-3">
                <label className="block">
                  <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                    Tenggat
                  </span>
                  <input
                    type="date"
                    value={editDate}
                    onChange={(e) => setEditDate(e.target.value)}
                    className={editInputClass}
                  />
                </label>
                <label className="block">
                  <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                    Inflasi (%/thn)
                  </span>
                  <input
                    type="number"
                    min="0"
                    step="0.1"
                    value={editInflation}
                    onChange={(e) => setEditInflation(e.target.value)}
                    className={editInputClass}
                  />
                </label>
              </div>
              <p className="text-xs text-slate-400">
                Saldo terkumpul tidak bisa diubah manual — hanya lewat setoran.
              </p>
              <button
                type="submit"
                disabled={editSaving}
                className="flex w-full items-center justify-center gap-2 rounded-lg bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-emerald-700 disabled:opacity-60"
              >
                {editSaving && <Loader2 className="h-4 w-4 animate-spin" />}
                Simpan perubahan
              </button>
            </form>
          </div>
        </div>
      )}

      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 dark:bg-black/60">
          <div className="w-full max-w-sm rounded-2xl bg-white p-6 shadow-2xl dark:bg-slate-900">
            <div className="flex items-center justify-between">
              <h4 className="font-bold text-slate-900 dark:text-white">
                Setor ke “{goal.title}”
              </h4>
              <button
                type="button"
                onClick={() => setModalOpen(false)}
                className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                aria-label="Tutup"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <form onSubmit={handleContribute} className="mt-4 space-y-3">
              <label className="block">
                <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
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
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                />
              </label>
              <label className="block">
                <span className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                  Catatan (opsional)
                </span>
                <input
                  type="text"
                  maxLength={500}
                  placeholder="Gaji bulan ini"
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
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
