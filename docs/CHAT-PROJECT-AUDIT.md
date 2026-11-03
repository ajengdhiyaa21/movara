# MOVARA Internal Chat — Project Audit

Tanggal audit: 27 Oktober 2026  
Status: Audit read-only, belum ada implementasi chat.

## 1. Kesimpulan

MOVARA adalah aplikasi Go dengan Gin, template server-side, HTMX/JavaScript, GORM, dan MySQL. Akses sudah dipisahkan untuk Admin, Manager, dan PM melalui middleware role. Session disimpan melalui cookie session dan user dibaca ulang dari repository pada setiap request.

Fitur chat internal belum ditemukan pada source code. Implementasi yang berkaitan dengan chat hanya tombol `Chat Admin` pada `web/templates/mutasi.html` yang menghasilkan link WhatsApp melalui `Renderer.WhatsAppURL`.

Rekomendasi awal: mulai dari chat teks privat Admin–PM berbasis HTTP polling atau long polling, kemudian evaluasi WebSocket setelah kebutuhan real-time dan jumlah user diketahui. WebSocket belum diperlukan sebagai keputusan final.

## 2. Arsitektur aktual

- Entry point: `cmd/server/main.go`.
- Framework HTTP: Gin.
- Bahasa: Go, versi modul `go 1.26.0`.
- Template: `web/templates/*.html` dengan layout dan partial.
- Frontend interaction: HTMX dan `web/static/app.js`.
- Styling: `web/static/app.css` dan utility class Tailwind CDN.
- Database: GORM dengan driver MySQL.
- Repository: kontrak di `internal/repository`, implementasi MySQL di `internal/repository/mysql`, memory store juga tersedia untuk data contoh.
- Business logic: `internal/service`.
- Authorization: `internal/middleware/auth.go` dan route groups di `internal/router/router.go`.
- Audit: `AuditLog`, `AuditService`, dan handler admin.

## 3. Peta autentikasi dan role

1. `POST /login` memvalidasi username/password melalui `AuthService`.
2. Password diverifikasi dengan bcrypt.
3. Session menyimpan user ID dengan key `uid`.
4. `LoadUser` membaca user ulang dari repository dan menolak user nonaktif.
5. `RequireAuth` melindungi route login-required.
6. `RequireRole` membatasi role.
7. PM menggunakan group yang juga mensyaratkan `RequireOutlet`.
8. Admin menggunakan group khusus untuk master data, rekap export, akun, outlet, dan audit.

Route relevan:

- PM: `/mutasi`, `/mutasi/input`, `/mutasi/history`, `/mutasi/riwayat`.
- Manager/Admin: `/rekap`, `/rekap/cetak`.
- Admin: `/master-data`, `/kelola/outlet`, `/kelola/pm`, `/rekap/export`.

## 4. Database dan pola akses

Model yang relevan:

- `User`: role, outlet, kategori, status aktif.
- `Outlet`: kode, nama, tipe, status aktif.
- `ProductCategory`: nama, slug, status aktif.
- `Mutation`: snapshot barcode, code, product name, category, outlet, user, direction, qty, status, waktu.
- `AuditLog`: user, outlet, action, detail, waktu.

Chat belum memiliki model conversation, participant, message, read receipt, atau notification.

Database dijalankan melalui `database.Migrate`, yang saat ini memakai GORM AutoMigrate. PRD sendiri merekomendasikan migration berversi untuk production.

## 5. Audit implementasi chat terdahulu

Tidak ditemukan:

- model chat/message/conversation;
- route chat;
- handler chat;
- service chat;
- repository chat;
- WebSocket/SSE/polling chat;
- dependency WebSocket;
- test chat;
- dokumentasi chat terdahulu.

Yang ditemukan:

- `i-chat` di `web/templates/icons.html`;
- tombol `Chat Admin` di `web/templates/mutasi.html`;
- `Renderer.WhatsAppURL` di `internal/handler/render.go`.

Kesimpulan: belum ada implementasi yang dapat digunakan kembali selain ikon dan pola tombol. Tidak ada bukti bahwa fitur chat internal pernah selesai atau terhubung.

## 6. Temuan keamanan dan regresi

### High — filter kategori PM belum konsisten

`HistoryPage` memakai `Period` dengan `CategoryID`, tetapi `historyPage` memakai `ByDate` yang hanya menerima outlet, query, dan tanggal. Riwayat hari ini berpotensi menampilkan kategori lain pada outlet yang sama. Ini bertentangan dengan PRD FR-02 dan aturan bisnis kategori PM.

### High — status outlet aktif belum dipastikan pada `RequireOutlet`

`RequireOutlet` memeriksa `OutletID`, tetapi tidak terlihat memeriksa `Outlet.IsActive`. Jika outlet dinonaktifkan, PM berpotensi tetap dapat mengakses input selama user masih aktif.

### Medium — CSRF belum menggunakan token

`SameOrigin` hanya memvalidasi Origin/Referer bila header tersedia. Belum ditemukan CSRF token per session. Ini perlu diperbaiki sebelum production.

### Medium — belum terlihat rate limiting login

Autentikasi menggunakan bcrypt, tetapi belum terlihat pembatasan percobaan login.

### Medium — validasi akun PM perlu diperketat

Handler pembuatan/edit PM perlu memastikan outlet dan kategori masih aktif. Validasi password custom pada pembuatan PM juga perlu diseragamkan dengan kebijakan minimum.

### Low — kode legacy dan warning static analysis

`go vet ./...` menemukan unreachable code pada handler `Kelola` karena route lama diarahkan ke halaman baru tetapi implementasi lama masih berada setelah `return`. Ini tidak terkait chat langsung, tetapi meningkatkan risiko maintenance.

## 7. Pengujian aktual

Pada sesi audit ini dijalankan:

- `go test ./...` — berhasil.
- `go build ./cmd/server` — berhasil pada audit sebelumnya dan source tetap dapat dibuild.
- `go vet ./...` — menemukan warning unreachable code pada `internal/handler/pages.go`.

Tidak ada browser E2E test, load test, atau security scanner yang dijalankan pada tahap ini.

## 8. Rekomendasi audit sebelum coding chat

1. Putuskan model akses percakapan Admin–PM.
2. Perbaiki pembatasan kategori dan outlet PM terlebih dahulu.
3. Tentukan apakah chat harus menjadi audit log atau hanya data komunikasi.
4. Tentukan retention dan kebutuhan backup pesan.
5. Tambahkan test authorization sebelum menambahkan koneksi real-time.
