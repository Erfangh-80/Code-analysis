import type { LucideIcon } from "lucide-react";
import { NavLink } from "react-router-dom";

import { cn } from "@/lib/utils";

interface NavigationItemProps {
  to: string;
  icon: LucideIcon;
  label: string;
  end?: boolean;
  onNavigate?: () => void;
}

export function NavigationItem({ to, icon: Icon, label, end, onNavigate }: NavigationItemProps) {
  return (
    <NavLink
      end={end}
      to={to}
      onClick={onNavigate}
      className={({ isActive }) =>
        cn(
          "flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-bold transition-colors outline-none focus-visible:ring-3 focus-visible:ring-ring/30",
          isActive
            ? "bg-secondary text-secondary-foreground"
            : "text-muted-foreground hover:bg-muted hover:text-foreground",
        )
      }
    >
      <Icon className="size-4 shrink-0" aria-hidden="true" />
      <span>{label}</span>
    </NavLink>
  );
}
