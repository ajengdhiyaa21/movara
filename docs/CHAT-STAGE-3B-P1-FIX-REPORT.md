# MOVARA — Tahap 3B: Laporan Implementasi Perbaikan P1

Tanggal: 27 Oktober 2026  
Scope: tiga perbaikan P1 dari Tahap 3A; tanpa implementasi chat.

## 1. File yang diubah

- `internal/handler/mutation.go`
- `internal/service/mutation.go`
- `internal/repository/repository.go`
- `internal/repository/mysql/mutations.go`
- `internal/repository/memory/memory.go`
- `internal/middleware/auth.go`
- `internal/router/router.go`
- `internal/handler/pages.go`
- `internal/middleware/auth_test.go` (baru)
- `internal/service/mutation_test.go` (baru)

Perubahan lokal lain yang sudah ada, termasuk `docs/PRD-MOVARA.md` dan dokumen chat sebelumnya, dipertahankan dan tidak ditimpa.

## 2. Ringkasan perubahan

### Filter kategori riwayat harian PM

`MutationService.ByDate` sekarang menerima `categoryID` dari user terautentikasi. `historyPage` meneruskan `u.CategoryID`; tidak ada category ID yang dibaca dari input client sebagai sumber otorisasi.

Kontrak `ListByOutletBetween` diperluas dengan `categoryID *uint`. Implementasi MySQL menambahkan `WHERE category_id = ?` bila category ID tersedia. Implementasi memory menggunakan aturan yang sama. Batas outlet dan rentang waktu tetap diterapkan. Jalur `Today` dan `DuplicateToday` tetap memakai category nil sehingga perilaku internal non-PM tersebut tidak berubah.

Data mutation dengan `CategoryID` kosong tidak cocok ketika filter kategori PM aktif; ini konsisten dengan pola `FindProductForUser` yang menolak produk tanpa kategori untuk PM berkategori.

### Validasi outlet nonaktif

`RequireOutlet` sekarang menerima `repository.Outlets` dan memverifikasi outlet user:

- tanpa `OutletID` ditolak;
- error pembacaan repository ditolak;
- outlet tidak ditemukan ditolak;
- outlet nonaktif ditolak;
- outlet aktif diteruskan.

Router meneruskan `repos.Outlets` ke middleware group PM. Pemeriksaan role, session, kategori, ownership, dan scope service tetap dipertahankan.

Catatan: implementasi MySQL `Outlets.List()` saat ini hanya mengembalikan outlet aktif. Akibatnya outlet nonaktif pada runtime MySQL dikategorikan sebagai tidak ditemukan dan tetap ditolak secara fail-closed. Unit test fake mencakup status nonaktif eksplisit dan tidak ditemukan.

### Unreachable code

Blok legacy setelah redirect dan `return` pada `PageHandler.Kelola` dihapus. Redirect `/kelola` ke `/kelola/outlet`, route, middleware, serta handler outlet/PM tidak diubah.

## 3. Regression test yang ditambahkan

### `internal/service/mutation_test.go`

`TestByDateFiltersPMCategoryWithinOutlet` membuat dua mutation dengan kategori berbeda pada outlet yang sama menggunakan memory repository. Test memanggil `ByDate` dengan satu category ID dan memastikan hanya kategori tersebut yang keluar.

### `internal/middleware/auth_test.go`

- outlet aktif diteruskan;
- outlet nonaktif ditolak 403;
- outlet tidak ditemukan ditolak 403;
- error repository ditolak 403.

Test tidak membuka koneksi database.

## 4. Perintah yang dijalankan dan hasil aktual

Sebelum verifikasi, source target diformat dengan `gofmt`.

```text
go test ./...
```

Status: **berhasil**. Package middleware dan service termasuk test baru lulus; package lain juga selesai tanpa error.

```text
go vet ./...
```

Status: **berhasil**. Temuan unreachable code sebelumnya tidak muncul lagi.

```text
go build ./...
```

Status: **berhasil**.

Tidak ada server, migration, seeder, AutoMigrate, atau command yang membuka koneksi database dijalankan.

## 5. Test yang belum dijalankan

- Integration test MySQL untuk query kategori belum dijalankan karena database test terisolasi dan guardrail fail-closed belum tersedia.
- Test route HTTP penuh untuk seluruh route PM belum dijalankan; test middleware unit mencakup keputusan outlet.
- Browser/HTMX E2E belum dijalankan.
- Tidak ada klaim bahwa data production atau database development telah diuji.

## 6. Risiko dan pekerjaan tersisa

1. `Outlets.List()` MySQL hanya mengembalikan outlet aktif, sehingga status outlet nonaktif tidak dapat dibedakan dari ID yang tidak ditemukan pada pesan. Keduanya tetap aman karena ditolak.
2. Perubahan signature repository dapat berdampak pada implementasi eksternal di luar repository yang terdeteksi; `go test` dan `go build` lokal lulus.
3. Test service memakai memory repository, bukan MySQL query planner/schema aktual.
4. CSRF token, database test, migration berversi, rate limiting, dan fitur chat belum dikerjakan.
5. Validasi outlet middleware bukan pengganti validasi service; service tetap harus memeriksa role, outlet, kategori, ownership, dan aturan tanggal.

## 7. Konfirmasi perubahan terlarang

- Database tidak diubah.
- Schema tidak diubah.
- Migration dan seeder tidak dijalankan.
- Konfigurasi tidak diubah.
- `DATABASE_URL` tidak diubah.
- Dependency tidak ditambahkan.
- Server tidak dijalankan.
- Fitur chat dan CSRF token tidak diimplementasikan.

Laporan ini dibuat setelah implementasi dan verifikasi aman selesai.
