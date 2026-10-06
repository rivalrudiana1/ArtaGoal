import { useCallback, useEffect, useState } from 'react'
import { History, Loader2, Trash2, X } from 'lucide-react'
import { apiErrorMessage, goalService } from '../services/api.js'
import { formatDate, formatIDR } from '../utils/format.js'

function formatDateTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return `${date.toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })} • ${date.toLocaleTimeString('id-ID', {
    hour: '2-digit',
    minute: '2-digit',
  })}`
}

function normalizeContributions(payload) {
  if (Array.isArray(payload)) return payload
  if (Array.isArray(payload?.data)) return payload.data
  if (Array.isArray(payload?.contributions)) return payload.contributions
  return []
}

export default function ContributionHistoryModal({
  goal,
  open,
  onClose,
  onUpdated,
}) {
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [deletingId, setDeletingId] = useState('')
  const [confirmId, setConfirmId] = useState('')

  const goalId = goal?.id

  const fetchHistory = useCallback(async () => {
    if (!goalId) return
    setLoading(true)
    setError('')
    try {
      const { data } = await goalService.listContributions(goalId)
      setItems(normalizeContributions(data))
    } catch (err) {
      setError(apiErrorMessage(err, 'Gagal memuat riwayat setoran.'))
    } finally {
      setLoading(false)
    }
  }, [goalId])

  useEffect(() => {
    if (open) {
      // eslint-disable-next-line react/set-state-in-effect -- sinkronisasi fetch saat modal dibuka
      setConfirmId('')
      setDeletingId('')
      fetchHistory()
    }
  }, [open, fetchHistory])

  useEffect(() => {
    if (!open) return
    function handleKey(event) {
      if (event.key === 'Escape') onClose?.()
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [open, onClose])

  async function handleDelete(contributionId) {
    if (!goal?.id || !contributionId || deletingId) return
    setDeletingId(contributionId)
    try {
      const { data } = await goalService.deleteContribution(
        goal.id,
        contributionId,
      )
      setItems((prev) => prev.filter((c) => c.id !== contributionId))
      setConfirmId('')
      // Backend DELETE mengembalikan GoalResponse terbaru (bukan envelope),
      // sedangkan POST addContribution mengembalikan { goal }. Tangani keduanya.
      const updatedGoal = data?.goal ?? data
      if (updatedGoal?.id && typeof onUpdated === 'function') {
        onUpdated(updatedGoal)
      }
    } catch (err) {
      alert(apiErrorMessage(err, 'Gagal menghapus transaksi.'))
    } finally {
      setDeletingId('')
    }
  }

  if (!open || !goal) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4"
      role="dialog"
      aria-modal="true"
      aria-label={`Riwayat setoran ${goal.title}`}
      onClick={onClose}
    >
      <div
        className="flex max-h-[85vh] w-full max-w-lg flex-col rounded-2xl bg-white p-6 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-2.5">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-indigo-100 text-indigo-700">
              <History className="h-5 w-5" />
            </span>
            <div>
              <h4 className="font-bold text-slate-900">
                Riwayat Setoran — {goal.title}
              </h4>
              <p className="text-xs text-slate-500">
                {items.length} transaksi • Terkumpul{' '}
                {formatIDR(goal.current_amount)}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-slate-400 transition hover:text-slate-600"
            aria-label="Tutup riwayat"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="mt-4 min-h-0 flex-1 overflow-y-auto">
          {loading ? (
            <div className="flex items-center justify-center gap-2 py-10 text-sm text-slate-500">
              <Loader2 className="h-5 w-5 animate-spin" /> Memuat riwayat...
            </div>
          ) : error ? (
            <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-center">
              <p className="text-sm text-red-700">{error}</p>
              <button
                type="button"
                onClick={fetchHistory}
                className="mt-2 rounded-lg bg-red-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-red-700"
              >
                Coba lagi
              </button>
            </div>
          ) : items.length === 0 ? (
            <div className="rounded-xl border border-dashed border-slate-300 p-8 text-center">
              <History className="mx-auto h-8 w-8 text-slate-300" />
              <p className="mt-2 text-sm font-semibold text-slate-700">
                Belum ada setoran
              </p>
              <p className="mt-1 text-xs text-slate-500">
                Setoran pertama akan tercatat di sini lengkap dengan tanggal
                dan catatan.
              </p>
            </div>
          ) : (
            <ul className="divide-y divide-slate-100 rounded-xl border border-slate-200">
              {items.map((item) => (
                <li key={item.id} className="p-3.5">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="text-xs text-slate-500">
                        {formatDate(item.created_at) !== '-'
                          ? formatDateTime(item.created_at)
                          : formatDate(item.created_at)}
                      </p>
                      <p className="mt-0.5 truncate text-sm font-bold text-emerald-700">
                        {formatIDR(item.amount)}
                      </p>
                      {item.note ? (
                        <p className="mt-0.5 truncate text-xs text-slate-500">
                          “{item.note}”
                        </p>
                      ) : (
                        <p className="mt-0.5 text-xs italic text-slate-400">
                          Tanpa catatan
                        </p>
                      )}
                    </div>
                    {confirmId === item.id ? (
                      <div className="flex shrink-0 items-center gap-1.5">
                        <button
                          type="button"
                          disabled={deletingId === item.id}
                          onClick={() => handleDelete(item.id)}
                          className="rounded-lg bg-red-600 px-2.5 py-1.5 text-xs font-semibold text-white transition hover:bg-red-700 disabled:opacity-60"
                        >
                          {deletingId === item.id ? (
                            <span className="flex items-center gap-1">
                              <Loader2 className="h-3.5 w-3.5 animate-spin" />{' '}
                              Ya
                            </span>
                          ) : (
                            'Ya, hapus'
                          )}
                        </button>
                        <button
                          type="button"
                          disabled={deletingId === item.id}
                          onClick={() => setConfirmId('')}
                          className="rounded-lg border border-slate-200 px-2.5 py-1.5 text-xs font-medium text-slate-600 hover:bg-slate-50"
                        >
                          Batal
                        </button>
                      </div>
                    ) : (
                      <button
                        type="button"
                        onClick={() => setConfirmId(item.id)}
                        className="flex shrink-0 items-center gap-1 rounded-lg border border-slate-200 px-2.5 py-1.5 text-xs font-medium text-slate-500 transition hover:border-red-200 hover:text-red-600"
                        aria-label={`Hapus setoran ${formatIDR(item.amount)}`}
                      >
                        <Trash2 className="h-3.5 w-3.5" /> Hapus
                      </button>
                    )}
                  </div>
                  {confirmId === item.id && (
                    <p className="mt-2 rounded-lg bg-red-50 px-2.5 py-1.5 text-xs text-red-700">
                      Hapus transaksi ini? Saldo target akan berkurang otomatis.
                    </p>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>

        <button
          type="button"
          onClick={onClose}
          className="mt-4 w-full rounded-lg border border-slate-200 px-4 py-2.5 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
        >
          Tutup
        </button>
      </div>
    </div>
  )
}
