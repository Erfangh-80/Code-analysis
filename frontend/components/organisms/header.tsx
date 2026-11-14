"use client";

import { KeyRound, Moon, ShieldCheck, Sun } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useSession } from "@/components/providers";

export function Header() {
  const { token, setToken, theme, toggleTheme } = useSession();
  return <header className="border-b bg-surface/85 backdrop-blur"><div className="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-4 px-5 py-4 lg:px-8"><div className="flex items-center gap-3"><div className="grid h-10 w-10 place-items-center rounded-xl bg-brand text-white shadow-lg shadow-brand/20"><ShieldCheck size={22} /></div><div><h1 className="text-base font-black tracking-tight">Code Hotspot Analyzer</h1><p className="text-xs text-muted">اولویت‌بندی توضیح‌پذیر برای بازبینی کد</p></div></div><div className="flex items-center gap-2"><div className="relative hidden sm:block"><KeyRound className="absolute right-3 top-3 text-muted" size={16} /><Input aria-label="توکن API" type="password" value={token} onChange={event => setToken(event.target.value)} placeholder="توکن API (فقط حافظه نشست)" className="w-64 pr-9" /></div><Button variant="outline" size="icon" aria-label="تغییر حالت رنگ" onClick={toggleTheme}>{theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}</Button></div></div></header>;
}
