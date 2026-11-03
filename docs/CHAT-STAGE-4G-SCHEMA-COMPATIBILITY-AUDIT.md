# MOVARA — Tahap 4G: Audit Kompatibilitas Skema Database Test

Tanggal audit: 27 Oktober 2026  
Mode: read-only.

## 1. File yang dibaca

- `docs/CHAT-STAGE-4F-TEST-DATABASE-SETUP-PLAN.md`
- `internal/database/database.go`
- `internal/models/models.go`
- `internal/repository/mysql/mysql.go`
- `internal/repository/mysql/users.go`
- `internal/repository/mysql/products.go`
- `internal/repository/mysql/outlets.go`
- `internal/repository/mysql/mutations.go`
- `internal/repository/mysql/audits.go`
- `.env.example` tanpa menampilkan credential.

## 2. Model dan tabel aktual

Nama tabel di bawah mengikuti konvensi default GORM dan perlu diverifikasi saat migration test disetujui.

### `outlets` — `internal/models/models.go:13-20`

`id` primary key; `code` varchar size 20 unique not null; `name` size 100 not null; `type` size 10 not null; `is_active` bool not null default true; `created_at`.

### `product_categories` — `models.go:22-29`

`id` primary key; `name` size 100 unique not null; `slug` size 100 unique not null; `is_active` bool not null default true; `created_at`, `updated_at`.

### `users` — `models.go:31-44`

`id` primary key; `name` size 100 not null; `username` size 50 unique not null; `password_hash` size 100 not null; `role` size 20 not null indexed; nullable indexed `outlet_id` and `category_id`; `is_active` bool not null default true; timestamps. `Outlet` dan `Category` adalah relasi GORM tanpa foreign key tag eksplisit.

### `products` — `models.go:47-59`

`id` primary key; `barcode` size 20 not null; `code` size 30 indexed not null; `name` size 255 not null; `is_active` bool not null default true; `outlet_id` uint not null; nullable indexed `category_id`; timestamps. Unique index gabungan eksplisit `idx_product_outlet_barcode`: outlet priority 1, barcode priority 2.

### `mutations` — `models.go:61-81`

`id` primary key; nullable indexed `product_id`; `barcode` size 20 not null indexed; nullable indexed `code`; `product_name` size 255 not null; nullable indexed snapshot `category_id`; `direction` size 20 not null default `counter_to_market` indexed; `status` size 20 not null default `approved` indexed; `qty` int not null; nullable `ba_name` size 100; `outlet_id` and `user_id` uint not null indexed; `created_at` indexed. Index gabungan `idx_mut_outlet_date`: outlet priority 1, created_at priority 2.

### `mutation_edit_logs` — `models.go:83-91`

`id` primary key; `mutation_id` dan `user_id` uint not null indexed; `old_qty`, `new_qty` int not null; `created_at` indexed; relasi `User`.

### `outlet_transfer_requests` — `models.go:101-110`

`id` primary key; `user_id`, `from_outlet_id`, `to_outlet_id` uint not null; `status` size 20 not null default `pending` indexed; `requested_at`; nullable `decided_by`, `decided_at`.

### `audit_logs` — `models.go:112-119`

`id` primary key; nullable indexed `user_id`, `outlet_id`; `action` size 50 not null; `detail` text; `created_at` indexed.

Foreign key constraint database tidak diberi tag eksplisit pada relasi-relasi tersebut; keberadaannya tidak boleh diasumsikan tanpa verifikasi schema hasil migration.

## 3. Migration dan repository

`internal/database/database.go:14-22` melakukan AutoMigrate outlet/kategori. Baris 26-40 memiliki kompatibilitas legacy: menambah nullable `products.outlet_id`, menghapus index legacy tertentu, lalu mengisi nilai kosong dengan outlet pertama. Baris 42-49 AutoMigrate model lain.

Migration SQL berversi dapat merepresentasikan schema, tetapi backfill legacy bukan operasi CREATE TABLE biasa. Ia harus menjadi langkah migration/backfill terpisah dan diuji dengan dataset legacy. Tidak ada perubahan schema yang dilakukan pada audit ini.

Repository MySQL:

- `users.go:8-22`: lookup user dan preload outlet/category.
- `products.go:8-13`: barcode + outlet + active.
- `outlets.go:8-14`: hanya outlet aktif.
- `mutations.go:11-75`: edit qty transaction, list/filter tanggal/outlet/kategori, search, total, preload relasi.
- `audits.go:3-5`: insert audit.

Regression mutasi membutuhkan semua tabel utama; transfer table hanya untuk test transfer. Belum ada model/tabel chat, sehingga schema chat tidak boleh ditebak dari kode saat ini.

## 4. Charset, collation, dan pencarian

`.env.example:9` hanya membuktikan DSN aplikasi menyebut `charset=utf8mb4`. Tidak ada deklarasi collation pada model atau migration.

Case handling memakai `strings.ToLower` pada `internal/service/auth.go:19-24`, memory query, dan service mutation. Repository MySQL memakai `LIKE` (`internal/repository/mysql/mutations.go:57-63`), handler import memakai `LOWER(name) = LOWER(?)`, dan beberapa query memakai `ORDER BY`.

Charset `utf8mb4` adalah kebutuhan yang didukung bukti. Collation final, case sensitivity, accent sensitivity, dan dampaknya terhadap unique/index/LIKE/sorting belum dapat ditentukan. **NEEDS APPROVAL:** pilihan collation dan aturan perbandingan string. Kandidat collation tidak boleh dianggap disetujui.

Risiko: hasil MySQL dapat berbeda dari memory repository pada casing, uniqueness username/name/slug, pencarian, dan sorting.

## 5. Privilege, fixture, dan skenario test

Runtime test kandidat Tahap 4F adalah SELECT/INSERT/UPDATE/DELETE pada database test saja, tanpa `GRANT OPTION` atau `*.*`. Privilege migration DDL harus terpisah dari runtime dan belum disetujui.

Fixture minimal perlu outlet aktif/nonaktif, kategori, Admin/Manager/PM, produk aktif/nonaktif dengan barcode/code berawalan nol, mutation dua kategori pada outlet sama, edit log, audit log, serta transfer request bila diuji. Fixture harus terpisah dari `database.Seed`, deterministik, scoped pada database test, dan tidak dijalankan startup.

## 6. Risiko kompatibilitas

1. Nama tabel/index default GORM perlu diverifikasi saat migration.
2. Foreign key tidak eksplisit dalam tag.
3. Backfill `outlet_id` bergantung pada outlet pertama dan mengubah data.
4. Collation memengaruhi uniqueness, LIKE, LOWER, dan sorting.
5. `uint`, bool, time, default, nullable, dan index harus diuji pada MySQL aktual.
6. Unique product adalah outlet+barcode, bukan barcode global.
7. Outlet repository hanya mengembalikan outlet aktif.

## 7. NEEDS APPROVAL dan rekomendasi

- Charset/collation final dan aturan case/accent sensitivity.
- Nama/index/foreign-key strategy migration berversi.
- Tool migration dan akun DDL terpisah.
- Strategi fixture/reset tanpa menyentuh schema non-test.
- Prosedur backfill legacy `products.outlet_id`.
- Schema chat belum boleh dirancang dari asumsi model yang belum ada.

Urutan berikutnya: setujui collation; siapkan baseline migration pada database test kosong; uji schema/constraint/index; uji backfill terpisah; siapkan fixture regression mutasi; verifikasi grants; baru masuk tahap desain schema chat.

## 8. Pernyataan perubahan

Tidak ada source code, konfigurasi, dependency, schema, database, user, privilege, migration, seeder, atau laporan lama yang diubah. Tidak ada koneksi MySQL, SQL, server, AutoMigrate, Seed, migration, atau integration test yang dijalankan. Satu-satunya file baru adalah laporan ini.
