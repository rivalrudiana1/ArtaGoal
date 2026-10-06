function toNumber(value, fallback = 0) {
  const n = Number(value)
  return Number.isFinite(n) ? n : fallback
}

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
    const diff = end.getTime() - now.getTime()
    if (Number.isFinite(diff) && diff > 0) {
      const years = diff / (1000 * 60 * 60 * 24 * 365)
      return target * (1 + rate / 100) ** years
    }
  }
  return target
}

function formatDateCell(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toISOString().slice(0, 10)
}

function escapeCsvCell(value) {
  const text = String(value ?? '')
  if (/[",\n\r]/.test(text)) {
    return `"${text.replace(/"/g, '""')}"`
  }
  return text
}

/**
 * Konversi array goals menjadi file .csv dan unduh otomatis.
 * Kolom: Judul Target, Kategori, Target Awal (PV), Estimasi Inflasi (FV),
 *        Terkumpul, Sisa, Progres (%), Status, Tenggat Waktu.
 */
export function exportGoalsToCsv(goals, filename = 'artagoal-laporan.csv') {
  const rows = Array.isArray(goals) ? goals : []
  const header = [
    'Judul Target',
    'Kategori',
    'Target Awal (PV)',
    'Estimasi Inflasi (FV)',
    'Terkumpul',
    'Sisa',
    'Progres (%)',
    'Status',
    'Tenggat Waktu',
  ]

  const lines = rows.map((goal) => {
    const target = toNumber(goal?.target_amount)
    const saved = toNumber(goal?.current_amount)
    const future = Math.round(resolveFutureTarget(goal))
    const remaining = Math.max(target - saved, 0)
    const progress =
      target > 0 ? Math.min(100, (saved / target) * 100).toFixed(1) : '0.0'
    return [
      goal?.title ?? '-',
      goal?.category ?? '-',
      Math.round(target),
      future,
      Math.round(saved),
      Math.round(remaining),
      progress,
      String(goal?.status ?? '-').toUpperCase(),
      formatDateCell(goal?.target_date),
    ]
      .map(escapeCsvCell)
      .join(',')
  })

  const csv = `\uFEFF${[header.map(escapeCsvCell).join(','), ...lines].join('\r\n')}`
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
