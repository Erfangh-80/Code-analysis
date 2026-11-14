# شاخص‌ها و امتیاز

## دامنه فایل

فقط blobهای ثبت‌شده در `HEAD` یا SHA انتخاب‌شده بررسی می‌شوند. `.go`, `.py`, `.js`, `.mjs`, `.cjs`, `.jsx` پشتیبانی می‌شوند. مسیرهای دارای segmentهای `.git`, `node_modules`, `vendor`, `dist`, `build`, `.next`, `__pycache__` کنار گذاشته می‌شوند. فایل NULدار، بزرگ‌تر از سقف، پسوند ناشناخته یا parse-error دلیل وضعیت خود را دارد.

## LOC و ساختار

Tree-sitter برای Go، Python و JavaScript استفاده می‌شود. comment nodeها از parse tree گرفته می‌شوند؛ هر خط فقط-comment یا خالی LOC نیست، اما کد همراه comment و stringهای دارای `#` یا `//` کد هستند. docstring پایتون کد است. CRLF و فایل بدون newline انتهایی در شمارش physical line مدیریت می‌شوند. برای parse error، شاخص‌های مشتق‌شده `null` و فایل از امتیاز حذف می‌شود.

`Complexity(function) = 1 + decision points`:

- Go: `if`, `for`, caseهای expression/select، و `&&`/`||`.
- Python: `if`, `for`, `while`, conditional expression، `except`، و `and`/`or`.
- JavaScript: شرط، حلقه، case، catch، ternary و `&&`/`||`.

تابع تو‌در‌تو جداگانه حساب می‌شود و تصمیم‌هایش به والد اضافه نمی‌شود. Complexity فایل برابر max complexity توابع آن است؛ فایل بدون تابع `0` می‌گیرد. Go کلاس ندارد: `classes: null`، نه صفر ساختگی.

## تاریخچه Git

بازه دقیق `[since, until)` و UTC است. پیش‌فرض از 180 روز قبل تا لحظه ثبت Job است. ancestorهای commit snapshot بررسی می‌شوند؛ merge commitها عمداً شمرده نمی‌شوند تا دوباره‌شماری نشود. root commit با tree خالی diff می‌شود. rename بر مبنای path است و تاریخ پیش از rename ممکن است دنبال نشود.

برای هر path: `churn = additions + deletions`، تعداد commit یکتا، تعداد نویسنده یکتا، نخستین/آخرین تغییر و تمرکز نویسنده محاسبه می‌شوند. نام/email فقط در حافظه برای شمارش است و در DB/API/log ذخیره نمی‌شود. اگر تاریخچه کامل و تغییری در بازه نبود churn/frequency صفر است، اما concentration/dispersion `null` است.

## فرمول hotspot

وزن‌های پایه پیش‌فرض complexity=.5، churn=.3، frequency=.2 هستند؛ منفی نیستند، مجموع باید مثبت باشد و به مجموع یک نرمال می‌شوند. برای هر شاخص واجد شرایط:

```
t = log1p(raw)
n = (t - min(t)) / (max(t) - min(t))
score = 100 × (wc × nc + wh × nh + wf × nf)
```

اگر یک شاخص ثابت باشد normalized آن صفر است و هشدار ثبت می‌شود. اگر کمتر از دو فایل قابل قیاس باشد یا همه ثابت باشند، `score=null` است. API مقدار خام، normalized، وزن مؤثر و contribution هر جزء را بازمی‌گرداند. امتیاز درصد احتمال باگ نیست و فقط در همان تحلیل معنا دارد. جمع وزنی تضمین نمی‌کند که هر فایل با Complexity و Churn هم‌زمان بالا همیشه اول باشد؛ UI فیلتر «هر دو شاخص بالا» دارد.

## ماژول و Heatmap

ماژول، اولین پوشه path و برای فایل ریشه `.` است. Heatmap فایل‌ها 10×10 است: محور X churn normalized و Y complexity normalized؛ رنگ متوسط score خانه، همراه تعداد و عدد score است. empty cell خاکستری است. Heatmap ماژول، میانگین normalized شاخص‌های فایل‌های قابل امتیاز همان ماژول است.
