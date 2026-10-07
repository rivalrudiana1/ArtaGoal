import { useCallback, useEffect, useState } from 'react'
import { History, Loader2, Trash2, X } from 'lucide-react'
import { toast } from 'sonner'
import { apiErrorMessage, goalService } from '../services/api.js'
import { formatDate, formatIDR } from '../utils/format.js'
import ConfirmModal from './ui/ConfirmModal.jsx'

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
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState('')
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [deletingId, setDeletingId] = useState('')
  const [pendingDelete, setPendingDelete] = useState(null)

  const goalId = goal?.id
  const PAGE_LIMIT = 20

  const fetchHistory = useCallback(
    async (pageToLoad = 1) => {
      if (!goalId) return
      if (pageToLoad === 1) {
        setLoading(true)
      } else {
        setLoadingMore(true)
      }
      setError('')
      try {
        const { data } = await goalService.listContributions(goalId, {
          page: pageToLoad,
          limit: PAGE_LIMIT,
        })
        const list = normalizeContributions(data)
        setItems((prev) => (pageToLoad === 1 ? list : [...prev, ...list]))
        setTotal(Number(data?.count ?? list.length))
        setPage(pageToLoad)
      } catch (err) {
        const message = apiErrorMessage(err, 'Gagal memuat riwayat setoran.')
        setError(message)
        toast.error(message)
      } finally {
        setLoading(false)
        setLoadingMore(false)
      }
    },
    [goalId],
  )

  useEffect(() => {
    if (open) {
      // eslint-disable-next-line react/set-state-in-effect -- sinkronisasi fetch saat modal dibuka
      setPendingDelete(null)
      setDeletingId('')
      setPage(1)
      setTotal(0)
      fetchHistory(1)
    }
  }, [open, fetchHistory])

  const hasMore = items.length < total

  useEffect(() => {
    if (!open) return
    function handleKey(event) {
      if (event.key === 'Escape') onClose?.()
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [open, onClose])

  async function handleDelete() {
    const contributionId = pendingDelete?.id
    if (!goal?.id || !contributionId || deletingId) return
    setDeletingId(contributionId)
    try {
      const { data } = await goalService.deleteContribution(
        goal.id,
        contributionId,
      )
      setItems((prev) => prev.filter((c) => c.id !== contributionId))
      setTotal((t) => Math.max(0, t - 1))
      setPendingDelete(null)
      // Backend DELETE mengembalikan GoalResponse terbaru (bukan envelope),
      // sedangkan POST addContribution mengembalikan { goal }. Tangani keduanya.
      const updatedGoal = data?.goal ?? data
      if (updatedGoal?.id && typeof onUpdated === 'function') {
        onUpdated(updatedGoal)
      }
      toast.success('Transaksi setoran berhasil dibatalkan')
    } catch (err) {
      toast.error(apiErrorMessage(err, 'Gagal menghapus transaksi.'))
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
        className="flex max-h-[85vh] w-full max-w-lg flex-col rounded-2xl bg-white p-6 shadow-2xl dark:bg-slate-900"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-2.5">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-indigo-100 text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-400">
              <History className="h-5 w-5" />
            </span>
            <div>
              <h4 className="font-bold text-slate-900 dark:text-white">
                Riwayat Setoran — {goal.title}
              </h4>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {total > 0 ? total : items.length} transaksi • Terkumpul{' '}
                {formatIDR(goal.current_amount)}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-slate-400 transition hover:text-slate-600 dark:hover:text-slate-200"
            aria-label="Tutup riwayat"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="mt-4 min-h-0 flex-1 overflow-y-auto">
          {loading ? (
            <div className="flex items-center justify-center gap-2 py-10 text-sm text-slate-500 dark:text-slate-400">
              <Loader2 className="h-5 w-5 animate-spin" /> Memuat riwayat...
            </div>
          ) : error ? (
            <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-center dark:border-red-500/30 dark:bg-red-500/10">
              <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
              <button
                type="button"
                onClick={fetchHistory}
                className="mt-2 rounded-lg bg-red-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-red-700"
              >
                Coba lagi
              </button>
            </div>
          ) : items.length === 0 ? (
            <div className="rounded-xl border border-dashed border-slate-300 p-8 text-center dark:border-slate-700">
              <History className="mx-auto h-8 w-8 text-slate-300 dark:text-slate-600" />
              <p className="mt-2 text-sm font-semibold text-slate-700 dark:text-slate-200">
                Belum ada setoran
              </p>
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                Setoran pertama akan tercatat di sini lengkap dengan tanggal
                dan catatan.
              </p>
            </div>
          ) : (
            <ul className="divide-y divide-slate-100 rounded-xl border border-slate-200 dark:divide-slate-800 dark:border-slate-800">              {items.map((item) => (
                <li key={item.id} className="p-3.5">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        {formatDate(item.created_at) !== '-'
                          ? formatDateTime(item.created_at)
                          : formatDate(item.created_at)}
                      </p>
                      <p className="mt-0.5 truncate text-sm font-bold text-emerald-700 dark:text-emerald-400">
                        {formatIDR(item.amount)}
                      </p>
                      {item.note ? (
                        <p className="mt-0.5 truncate text-xs text-slate-500 dark:text-slate-400">
                          “{item.note}”
                        </p>
                      ) : (
                        <p className="mt-0.5 text-xs italic text-slate-400 dark:text-slate-500">
                          Tanpa catatan
                        </p>
                      )}
                    </div>
                    <button
                      type="button"
                      onClick={() => setPendingDelete(item)}
                      className="flex shrink-0 items-center gap-1 rounded-lg border border-slate-200 px-2.5 py-1.5 text-xs font-medium text-slate-500 transition hover:border-red-200 hover:text-red-600 dark:border-slate-700 dark:text-slate-400 dark:hover:border-red-500 dark:hover:text-red-400"
                      aria-label={`Hapus setoran ${formatIDR(item.amount)}`}
                    >
                      <Trash2 className="h-3.5 w-3.5" /> Hapus
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          )}
          {hasMore && !loading && (
            <button
              type="button"
              disabled={loadingMore}
              onClick={() => fetchHistory(page + 1)}
              className="mt-3 flex w-full items-center justify-center gap-2 rounded-lg border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-600 transition hover:border-emerald-300 hover:text-emerald-700 disabled:opacity-60 dark:border-slate-700 dark:text-slate-300 dark:hover:border-emerald-500"
            >
              {loadingMore && <Loader2 className="h-4 w-4 animate-spin" />}
              Muat lebih banyak ({items.length} dari {total})
            </button>
          )}
        </div>

        <button
          type="button"
          onClick={onClose}
          className="mt-4 w-full rounded-lg border border-slate-200 px-4 py-2.5 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800"
        >
          Tutup
        </button>
      </div>

      <ConfirmModal
        isOpen={pendingDelete !== null}
        onClose={() => setPendingDelete(null)}
        onConfirm={handleDelete}
        title="Batalkan setoran ini?"
        description={
          pendingDelete
            ? `Setoran sebesar ${formatIDR(pendingDelete.amount)} akan dihapus dan saldo target terkoreksi otomatis.`
            : ''
        }
        confirmText="Ya, Hapus"
        variant="danger"
        isLoading={Boolean(deletingId)}
      />
    </div>
  )
}
