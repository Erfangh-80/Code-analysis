"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createContext, useContext, useEffect, useMemo, useState } from "react";
import { setApiToken } from "@/lib/api";

type Session = { token: string; setToken: (token: string) => void; theme: "light" | "dark"; toggleTheme: () => void };
const SessionContext = createContext<Session | null>(null);
export function useSession() { const value = useContext(SessionContext); if (!value) throw new Error("Session provider is missing"); return value; }

export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false } } }));
  const [token, updateToken] = useState("");
  const [theme, setTheme] = useState<"light" | "dark">("light");
  useEffect(() => { const saved = window.localStorage.getItem("hotspot-theme") as "light" | "dark" | null; if (saved) setTheme(saved); }, []);
  useEffect(() => { document.documentElement.classList.toggle("dark", theme === "dark"); window.localStorage.setItem("hotspot-theme", theme); }, [theme]);
  const value = useMemo(() => ({ token, setToken: (next: string) => { updateToken(next); setApiToken(next); }, theme, toggleTheme: () => setTheme(current => current === "light" ? "dark" : "light") }), [token, theme]);
  return <QueryClientProvider client={client}><SessionContext.Provider value={value}>{children}</SessionContext.Provider></QueryClientProvider>;
}
