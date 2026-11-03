# MOVARA — Tahap 3A: Rencana Perbaikan Bug P1

Tanggal audit: 27 Oktober 2026  
Mode: read-only  
Scope: verifikasi filter kategori PM, outlet nonaktif, dan unreachable code.

## 1. Ringkasan

Ketiga temuan diverifikasi dari kode aktual:

| Temuan | Status | Prioritas |
|---|---|---|
| Filter kategori pada riwayat harian PM | Terkonfirmasi | P1 |
| Validasi outlet nonaktif pada middleware PM | Terkonfirmasi sebagai gap validasi | P1 |
| Unreachable code pada `pages.go` | Terkonfirmasi | P1 kualitas kode |

Tidak ada perbaikan yang diterapkan.

## 2. A. Filter kategori riwayat harian PM

### Bukti kode aktual

- `internal/handler/mutation.go:118-131`: `HistoryPage` memanggil `Period(*u.OutletID, u.CategoryID, ...)` sehingga jalur riwayat periode meneruskan kategori PM.
- `internal/handler/mutation.go:285-292`: `historyPage` memanggil `h.svc.ByDate(*u.OutletID, q, date)` tanpa `u.CategoryID`.
- `internal/service/mutation.go:60-63`: `Period` mengisi `repository.MutationFilter{OutletID, CategoryID, ...}`.
- `internal/service/mutation.go:71-94`: `ByDate` memakai `ListByOutletBetween(outletID, from, to)` lalu hanya memfilter query; tidak ada filter kategori.
- `internal/repository/repository.go:31-37`: `MutationFilter` sudah memiliki `CategoryID`.
- `internal/repository/mysql/mutations.go:34-37`: `ListByOutletBetween` hanya memakai `outlet_id` dan rentang waktu.
- `internal/repository/mysql/mutations.go:40-47`: `Search` sudah mendukung `category_id`.
- `internal/models/models.go:63-70`: `Mutation` menyimpan snapshot `CategoryID`.

### Kesimpulan

Temuan **terkonfirmasi**. PM dibatasi ke outlet pada jalur harian, tetapi kategori PM belum diterapkan pada hasil harian. Ini tidak memperluas akses ke outlet lain, namun berpotensi menampilkan kategori lain dalam outlet yang sama.

### Perubahan minimal yang direncanakan

Pilihan minimal yang perlu diputuskan pada implementasi:

1. Tambahkan `categoryID *uint` pada signature `ByDate` dan teruskan ke query repository; atau
2. Ubah `ListByOutletBetween` menjadi menerima filter kategori yang eksplisit.

Pilihan yang lebih kecil terhadap kontrak saat ini adalah menambah parameter kategori pada `ByDate` dan menambahkan filter kategori pada repository read harian. Jangan melakukan filter hanya di template/handler setelah query, agar service tetap menjadi batas akses data.

File yang kemungkinan terdampak:

- `internal/service/mutation.go`.
- `internal/repository/repository.go` bila kontrak read harian berubah.
- `internal/repository/mysql/mutations.go`.
- `internal/repository/memory/...` bila implementasi kontrak terkait berubah.
- `internal/handler/mutation.go` untuk meneruskan `u.CategoryID`.

Tidak perlu mengubah model/schema karena `Mutation.CategoryID` sudah ada.

### Regression test

- Buat dua kategori pada outlet yang sama.
- Buat PM kategori A dan PM kategori B pada outlet tersebut.
- Buat mutasi harian untuk kedua kategori.
- Panggil jalur riwayat harian sebagai PM A: hanya kategori A yang muncul dan total qty hanya kategori A.
- Ulangi untuk PM B.
- Pastikan mutasi outlet lain tidak muncul.
- Pastikan pencarian barcode, SKU/PLU, nama, tanggal, arah, pagination, dan total tetap konsisten.
- Pastikan `Period` tetap menghasilkan scope kategori yang sama.

Hasil yang diharapkan: seluruh jalur riwayat PM memakai batas `outlet_id` dan `category_id` dari user/session, bukan input bebas dari client.

## 3. B. Validasi outlet nonaktif

### Bukti kode aktual

- `internal/models/models.go:13-20`: `Outlet.IsActive bool` adalah representasi status outlet.
- `internal/models/models.go:31-44`: user memiliki `OutletID *uint` dan relasi `Outlet`.
- `internal/middleware/auth.go:28-44`: `LoadUser` memuat user berdasarkan session dan hanya memeriksa `u.IsActive`; status outlet tidak diperiksa.
- `internal/middleware/auth.go:89-99`: `RequireOutlet` hanya memeriksa user tidak nil dan `OutletID != nil`, lalu memanggil `c.Next()`.
- `internal/router/router.go:75-89`: seluruh group PM memakai `RequireRole(RolePM)` dan `RequireOutlet`; route GET dan POST PM berada di group tersebut.
- `internal/service/mutation.go:109-120`: lookup produk memvalidasi outlet dan kategori produk, tetapi tidak memeriksa status outlet.
- `internal/service/mutation.go:37-52`: update qty/edit history memeriksa role dan `OutletID`, bukan status outlet.

### Kesimpulan

Temuan **terkonfirmasi sebagai gap validasi**. Outlet yang tidak ditemukan maupun outlet nonaktif belum ditolak oleh `RequireOutlet` berdasarkan kode saat ini. Pemeriksaan outlet aktif saja juga tidak boleh dianggap sebagai seluruh otorisasi; service mutasi tetap harus mempertahankan scope user, outlet, kategori, ownership, dan tanggal transaksi.

### Respons yang direncanakan

Untuk user PM dengan outlet nil, outlet tidak ditemukan, atau outlet nonaktif, gunakan respons akses ditolak yang konsisten dan tidak membocorkan detail internal. Pesan dapat membedakan “belum ditempatkan” dan “outlet tidak aktif” hanya jika kebijakan UX/security menyetujuinya; status HTTP tetap perlu konsisten untuk route browser dan HTMX.

Perubahan minimal kemungkinan berada di `internal/middleware/auth.go`, dengan dependency repository outlet atau validasi relasi outlet yang sudah dimuat. Jika middleware perlu mengakses outlet, kontrak `repository.Outlets` dan wiring `router.New` mungkin terdampak. Jangan memindahkan seluruh authorization ke middleware tanpa mempertahankan validasi service.

### Regression test

Uji pada seluruh route PM, bukan hanya halaman utama:

1. User PM aktif + outlet aktif: GET `/mutasi`, lookup, history, dan POST create/update tetap berhasil sesuai policy.
2. User PM aktif + `OutletID == nil`: GET dan POST ditolak.
3. User PM aktif + outlet tidak ditemukan: GET dan POST ditolak.
4. User PM aktif + outlet `IsActive == false`: GET dan POST ditolak.
5. User nonaktif: session dibersihkan dan route ditolak oleh `LoadUser`/`RequireAuth`.
6. PM outlet A tidak dapat membaca atau mengubah mutasi outlet B.
7. PM kategori A tidak dapat lookup atau membaca data kategori B.
8. Manager/Admin tidak mendapatkan akses PM hanya karena memiliki outlet.
9. HTMX request menerima status/redirect yang sesuai ketika outlet tidak valid.

Hasil yang diharapkan: status outlet tidak aktif memblokir akses PM, tetapi tidak mengurangi atau menggantikan pemeriksaan role, session, outlet ownership, kategori, dan ownership transaksi di service/repository.

## 4. C. Unreachable code

### Bukti kode aktual

- `internal/handler/pages.go:71-74`: `Kelola` melakukan redirect ke `/kelola/outlet`, lalu `return`.
- `internal/handler/pages.go:75-123`: seluruh implementasi legacy combined page berada setelah return.
- `go vet ./...` pada baseline sebelumnya melaporkan `internal/handler/pages.go:76:2: unreachable code`.
- `internal/router/router.go:121-127`: route `/kelola`, `/kelola/outlet`, dan `/kelola/pm` memang sudah memiliki handler terpisah.

### Kesimpulan

Temuan **terkonfirmasi**. Kode setelah return tidak mungkin dijalankan pada invocation `Kelola`; ini bukan sekadar kode yang jarang dipakai. Route `/kelola` seharusnya tetap redirect sebagaimana perilaku aktual.

### Perubahan minimal yang direncanakan

Pada implementasi, hapus hanya blok legacy setelah `return` di `Kelola`, tanpa mengubah redirect, route, template, query handler outlet/PM, atau perilaku navigasi. Tidak perlu refactor besar.

File yang diperkirakan berubah: `internal/handler/pages.go` saja.

### Regression test

- `GET /kelola` tetap menghasilkan redirect ke `/kelola/outlet`.
- User non-admin tetap ditolak oleh route group sebelum handler.
- `GET /kelola/outlet` tetap memanggil halaman outlet.
- `GET /kelola/pm` tetap memanggil halaman PM.
- `go vet ./...` tidak lagi melaporkan unreachable code.
- `go test ./...` dan `go build ./...` tetap lulus.

## 5. Risiko regresi dan verifikasi

| Area | Risiko | Verifikasi |
|---|---|---|
| Filter harian | Query harian berubah scope atau total qty | Test dua kategori/outlet dan query/pagination |
| Repository contract | Memory/MySQL tidak konsisten | Unit test implementasi dan compile/build |
| Outlet aktif | Middleware butuh repository tambahan atau error DB | Test aktif/nonaktif/hilang dan failure-path |
| HTMX | Status error tidak diproses UI | Handler test dengan `HX-Request` |
| Kelola | Redirect berubah saat menghapus legacy | Route test `/kelola` dan `/kelola/outlet` |
| Data existing | Query baru mengabaikan snapshot category nil | Fixture kategori nil dan policy eksplisit |

Tidak ada test runtime/database yang dijalankan pada tahap read-only ini.

## 6. Perintah pengujian aman pada tahap implementasi

Perintah berikut aman jika tidak memakai server, migration, seeder, atau database aplikasi:

```text
go test ./...
go vet ./...
go build ./...
```

Untuk regression integration yang menyentuh MySQL, wajib memakai database test terisolasi sesuai `CHAT-STAGE-2B-TEST-AND-FOUNDATION-PLAN.md`. Jangan menjalankan `go run`, `server.exe`, `php`, migration, seeder, atau command yang membuka koneksi database sebelum guardrail test disetujui.

## 7. File yang diperkirakan berubah pada tahap implementasi

- `internal/handler/mutation.go` — meneruskan category scope ke riwayat harian.
- `internal/service/mutation.go` — menerapkan category scope pada `ByDate`.
- `internal/repository/repository.go` — hanya jika kontrak repository read berubah.
- `internal/repository/mysql/mutations.go` — filter kategori untuk query harian.
- `internal/repository/memory/...` — menjaga implementasi memory konsisten bila kontrak berubah.
- `internal/middleware/auth.go` — validasi outlet aktif/hilang untuk PM.
- `internal/router/router.go` — hanya jika middleware memerlukan wiring dependency outlet.
- `internal/handler/pages.go` — menghapus blok unreachable secara minimal.
- File test baru/yang ada untuk service, repository, middleware, dan route.

Tidak ada file-file tersebut yang diubah pada Tahap 3A.

## 8. Status perubahan

Audit ini hanya membaca kode aktual dan laporan acuan. Tidak ada source code, test, konfigurasi, dependency, database, migration, seeder, atau dokumentasi lama yang diubah. Satu-satunya file baru adalah laporan ini. Implementasi ditunda sampai ada persetujuan eksplisit.
