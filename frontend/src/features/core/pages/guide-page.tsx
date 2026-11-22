import { BookOpen, FileCode2, GitBranch, Info, ListOrdered, ShieldCheck, Sparkles } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

const metrics = [
  { name: "خطوط کد (LOC)", source: "تحلیل ایستا", description: "تعداد خطوط واقعی کد؛ بدون خطوط خالی و کامنت. فایل‌های بزرگ‌تر معمولاً سخت‌تر خوانده و نگهداری می‌شوند." },
  { name: "بیشترین طول تابع", source: "تحلیل ایستا", description: "طولانی‌ترین تابع یا متد فایل. توابع خیلی بلند نشانهٔ پیچیدگی و محل مناسب بازسازی (refactor) هستند." },
  { name: "بیشترین طول کلاس", source: "تحلیل ایستا", description: "طولانی‌ترین کلاس در فایل‌های JavaScript و Python. کلاس‌های بزرگ معمولاً مسئولیت‌های زیادی بر عهده دارند." },
  { name: "حجم تغییرات (Git Churn)", source: "تاریخچهٔ Git", description: "مجموع خطوطی که در تاریخچهٔ انتخاب‌شده به فایل افزوده یا از آن حذف شده‌اند. Churn بالا یعنی فایل مدام دستکاری شده است." },
  { name: "دفعات تغییر (Change Frequency)", source: "تاریخچهٔ Git", description: "تعداد commitهای متفاوتی که فایل را تغییر داده‌اند. فایلی که زیاد تغییر می‌کند، نقطهٔ حساس پروژه است." },
];

const levels = [
  { badge: "success" as const, label: "کم (۰ تا ۳۴)", description: "فایل از نظر معیارها وضعیت عادی دارد؛ نیاز فوری به بازبینی نیست." },
  { label: "متوسط (۳۵ تا ۶۴)", badge: "warning" as const, description: "ارزش بازبینی دارد؛ مخصوصاً اگر بخش مهمی از سیستم است." },
  { label: "زیاد (۶۵ تا ۱۰۰)", badge: "destructive" as const, description: "از نظر ترکیب بزرگی، پیچیدگی و تغییرپذیری در صدر قرار دارد؛ اولویت بازبینی و بازسازی." },
];

export function GuidePage() {
  return (
    <div className="space-y-6">
      <section className="rounded-2xl border bg-card/80 p-6 shadow-card">
        <p className="flex items-center gap-2 text-sm font-black text-primary"><BookOpen className="size-4" />راهنمای استفاده</p>
        <h2 className="mt-2 text-2xl font-black tracking-tight sm:text-3xl">Code Risk Analyzer چه کار می‌کند؟</h2>
        <p className="mt-3 max-w-3xl text-sm leading-7 text-muted-foreground">
          این ابزار یک تحلیل‌گر کاملاً محلی است که به شما کمک می‌کند بفهمید کدام فایل‌های پروژه برای بازبینی و بازسازی اولویت دارند.
          کد شما هرگز اجرا نمی‌شود و به هیچ سرویس خارجی ارسال نمی‌شود؛ همهٔ محاسبات روی همان دستگاهی انجام می‌شود که پروژه در آن قرار دارد.
        </p>
        <div className="mt-4 flex flex-wrap gap-2">
          <Badge variant="info">تحلیل ایستای Go، JavaScript و Python</Badge>
          <Badge variant="info">تحلیل تاریخچهٔ Git</Badge>
          <Badge variant="info">امتیاز Hotspot قابل‌توضیح</Badge>
          <Badge variant="success">کاملاً محلی و خواندنی</Badge>
        </div>
      </section>

      <section className="grid gap-6 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <FileCode2 className="size-5 text-primary" />
            <CardTitle className="mt-2">۱. تحلیل ایستا</CardTitle>
            <CardDescription>فقط مسیر پروژه را وارد می‌کنید.</CardDescription>
          </CardHeader>
          <CardContent className="text-sm leading-7 text-muted-foreground">
            ابزار فایل‌های Go، JavaScript و Python را خودش پیدا می‌کند (پوشه‌های وابستگی و خروجی build رد می‌شوند) و بدون اجرای کد، خطوط کد، توابع و کلاس‌های هر فایل را می‌شمارد. فایل‌های خراب یا ناخوانا با وضعیت مشخص گزارش می‌شوند و بقیهٔ نتایج سالم می‌مانند.
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <GitBranch className="size-5 text-primary" />
            <CardTitle className="mt-2">۲. تاریخچهٔ Git</CardTitle>
            <CardDescription>اختیاری، ولی برای امتیاز کامل توصیه می‌شود.</CardDescription>
          </CardHeader>
          <CardContent className="text-sm leading-7 text-muted-foreground">
            تاریخچهٔ commitهای مخزن فقط‌خواندنی خوانده می‌شود تا مشخص شود هر فایل چقدر تغییر کرده و چند نفر روی آن کار کرده‌اند. انتخاب بازهٔ تاریخ اختیاری است: اگر دورهٔ مشخصی مدنظرتان است آن را با تقویم شمسی انتخاب کنید، وگرنه هر دو فیلد را خالی بگذارید تا کل تاریخچه بررسی شود.
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <Sparkles className="size-5 text-primary" />
            <CardTitle className="mt-2">۳. امتیاز Hotspot</CardTitle>
            <CardDescription>محاسبه فقط در Backend انجام می‌شود.</CardDescription>
          </CardHeader>
          <CardContent className="text-sm leading-7 text-muted-foreground">
            موتور امتیازدهی، معیارهای هر فایل را با بقیهٔ فایل‌های همان تحلیل مقایسه می‌کند و امتیازی از ۰ تا ۱۰۰ با رتبه و سطح ریسک می‌سازد. در صفحهٔ جزئیات هر فایل می‌بینید دقیقاً کدام معیار چقدر در امتیازش سهم داشته است.
          </CardContent>
        </Card>
      </section>

      <Card>
        <CardHeader>
          <ListOrdered className="size-5 text-primary" />
          <CardTitle className="mt-2">معیارها یعنی چه؟</CardTitle>
          <CardDescription>پنج معیاری که امتیاز Hotspot از روی آن‌ها ساخته می‌شود.</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader><TableRow><TableHead>معیار</TableHead><TableHead>منبع</TableHead><TableHead>معنی</TableHead></TableRow></TableHeader>
            <TableBody>
              {metrics.map((metric) => (
                <TableRow key={metric.name}>
                  <TableCell className="font-extrabold">{metric.name}</TableCell>
                  <TableCell><Badge variant="muted">{metric.source}</Badge></TableCell>
                  <TableCell className="leading-6 text-muted-foreground">{metric.description}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>سطح ریسک یعنی چه؟</CardTitle>
          <CardDescription>امتیاز بین ۰ تا ۱۰۰ است و به سه سطح تقسیم می‌شود.</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 sm:grid-cols-3">
          {levels.map((level) => (
            <div key={level.label} className="rounded-xl border bg-background/60 p-4">
              <Badge variant={level.badge}>{level.label}</Badge>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">{level.description}</p>
            </div>
          ))}
          <p className="text-sm leading-7 text-muted-foreground sm:col-span-3">
            امتیاز از ترکیب وزن‌دار معیارها ساخته می‌شود: مقدار هر معیار با سقفِ صدک ۹۵ همان تحلیل نرمال و در وزنش ضرب می‌شود و مجموع، امتیاز نهایی است. امتیازها نسبی هستند؛ یعنی جایگاه فایل را نسبت به بقیهٔ فایل‌های همان تحلیل نشان می‌دهند و بین دو پروژهٔ متفاوت قابل مقایسه نیستند.
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <Info className="size-5 text-primary" />
          <CardTitle className="mt-2">نکته‌های مهم</CardTitle>
        </CardHeader>
        <CardContent>
          <ul className="space-y-2 text-sm leading-7 text-muted-foreground">
            <li>• این امتیاز یک «سیگنال اولویت‌بندی برای بازبینی» است، نه پیش‌بینی وجود باگ یا آسیب‌پذیری امنیتی.</li>
            <li>• اگر تحلیل Git انجام نشده باشد، امتیاز فقط با معیارهای ایستا و با وزن‌های بازتنظیم‌شده محاسبه می‌شود و این موضوع در صفحه اعلام می‌شود.</li>
            <li>• جدول‌های نتیجه صفحه‌بندی دارند و با دکمهٔ «جزئیات» می‌توانید همهٔ داده‌های هر فایل را یکجا ببینید.</li>
            <li>• نتایج فقط در حافظهٔ همین صفحه نگه داشته می‌شوند و با refresh پاک می‌شوند.</li>
            <li className="flex items-start gap-2"><ShieldCheck className="mt-1 size-4 shrink-0 text-success" />همهٔ داده‌ها روی دستگاه شما می‌مانند؛ پروژه اجرا نمی‌شود و چیزی به اینترنت ارسال نمی‌شود.</li>
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
