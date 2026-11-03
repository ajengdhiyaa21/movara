# MOVARA — Tahap 2A: Audit dan Desain Keamanan CSRF

Tanggal audit: 27 Oktober 2026  
Mode: read-only  
Scope: keamanan session, Same-Origin, form HTML, HTMX, dan rancangan CSRF sebelum chat internal

## 1. Ringkasan temuan

1. Session memakai `gin-contrib/sessions/cookie`, menyimpan user ID pada key `uid`, dan user dimuat ulang dari repository pada setiap request.
2. Cookie session dikonfigurasi `HttpOnly: true`, `SameSite: Lax`, `Path: /`, `MaxAge: 8 jam`, dan `Secure` mengikuti mode production.
3. `SameOrigin()` dipasang pada root router sehingga berjalan sebelum route publik maupun authenticated. Middleware hanya memeriksa method POST, PUT, dan DELETE.
4. Jika Origin dan Referer sama-sama kosong, request diteruskan. Jika salah satu tersedia dan host berbeda, request ditolak 403.
5. Belum ditemukan token CSRF pada form HTML, HTMX, atau JavaScript. Same-Origin check bukan pengganti token CSRF.
6. Semua endpoint yang mengubah state yang ditemukan saat audit menggunakan POST, tetapi perlindungan saat ini masih bergantung pada Same-Origin dan belum memiliki token.
7. Logout sudah menggunakan POST, namun belum memiliki token CSRF.
8. Tidak ada perubahan source code, database, dependency, konfigurasi, atau dokumen lama pada tahap ini.

## 2. Kondisi keamanan yang terverifikasi

### Session dan autentikasi

| Area | Kondisi aktual | Bukti |
|---|---|---|
| Pembuatan session | Session dibersihkan, diisi `uid`, lalu disimpan setelah autentikasi berhasil | `internal/handler/auth.go:46-50` |
| Validasi session | `LoadUser` mengambil `uid`, memuat user dari repository, dan hanya memasang user jika `IsActive` | `internal/middleware/auth.go:28-44` |
| User nonaktif | Session dibersihkan dan disimpan ulang jika user tidak ada atau nonaktif | `internal/middleware/auth.go:35-41` |
| Pengakhiran session | Logout melakukan `Clear`, `Save`, no-store, dan `Clear-Site-Data: "cache"` | `internal/handler/auth.go:57-66` |
| Login session fixation | Session data dibersihkan sebelum `uid` ditulis | `internal/handler/auth.go:46-48` |
| Session expiry | Cookie memiliki `MaxAge: 8 * 60 * 60`; mekanisme idle timeout terpisah tidak ditemukan | `internal/router/router.go:31-37` |
| HttpOnly | Aktif | `internal/router/router.go:34` |
| Secure | Aktif hanya ketika `cfg.IsProduction()` true | `internal/router/router.go:35` |
| SameSite | `SameSiteLaxMode` | `internal/router/router.go:36` |
| Secret | Diambil dari `SESSION_SECRET`; konfigurasi menolak secret production yang kosong/terlalu pendek | `internal/config/config.go:26-38` |

Catatan: `Secure` pada mode development bernilai false berdasarkan kode. Hal ini kompatibel dengan HTTP lokal, tetapi deployment wajib memakai production mode dan HTTPS.

### Same-Origin

`SameOrigin()` didaftarkan pada `r.Use(...)` di `internal/router/router.go:27`, sebelum route didaftarkan. Karena itu middleware berada pada jalur global router, termasuk `/login`, `/logout`, route PM, manager, dan admin.

Perilaku aktual di `internal/middleware/csrf.go:13-30`:

- GET, PATCH, dan OPTIONS tidak masuk pemeriksaan method saat ini.
- POST, PUT, dan DELETE diperiksa.
- Origin dan Referer dibaca satu per satu.
- Header kosong dilewati.
- Header yang tersedia di-parse; jika parse gagal atau `u.Host` berbeda dari `Request.Host`, request dibalas 403.
- Jika kedua header kosong, request diteruskan.
- Perbandingan host memakai `strings.EqualFold`.

Proteksi ini membatasi request browser lintas origin yang mengirim header tersebut, tetapi tidak membuktikan kepemilikan session dan tidak mengharuskan secret token per session.

### Autentikasi, otorisasi, dan CSRF dibedakan

- Autentikasi: `LoadUser` dan `RequireAuth` memastikan request memiliki user aktif.
- Otorisasi: `RequireRole` dan `RequireOutlet` membatasi role/outlet.
- Same-Origin: memeriksa host pada header browser tertentu.
- CSRF token: **belum ada**. Belum ditemukan token acak yang terikat pada session dan diverifikasi pada endpoint state-changing.

## 3. Inventaris route dan middleware

Semua route di bawah berada di `internal/router/router.go`. Route authenticated mewarisi `RequireAuth`; route PM juga mewarisi `RequireRole(pm)` dan `RequireOutlet`; route manager/admin mewarisi role group masing-masing. Semua route juga melewati middleware global `SameOrigin` dan session middleware.

| Method | Path | Handler | Auth/role | State | CSRF saat ini |
|---|---|---|---|---|---|
| GET | `/login` | `ShowLogin` | Publik | Tidak | Tidak perlu token |
| POST | `/login` | `Login` | Publik | Membuat session/audit | Same-Origin, tanpa token |
| GET | `/` | `Home` | Auth | Tidak | Tidak perlu token |
| POST | `/logout` | `Logout` | Auth | Menghapus session/audit | Same-Origin, tanpa token |
| GET | `/mutasi*` | Mutation handlers | PM + outlet | Baca/partial | Tidak perlu token |
| POST | `/mutasi` | `Create` | PM + outlet | Membuat mutasi | Same-Origin, tanpa token |
| POST | `/mutasi/today/qty` | `UpdateQty` | PM + outlet | Mengubah qty | Same-Origin, tanpa token |
| GET | `/mutasi/today/:id/edit-history` | `EditHistory` | PM + outlet | Baca | Tidak perlu token |
| POST | `/pindah-outlet` | `RequestTransfer` | PM + outlet | Membuat permintaan | Same-Origin, tanpa token |
| GET | `/rekap*` | Recap handlers | Manager/Admin | Baca/export/print | Tidak perlu token |
| POST | `/rekap/mutasi/:id/edit` | `Update` | Admin | Mengubah jurnal | Same-Origin, tanpa token |
| POST | `/rekap/mutasi/:id/delete` | `Delete` | Admin | Menghapus jurnal | Same-Origin, tanpa token |
| POST | `/password` | `ChangePassword` | Admin | Mengubah password | Same-Origin, tanpa token |
| POST | `/profile` | `UpdateProfile` | Admin | Mengubah profil | Same-Origin, tanpa token |
| POST | `/master-data/categories` | `AddCategory` | Admin | Membuat kategori | Same-Origin, tanpa token |
| POST | `/master-data/categories/:id/toggle` | `ToggleCategory` | Admin | Mengubah status kategori | Same-Origin, tanpa token |
| POST | `/master-data` | `Create` | Admin | Membuat produk | Same-Origin, tanpa token |
| POST | `/master-data/import` | `Import` | Admin | Import produk | Same-Origin, tanpa token |
| POST | `/master-data/:id` | `Update` | Admin | Mengubah produk | Same-Origin, tanpa token |
| POST | `/master-data/:id/deactivate` | `Deactivate` | Admin | Mengubah status produk | Same-Origin, tanpa token |
| POST | `/kelola/outlet` | `AddOutlet` | Admin | Membuat outlet | Same-Origin, tanpa token |
| POST | `/kelola/outlet/:id/edit` | `UpdateOutlet` | Admin | Mengubah outlet | Same-Origin, tanpa token |
| POST | `/kelola/outlet/:id/toggle` | `ToggleOutlet` | Admin | Mengubah status outlet | Same-Origin, tanpa token |
| POST | `/kelola/pm` | `AddPM` | Admin | Membuat akun PM | Same-Origin, tanpa token |
| POST | `/kelola/pm/:id/edit` | `UpdatePM` | Admin | Mengubah akun PM | Same-Origin, tanpa token |
| POST | `/kelola/pm/:id/deactivate` | `DeactivatePM` | Admin | Mengubah status PM | Same-Origin, tanpa token |
| POST | `/kelola/transfer/:id` | `DecideTransfer` | Admin | Menyetujui/menolak transfer | Same-Origin, tanpa token |

Tidak ditemukan route GET yang secara eksplisit mengubah state pada router. Endpoint `/rekap/export` dan `/master-data/import/template` adalah GET tetapi fungsi utamanya membaca/menghasilkan file.

## 4. Inventaris form, HTMX, dan JavaScript

### Form HTML

Form POST aktual ditemukan pada:

- `web/templates/login.html:44` — login.
- `web/templates/layout.html:60` — logout.
- `web/templates/mutasi.html:78` dan `:276` — create mutasi dan edit qty.
- `web/templates/transfer.html:5` — request transfer.
- `web/templates/rekap.html:168` dan `web/templates/mutation_edit.html:9` — delete/edit jurnal admin.
- `web/templates/master.html`, `product_create.html`, `product_edit.html`, `product_import.html` — produk/import.
- `web/templates/categories.html` — kategori.
- `web/templates/outlet_create.html`, `outlet_edit.html`, `outlet_manage.html` — outlet.
- `web/templates/pm_create.html`, `pm_edit.html`, `pm_manage.html` — akun PM.
- `web/templates/profile.html`, `password.html` — profil dan password admin.

Belum ada input hidden atau meta token CSRF yang terlihat pada layout/form tersebut.

### HTMX

HTMX digunakan terutama pada:

- `web/templates/mutasi.html:78` untuk POST `/mutasi`.
- `web/templates/mutasi.html:276` untuk POST `/mutasi/today/qty`.
- GET lookup, reset, history, pagination, dan partial history pada template mutasi.

Desain token nantinya harus mendukung `hx-post` tanpa mengganti nama field bisnis atau target swap.

### JavaScript

`web/static/app.js` menangani filter UI, modal, dan request history edit. Pada sekitar baris 201 terdapat `fetch('/mutasi/today/' + id + '/edit-history', { headers: { 'X-Requested-With': 'XMLHttpRequest' } })`; request ini GET/read-only sehingga tidak memerlukan token CSRF.

Belum ditemukan fetch/XHR JavaScript yang melakukan POST/PUT/PATCH/DELETE. Jika nanti ditambahkan untuk chat, header token harus disisipkan melalui wrapper global atau helper terpusat.

## 5. Kesenjangan Same-Origin versus CSRF token

| Mekanisme | Melindungi dari | Tidak menjamin |
|---|---|---|
| Session cookie HttpOnly/SameSite | Pengurangan akses script dan pengiriman lintas-site tertentu | Validasi intent setiap request |
| Same-Origin | Menolak sebagian request dengan Origin/Referer host berbeda | Request tanpa header, konfigurasi proxy, atau token rahasia |
| CSRF token | Membuktikan request membawa nilai rahasia dari halaman/session sah | XSS, authorization yang salah, atau Origin WebSocket |
| Role authorization | Membatasi user sesuai role | Keaslian request dari browser user |

Desain target harus memakai token CSRF sebagai lapisan tambahan, bukan menggantikan session, authorization, atau validasi Origin.

## 6. Desain implementasi yang direkomendasikan

### Token dan session

Gunakan token acak kuat yang terikat pada session server/cookie session. Token dibuat saat halaman authenticated dirender atau saat session authenticated pertama tersedia, lalu disimpan pada session key khusus, misalnya `csrf_token`. Nilai token tidak boleh berasal dari user ID, timestamp, atau secret statis.

Satu token session yang stabil selama session aktif cukup untuk pola server-rendered saat ini. Setelah login berhasil, token harus dibuat ulang; setelah logout token tidak boleh dapat dipakai lagi karena session dibersihkan.

### Form HTML

Layout authenticated dapat menyediakan token untuk template. Setiap form state-changing menambahkan hidden input `csrf_token`, tanpa mengubah nama field bisnis yang sudah ada. Form login publik dapat dikecualikan atau diberi token dari anonymous session berdasarkan keputusan threat model; logout sebaiknya diwajibkan token karena mengubah session.

### HTMX

Tambahkan token melalui hidden input pada form HTMX dan/atau header global `X-CSRF-Token` pada event konfigurasi HTMX. Pilih satu sumber canonical agar tidak terjadi perbedaan token antara body dan header. Untuk endpoint yang sudah memakai form body, hidden input paling kompatibel; header global membantu request HTMX non-form.

### JavaScript

Sediakan helper yang membaca token dari meta tag atau cookie non-HttpOnly khusus CSRF, lalu menambahkan header `X-CSRF-Token` hanya untuk method state-changing. Jangan menaruh session secret pada JavaScript. Request GET/read-only tidak perlu token.

### Middleware

Middleware token ditempatkan setelah session middleware dan sebelum handler state-changing. Ia memvalidasi method yang mengubah state, token request, token session, dan session authenticated sesuai kebijakan route. Kegagalan harus mengembalikan 403 dengan pesan generik dan tidak membocorkan token atau detail session.

### Error dan expiry

- Token hilang: 403.
- Token tidak cocok: 403.
- Session expired/nonaktif: redirect `/login` untuk navigasi biasa; response HTMX memakai mekanisme `HX-Redirect` yang sudah dipakai `RequireAuth`.
- Setelah login/logout, halaman lama harus di-refresh agar token lama tidak dipakai.

### WebSocket chat

Token CSRF HTTP tidak menggantikan autentikasi dan validasi Origin WebSocket. Handshake chat harus:

1. Memastikan session cookie valid dan user aktif.
2. Memastikan role dan membership conversation.
3. Memvalidasi `Origin` terhadap allowlist konfigurasi.
4. Menetapkan batas frame, deadline, ping/pong, dan cleanup connection.
5. Jika pesan pertama dikirim melalui HTTP, endpoint tersebut tetap memakai CSRF token.

## 7. Rencana test dan regression

### Tahap A — baseline

Kemungkinan file: `internal/middleware/csrf_test.go`, route test baru, dan test helper session.

Kriteria lulus: baseline test/build tetap lulus; tidak ada migration atau database production.

### Tahap B — token middleware

Kemungkinan file: `internal/middleware/csrf.go`, middleware test, dan session helper.

Test positif: token valid pada POST authenticated diteruskan.  
Test negatif: token hilang, salah, session berbeda, dan method state-changing ditolak.  
Test kompatibilitas: GET tidak ditolak; request HTMX valid diteruskan.

### Tahap C — template dan HTMX

Kemungkinan file: `web/templates/layout.html`, seluruh form POST, dan `web/static/app.js` bila header global digunakan.

Test: login, logout, create/update/delete mutasi, produk, kategori, outlet, PM, import, transfer, dan edit qty tetap mengirim token; partial HTMX tetap swap ke target yang sama.

### Tahap D — authorization regression

Test matrix minimal:

- Admin hanya mengakses route admin.
- Manager hanya membaca rekap sesuai kebijakan.
- PM hanya mutasi outlet/kategori yang diizinkan.
- User nonaktif ditolak walaupun memiliki cookie lama.
- Token user A tidak berlaku pada session user B.

### Tahap E — deployment check

Verifikasi production mode, HTTPS, `Secure` cookie, session secret kuat, reverse proxy host/origin, dan observability 403 tanpa mencatat token.

## 8. Daftar file kemungkinan terdampak pada implementasi mendatang

- `internal/middleware/csrf.go` — validasi token dan policy method.
- `internal/middleware/csrf_test.go` — test positif/negatif.
- `internal/router/router.go` — urutan middleware dan kemungkinan route exception.
- `internal/handler/auth.go` — pembuatan/rotasi token saat login/logout bila diperlukan.
- `internal/middleware/auth.go` — integrasi expiry/HTMX error bila diperlukan.
- `web/templates/layout.html` — token global/meta dan hidden field strategy.
- Semua template yang memiliki POST form — hidden token.
- `web/static/app.js` — helper/header untuk fetch atau HTMX bila diperlukan.
- Route chat masa depan dan service session WebSocket — validasi session/Origin terpisah.

Daftar ini adalah dampak yang diperkirakan berdasarkan kode aktual; belum ada file yang diubah.

## 9. Risiko dan asumsi yang belum terverifikasi

- Reverse proxy production belum diaudit; `Request.Host`, Origin, dan Referer dapat dipengaruhi konfigurasi proxy.
- Tidak ada browser E2E test yang dijalankan.
- Tidak ada database test terisolasi; alur runtime form tidak dijalankan.
- Tidak diketahui apakah ada deployment lain yang mengandalkan client non-browser tanpa Origin/Referer. Saat ini middleware memang mengizinkan kedua header kosong.
- Session store cookie berarti token session ikut berada dalam payload cookie yang ditandatangani/enkripsi oleh library; detail kriptografi library belum diaudit lebih lanjut.
- Kebijakan apakah `/login` wajib token CSRF memerlukan keputusan threat model; logout tetap direkomendasikan wajib token.

## 10. Keputusan yang membutuhkan persetujuan

1. Apakah token dikirim melalui hidden input, header HTMX, atau kombinasi dengan satu sumber canonical?
2. Apakah login publik diwajibkan token atau hanya endpoint authenticated/state-changing?
3. Apakah request POST dengan Origin/Referer kosong tetap diizinkan untuk integrasi non-browser?
4. Apakah token disimpan di session cookie yang ada atau memakai cookie CSRF terpisah non-HttpOnly?
5. Apa allowlist Origin production untuk handshake WebSocket?
6. Database test dan environment test apa yang akan digunakan untuk regression?

## 11. Pernyataan perubahan

Audit dan desain ini hanya membaca kode serta konfigurasi non-rahasia. Pada Tahap 2A:

- Tidak ada source code yang diubah.
- Tidak ada template atau JavaScript yang diubah.
- Tidak ada dependency yang ditambahkan.
- Tidak ada konfigurasi yang diubah.
- Tidak ada migration, seeder, startup server, atau koneksi database.
- Tidak ada token, password, secret, atau isi `.env` yang dicantumkan.
- Satu-satunya file baru adalah dokumen laporan ini.

Implementasi CSRF belum dilakukan dan audit berhenti setelah dokumen ini dibuat.
