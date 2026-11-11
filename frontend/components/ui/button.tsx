import { cva, type VariantProps } from "class-variance-authority";
import { Slot } from "@radix-ui/react-slot";
import type { ButtonHTMLAttributes } from "react";
import { cn } from "@/lib/utils";
const buttonVariants = cva("focus-ring inline-flex items-center justify-center gap-2 rounded-xl text-sm font-semibold transition disabled:pointer-events-none disabled:opacity-50", { variants: { variant: { default: "bg-brand px-4 py-2.5 text-white hover:bg-brand/90", secondary: "bg-brand/10 px-4 py-2.5 text-brand hover:bg-brand/20", outline: "border bg-transparent px-4 py-2.5 hover:bg-black/5 dark:hover:bg-white/5", ghost: "px-3 py-2 hover:bg-black/5 dark:hover:bg-white/5", danger: "bg-danger px-4 py-2.5 text-white" }, size: { default: "", sm: "px-3 py-1.5 text-xs", icon: "h-10 w-10 p-0" } }, defaultVariants: { variant: "default", size: "default" } });
export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof buttonVariants> { asChild?: boolean }
export function Button({ className, variant, size, asChild, ...props }: ButtonProps) { const Comp = asChild ? Slot : "button"; return <Comp className={cn(buttonVariants({ variant, size }), className)} {...props} />; }
