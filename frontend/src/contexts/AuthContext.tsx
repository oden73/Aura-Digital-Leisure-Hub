import React, { createContext, useContext, useEffect, useState } from 'react';
import { LoadingScreen } from '../components/LoadingScreen';
import {
  apiLogin,
  apiRegister,
  apiProfile,
  clearTokens,
  getAccessToken,
  ApiProfile,
} from '../services/api';

export interface AuthUser {
  id: string;
  username: string;
  email: string;
}

interface AuthContextType {
  user: AuthUser | null;
  loading: boolean;
  isAdmin: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType>({
  user: null,
  loading: true,
  isAdmin: false,
  login: async () => {},
  register: async () => {},
  logout: () => {},
});

function profileToUser(p: ApiProfile): AuthUser {
  return { id: p.id, username: p.username, email: p.email };
}

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!getAccessToken()) {
      setLoading(false);
      return;
    }
    apiProfile()
      .then(p => setUser(profileToUser(p)))
      .catch(() => clearTokens())
      .finally(() => setLoading(false));
  }, []);

  const login = async (email: string, password: string) => {
    await apiLogin(email, password);
    const p = await apiProfile();
    setUser(profileToUser(p));
  };

  const register = async (username: string, email: string, password: string) => {
    await apiRegister(username, email, password);
    const p = await apiProfile();
    setUser(profileToUser(p));
  };

  const logout = () => {
    clearTokens();
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, loading, isAdmin: false, login, register, logout }}>
      {loading ? <LoadingScreen /> : children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
