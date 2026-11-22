import { ChevronLeft, ChevronRight } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { formatNumber } from "@/lib/analysis";

export const DEFAULT_PAGE_SIZE = 10;

interface PaginationState<T> {
  page: number;
  pageCount: number;
  pageItems: T[];
  rangeStart: number;
  rangeEnd: number;
  setPage: (page: number) => void;
}

/**
 * Client-side pagination for API result tables. The page resets whenever the
 * underlying report (resetKey) changes so stale pages never survive new data.
 */
export function usePagination<T>(items: readonly T[], resetKey: unknown, pageSize: number = DEFAULT_PAGE_SIZE): PaginationState<T> {
  const [page, setPage] = useState(1);

  useEffect(() => {
    setPage(1);
  }, [resetKey]);

  const pageCount = Math.max(1, Math.ceil(items.length / pageSize));
  const safePage = Math.min(Math.max(1, page), pageCount);
  const rangeStart = items.length === 0 ? 0 : (safePage - 1) * pageSize + 1;
  const rangeEnd = Math.min(items.length, safePage * pageSize);

  return {
    page: safePage,
    pageCount,
    pageItems: items.slice(rangeStart - 1, rangeEnd),
    rangeStart,
    rangeEnd,
    setPage,
  };
}

interface PaginationControlsProps {
  label: string;
  page: number;
  pageCount: number;
  rangeStart: number;
  rangeEnd: number;
  total: number;
  onPageChange: (page: number) => void;
}

export function PaginationControls({ label, page, pageCount, rangeStart, rangeEnd, total, onPageChange }: PaginationControlsProps) {
  if (total <= DEFAULT_PAGE_SIZE && pageCount <= 1) {
    return null;
  }

  return (
    <nav className="flex flex-wrap items-center justify-between gap-3 pt-3 text-sm text-muted-foreground" aria-label={label}>
      <p>
        نمایش {formatNumber(rangeStart)} تا {formatNumber(rangeEnd)} از {formatNumber(total)}
      </p>
      <div className="flex items-center gap-2">
        <Button type="button" variant="outline" size="sm" disabled={page <= 1} onClick={() => onPageChange(page - 1)} aria-label="صفحهٔ قبلی">
          <ChevronRight className="size-4" />
          قبلی
        </Button>
        <span className="min-w-20 text-center">
          صفحهٔ {formatNumber(page)} از {formatNumber(pageCount)}
        </span>
        <Button type="button" variant="outline" size="sm" disabled={page >= pageCount} onClick={() => onPageChange(page + 1)} aria-label="صفحهٔ بعدی">
          بعدی
          <ChevronLeft className="size-4" />
        </Button>
      </div>
    </nav>
  );
}
