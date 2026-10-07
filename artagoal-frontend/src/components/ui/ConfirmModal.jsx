import { useEffect } from 'react'
import { AlertTriangle, Info, Loader2, LogOut } from 'lucide-react'

const VARIANTS = {
  danger: {
    Icon: AlertTriangle,
    ring: 'bg-rose-500/15 text-rose-400 ring-rose-500/40 shadow-[0_0_36px_-6px_rgba(244,63,94,0.55)]',
    confirmButton:
      'bg-rose-600 hover:bg-rose-500 focus-visible:ring-rose-400/60 shadow-[0_8px_24px_-8px_rgba(244,63,94,0.7)]',
  },
  warning: {
    Icon: LogOut,
    ring: 'bg-amber-500/15 text-amber-400 ring-amber-500/40 shadow-[0_0_36px_-6px_rgba(245,158,11,0.55)]',
    confirmButton:
      'bg-amber-500 hover:bg-amber-400 focus-visible:ring-amber-400/60 text-slate-950 shadow-[0_8px_24px_-8px_rgba(245,158,11,0.7)]',
  },
  info: {
    Icon: Info,
    ring: 'bg-indigo-500/15 text-indigo-300 ring-indigo-500/40 shadow-[0_0_36px_-6px_rgba(99,102,241,0.55)]',
    confirmButton:
      'bg-indigo-600 hover:bg-indigo-500 focus-visible:ring-indigo-400/60 shadow-[0_8px_24px_-8px_rgba(99,102,241,0.7)]',
  },
}

export default function ConfirmModal({
  isOpen,
  onClose,
  onConfirm,
  title,
  description,
  confirmText = 'Ya, Lanjutkan',
  cancelText = 'Batal',
  variant = 'danger',
  isLoading = false,
}) {
  const config = VARIANTS[variant] ?? VARIANTS.danger
  const { Icon } = config

  useEffect(() => {
    if (!isOpen) return
    function handleKey(event) {
      if (event.key === 'Escape' && !isLoading) onClose?.()
    }
    window.addEventListener('keydown', handleKey)
    document.body.style.overflow = 'hidden'
    return () => {
      window.removeEventListener('keydown', handleKey)
      document.body.style.overflow = ''
    }
  }, [isOpen, isLoading, onClose])

  if (!isOpen) return null

  return (
    <div
      className="animate-confirm-overlay fixed inset-0 z-60 flex items-center justify-center bg-black/60 px-4 backdrop-blur-md"
      role="alertdialog"
      aria-modal="true"
      aria-label={title}
      onClick={() => {
        if (!isLoading) onClose?.()
      }}
    >
      <div
        className="animate-confirm-card w-full max-w-sm rounded-2xl border border-white/10 bg-slate-900/95 p-6 text-center shadow-2xl sm:p-8"
        onClick={(e) => e.stopPropagation()}
      >
        <span
          className={`mx-auto flex h-14 w-14 items-center justify-center rounded-full ring-2 ${config.ring}`}
        >
          <Icon className="h-7 w-7" />
        </span>

        <h3 className="mt-4 text-lg font-extrabold tracking-tight text-white">
          {title}
        </h3>
        {description && (
          <p className="mt-1.5 text-sm leading-relaxed text-slate-400">
            {description}
          </p>
        )}

        <div className="mt-6 flex gap-3">
          <button
            type="button"
            disabled={isLoading}
            onClick={onClose}
            className="flex-1 rounded-xl border border-white/15 bg-white/5 px-4 py-2.5 text-sm font-semibold text-slate-200 transition hover:border-white/25 hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {cancelText}
          </button>
          <button
            type="button"
            disabled={isLoading}
            onClick={onConfirm}
            className={`flex flex-1 items-center justify-center gap-2 rounded-xl px-4 py-2.5 text-sm font-semibold text-white transition focus-visible:ring-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-60 ${config.confirmButton}`}
          >
            {isLoading && <Loader2 className="h-4 w-4 animate-spin" />}
            {confirmText}
          </button>
        </div>
      </div>
    </div>
  )
}
