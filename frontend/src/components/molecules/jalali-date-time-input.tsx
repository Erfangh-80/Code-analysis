import { useId } from "react";
import DatePicker, { type DateObject } from "react-multi-date-picker";
import TimePicker from "react-multi-date-picker/plugins/time_picker";
import persian from "react-date-object/calendars/persian";
import persian_fa from "react-date-object/locales/persian_fa";

const timePlugins = [<TimePicker key="time" position="bottom" hideSeconds />];

interface JalaliDateTimeInputProps {
  label: string;
  value: DateObject | null;
  onChange: (value: DateObject | null) => void;
}

/**
 * Jalali (شمسی) date-time picker. The user sees and picks Persian calendar
 * values; conversion to Gregorian UTC happens only at the API boundary in
 * `@/lib/datetime`.
 */
export function JalaliDateTimeInput({ label, value, onChange }: JalaliDateTimeInputProps) {
  const id = useId();

  return (
    <label className="grid gap-2 text-sm font-extrabold" htmlFor={id}>
      {label}
      <DatePicker
        id={id}
        calendar={persian}
        locale={persian_fa}
        value={value}
        onChange={(next: DateObject | null) => onChange(next ?? null)}
        format="YYYY/MM/DD HH:mm"
        plugins={timePlugins}
        inputClass="h-10 w-full rounded-md border bg-background px-3 font-mono text-sm ltr"
        containerClassName="w-full"
        calendarPosition="bottom-right"
        placeholder="انتخاب کنید"
      />
    </label>
  );
}
