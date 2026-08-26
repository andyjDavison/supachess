import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from "react";

interface AuthModalContextValue {
  isOpen: boolean;
  /** Opens the modal. If onSuccess is given, it runs automatically right after a successful login. */
  openModal: (onSuccess?: () => void) => void;
  /** Closes the modal and discards any pending action - used for both a successful login and a manual dismiss. */
  closeModal: () => void;
  pendingAction: (() => void) | null;
}

const AuthModalContext = createContext<AuthModalContextValue | null>(null);

export function AuthModalProvider({ children }: { children: ReactNode }) {
  const [isOpen, setIsOpen] = useState(false);
  const [pendingAction, setPendingAction] = useState<(() => void) | null>(null);

  const openModal = useCallback((onSuccess?: () => void) => {
    // setPendingAction(fn) would make React treat fn as a state-updater
    // function rather than the value to store - wrapping it in another
    // function is the standard way to store a function in useState.
    setPendingAction(() => onSuccess ?? null);
    setIsOpen(true);
  }, []);

  const closeModal = useCallback(() => {
    setIsOpen(false);
    setPendingAction(null);
  }, []);

  return (
    <AuthModalContext.Provider
      value={{ isOpen, openModal, closeModal, pendingAction }}
    >
      {children}
    </AuthModalContext.Provider>
  );
}

export function useAuthModal(): AuthModalContextValue {
  const ctx = useContext(AuthModalContext);
  if (!ctx) {
    throw new Error("useAuthModal must be used within an AuthModalProvider");
  }
  return ctx;
}
