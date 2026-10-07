import { BrowserRouter } from 'react-router-dom'
import { Toaster } from 'sonner'
import { AuthProvider } from './context/AuthContext.jsx'
import { ThemeProvider } from './context/ThemeContext.jsx'
import AppRoutes from './routes/AppRoutes.jsx'

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
        <Toaster
          position="top-right"
          richColors
          closeButton
          theme="dark"
          expand={false}
          toastOptions={{
            style: {
              background: '#0f172a',
              border: '1px solid rgba(255, 255, 255, 0.1)',
              color: '#f1f5f9',
            },
          }}
        />
      </AuthProvider>
    </ThemeProvider>
  )
}

export default App
