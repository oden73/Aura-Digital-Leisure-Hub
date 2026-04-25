import { Component, type ErrorInfo, type ReactNode } from 'react';
import { AlertCircle, RefreshCcw, Home } from 'lucide-react';

interface ErrorBoundaryProps {
  children: ReactNode;
}

interface ErrorBoundaryState {
  error: Error | null;
}

/**
 * Top-level error boundary. Catches render-time exceptions in the
 * subtree and shows a recoverable fallback instead of a white screen.
 *
 * Async/event-handler errors (e.g. Firebase rejections) are not
 * caught here by design — they should be surfaced via toasts or
 * inline UI in the calling component.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Unhandled UI error:', error, info.componentStack);
  }

  private handleReset = () => {
    this.setState({ error: null });
  };

  render() {
    if (!this.state.error) return this.props.children;

    return (
      <div className="min-h-screen bg-slate-950 text-slate-200 flex items-center justify-center px-6">
        <div className="max-w-md w-full glass-panel rounded-3xl p-8 text-center space-y-6">
          <div className="w-16 h-16 rounded-2xl bg-red-500/10 border border-red-500/20 flex items-center justify-center mx-auto">
            <AlertCircle className="w-8 h-8 text-red-500" />
          </div>
          <div className="space-y-2">
            <h1 className="font-display font-bold text-2xl">Something went sideways</h1>
            <p className="text-slate-400 text-sm">
              The interface ran into an unexpected error and stopped rendering.
              You can try recovering this view or go back to the home page.
            </p>
            {import.meta.env.DEV && (
              <pre className="mt-4 text-xs text-left text-red-400 bg-red-500/5 border border-red-500/10 rounded-xl p-3 overflow-auto max-h-32">
                {this.state.error.message}
              </pre>
            )}
          </div>
          <div className="flex flex-col sm:flex-row gap-3 justify-center">
            <button
              type="button"
              onClick={this.handleReset}
              className="flex items-center justify-center gap-2 bg-brand-500 hover:bg-brand-600 text-white font-bold px-6 py-3 rounded-2xl transition-all"
            >
              <RefreshCcw className="w-4 h-4" />
              Try again
            </button>
            <a
              href="/"
              className="flex items-center justify-center gap-2 bg-white/5 hover:bg-white/10 border border-white/10 text-slate-200 font-bold px-6 py-3 rounded-2xl transition-all"
            >
              <Home className="w-4 h-4" />
              Go home
            </a>
          </div>
        </div>
      </div>
    );
  }
}
