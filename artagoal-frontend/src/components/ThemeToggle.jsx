import { Moon, Sun } from 'lucide-react'
import { useTheme } from '../context/ThemeContext.jsx'

export default function ThemeToggle({ dark = false }) {
  const { isDark, toggle } = useTheme()
  const Icon = isDark ? Sun : Moon
  return (
    <button
      type="button"
      onClick={toggle}
      aria-label={isDark ? 'Mode terang' : 'Mode gelap'}
      title={isDark ? 'Mode terang' : 'Mode gelap'}
      className={
        dark
          ? 'flex h-9 w-9 items-center justify-center rounded-full border border-slate-700 text-slate-300 transition hover:border-emerald-500 hover:text-emerald-400'
          : 'flex h-9 w-9 items-center justify-center rounded-full border border-slate-200 text-slate-500 transition hover:border-emerald-300 hover:text-emerald-600'
      }
    >
      <Icon className="h-4 w-4" />
    </button>
  )
}
