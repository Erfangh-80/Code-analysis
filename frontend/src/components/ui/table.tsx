import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

function Table({ className, ...props }: ComponentProps<"table">) {
  return (
    <div className="w-full overflow-x-auto rounded-lg outline-none focus-visible:ring-3 focus-visible:ring-ring/25" role="region" tabIndex={0} aria-label="جدول قابل پیمایش افقی">
      <table className={cn("w-full caption-bottom text-right text-sm", className)} {...props} />
    </div>
  );
}

function TableHeader({ className, ...props }: ComponentProps<"thead">) {
  return <thead className={cn("border-b", className)} {...props} />;
}

function TableBody({ className, ...props }: ComponentProps<"tbody">) {
  return <tbody className={cn("[&_tr:last-child]:border-0", className)} {...props} />;
}

function TableRow({ className, ...props }: ComponentProps<"tr">) {
  return <tr className={cn("border-b transition-colors hover:bg-muted/55", className)} {...props} />;
}

function TableHead({ className, ...props }: ComponentProps<"th">) {
  return <th className={cn("h-11 px-4 text-right align-middle text-xs font-extrabold text-muted-foreground", className)} {...props} />;
}

function TableCell({ className, ...props }: ComponentProps<"td">) {
  return <td className={cn("p-4 align-middle", className)} {...props} />;
}

function TableCaption({ className, ...props }: ComponentProps<"caption">) {
  return <caption className={cn("mt-4 text-sm text-muted-foreground", className)} {...props} />;
}

export { Table, TableBody, TableCaption, TableCell, TableHead, TableHeader, TableRow };
