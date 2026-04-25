import React from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

/**
 * Guards routes that require an authenticated Firebase user.
 *
 * If the auth state is still loading, AuthProvider already blocks the
 * tree below it, so by the time we get here `loading` is false. We
 * still check it defensively to avoid a flash of <Navigate /> if that
 * behavior changes in the future.
 *
 * Anonymous users are redirected to /login while preserving the
 * originally requested path in `state.from` so the login screen can
 * send them back after a successful sign-in.
 */
export const ProtectedRoute: React.FC = () => {
  const { user, loading } = useAuth();
  const location = useLocation();

  if (loading) return null;

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }

  return <Outlet />;
};
