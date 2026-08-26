import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";

export interface AuthUser {
  id: string;
  email: string;
  username: string;
}

interface LoginValues {
  email: string;
  password: string;
}

interface AuthContextValue {
  user: AuthUser | null;
  isLoading: boolean;
  login: (values: LoginValues) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

async function fetchCurrentUser(): Promise<AuthUser | null> {
  const res = await fetch("/api/me", { credentials: "include" });
  return res.ok ? res.json() : null;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // On first load (including a hard refresh), ask the backend whether the
  // httpOnly cookie we might already have is still valid.
  useEffect(() => {
    let cancelled = false;

    fetchCurrentUser()
      .then((current) => {
        if (!cancelled) setUser(current);
      })
      .catch(() => {
        if (!cancelled) setUser(null);
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, []);

  const login = async (values: LoginValues) => {
    const res = await fetch("/api/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(values),
    });

    if (!res.ok) {
      throw new Error("Invalid email or password");
    }

    setUser(await fetchCurrentUser());
  };

  const logout = async () => {
    await fetch("/api/logout", { method: "POST", credentials: "include" });
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, isLoading, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
