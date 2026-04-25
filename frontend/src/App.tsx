import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { Toaster } from 'sonner';
import { AuthProvider } from './contexts/AuthContext';
import { Layout } from './components/Layout';
import { ProtectedRoute } from './components/ProtectedRoute';
import { ErrorBoundary } from './components/ErrorBoundary';
import Home from './pages/Home';
import Login from './pages/Login';
import Register from './pages/Register';
import ContentDetail from './pages/ContentDetail';
import Library from './pages/Library';
import AIAssistant from './pages/AIAssistant';
import NotFound from './pages/NotFound';

export default function App() {
  return (
    <ErrorBoundary>
      <AuthProvider>
        <Router>
          <Routes>
            <Route path="/" element={<Layout />}>
              <Route index element={<Home />} />
              <Route path="login" element={<Login />} />
              <Route path="register" element={<Register />} />
              <Route path="content/:id" element={<ContentDetail />} />
              <Route element={<ProtectedRoute />}>
                <Route path="library" element={<Library />} />
                <Route path="assistant" element={<AIAssistant />} />
              </Route>
              <Route path="*" element={<NotFound />} />
            </Route>
          </Routes>
        </Router>
        <Toaster
          theme="dark"
          position="bottom-right"
          richColors
          closeButton
          toastOptions={{
            classNames: {
              toast:
                'glass-panel !bg-slate-900/90 !border-white/10 !text-slate-200',
            },
          }}
        />
      </AuthProvider>
    </ErrorBoundary>
  );
}
