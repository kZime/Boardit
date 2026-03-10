/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useState, useCallback, type ReactNode } from "react";

export type AuthModalMode = "login" | "register" | null;

type AuthModalContextType = {
  authModal: AuthModalMode;
  openLoginModal: () => void;
  openRegisterModal: () => void;
  closeAuthModal: () => void;
};

const AuthModalContext = createContext<AuthModalContextType>(null!);

export function AuthModalProvider({ children }: { children: ReactNode }) {
  const [authModal, setAuthModal] = useState<AuthModalMode>(null);

  const openLoginModal = useCallback(() => setAuthModal("login"), []);
  const openRegisterModal = useCallback(() => setAuthModal("register"), []);
  const closeAuthModal = useCallback(() => setAuthModal(null), []);

  return (
    <AuthModalContext.Provider
      value={{ authModal, openLoginModal, openRegisterModal, closeAuthModal }}
    >
      {children}
    </AuthModalContext.Provider>
  );
}

export function useAuthModal() {
  return useContext(AuthModalContext);
}
