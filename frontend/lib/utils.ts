import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
export function cn(...inputs: ClassValue[]) { return twMerge(clsx(inputs)); }
export function scoreText(score?: number | null) { return score == null ? "—" : score.toFixed(1); }
export function nullableNumber(value?: number | null) { return value == null ? "—" : new Intl.NumberFormat("fa-IR").format(value); }
