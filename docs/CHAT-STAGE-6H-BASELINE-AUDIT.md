# MOVARA Chat — Stage 6H: Baseline Schema Audit

Tanggal: 28 Oktober 2026  
Mode: audit read-only statis.

## 1. Kesimpulan

Baseline SQL schema aplikasi MOVARA **belum tersedia**. Repository hanya memiliki:

- `internal/database/database.go`, yang membuat/mengubah schema melalui GORM `AutoMigrate` dan operasi kompatibilitas legacy;
- `internal/database/seed.go`, yang memasukkan data contoh;
- `migrations/000001_create_chat_tables.up.sql`, yang hanya membuat tabel chat dan mengasumsikan tabel `users` sudah ada.

Migration chat tidak dapat dijalankan pada database kosong karena memiliki foreign key ke `users.id`. Struktur aktual database belum boleh dianggap sama dengan model sampai diverifikasi pada database test yang benar.

## 2. Sumber yang dibaca

- `internal/models/models.go`
- `internal/database/database.go`
- `internal/database/seed.go`
- `cmd/server/main.go`
- `internal/config/config.go`
- `migrations/000001_create_chat_tables.up.sql`
- `internal/repository/mysql/chat.go`
- `docs/CHAT-STAGE-6C-MIGRATION-PREP-REPORT.md`
- `docs/CHAT-STAGE-6D-READONLY-VERIFICATION-PLAN.md`
- `docs/CHAT-STAGE-6E-MIGRATION-READINESS-REVIEW.md`

Tidak ada koneksi database, SQL, AutoMigrate, Seed, server, migration, atau integration test yang dijalankan.

## 3. Struktur `users` yang didukung model

`models.User` memiliki field berikut:

| Field model | Tipe Go | Informasi schema dari tag |
|---|---|---|
| `ID` | `uint` | primary key |
| `Name` | `string` | varchar 100, not null |
| `Username` | `string` | varchar 50, unique, not null |
| `PasswordHash` | `string` | varchar 100, not null |
| `Role` | `models.Role` | varchar 20, indexed, not null |
| `OutletID` | `*uint` | nullable, indexed |
| `CategoryID` | `*uint` | nullable, indexed |
| `IsActive` | `bool` | not null, default true |
| `CreatedAt` | `time.Time` | timestamp convention GORM |
| `UpdatedAt` | `time.Time` | timestamp convention GORM |

Relasi model:

- `User.Outlet` mengarah secara konseptual ke `Outlet`;
- `User.Category` mengarah secara konseptual ke `ProductCategory`.

Tag model tidak menuliskan tipe SQL eksplisit untuk `uint`, tidak menuliskan nama tabel, dan tidak menuliskan foreign key constraint secara manual. Mapping aktual hasil GORM dan constraint yang dibuat AutoMigrate tetap **BELUM TERVERIFIKASI**.

## 4. Tabel minimum untuk migration chat

### Dependency database langsung

Migration `migrations/000001_create_chat_tables.up.sql` hanya merujuk langsung ke:

1. `users.id` sebagai parent untuk `chat_conversations.pm_user_id`;
2. `users.id` sebagai parent untuk `chat_messages.sender_user_id`;
3. `users.id` sebagai parent untuk `chat_read_states.user_id`.

Tabel `outlets`, `product_categories`, `products`, `mutations`, dan tabel lain tidak dirujuk oleh migration chat.

### Tabel yang dibuat migration chat

Migration membuat:

- `chat_conversations`;
- `chat_messages`;
- `chat_read_states`.

Relasi internal:

- `chat_messages.conversation_id → chat_conversations.id`;
- `chat_read_states.conversation_id → chat_conversations.id`.

Semua FK menggunakan `ON DELETE RESTRICT` dan `ON UPDATE RESTRICT`.

### Minimum untuk menjalankan SQL migration

Secara struktural, database target membutuhkan:

- schema target yang benar-benar terisolasi;
- tabel `users` yang sudah ada;
- `users.id` dengan tipe, signedness, dan ukuran yang kompatibel dengan `BIGINT UNSIGNED`;
- engine parent yang mendukung foreign key, umumnya InnoDB;
- hak akun migration untuk membuat tabel, index, dan foreign key.

Kondisi tersebut belum terbukti pada database test.

### Minimum untuk integration test chat yang memakai autentikasi aplikasi

Walaupun tidak diperlukan oleh FK migration chat, fixture authorization kemungkinan memerlukan data pada `users` dengan:

- user Admin aktif;
- user PM aktif;
- user PM nonaktif;
- user Manager untuk pengujian penolakan role;
- `username`, `password_hash`, `role`, dan `is_active` yang valid.

Jika test membuat user PM dengan outlet/kategori, tabel `outlets` dan `product_categories` juga diperlukan sebagai data pendukung. Ini adalah kebutuhan fixture/application flow, bukan dependency langsung migration chat. Jangan membuat tabel tersebut berdasarkan asumsi tanpa baseline yang disetujui.

## 5. Tabel aplikasi yang dibuat oleh AutoMigrate

`database.Migrate` secara aktual menjalankan:

1. `AutoMigrate(&models.Outlet{})`;
2. `AutoMigrate(&models.ProductCategory{})`;
3. kompatibilitas `products.outlet_id`, drop index tertentu, dan backfill ke outlet pertama bila kondisi terpenuhi;
4. `AutoMigrate` untuk `User`, `Product`, `Mutation`, `MutationEditLog`, `OutletTransferRequest`, dan `AuditLog`.

Dengan demikian, model schema aplikasi yang tersedia mencakup setidaknya:

- `outlets`;
- `product_categories`;
- `users`;
- `products`;
- `mutations`;
- `mutation_edit_logs`;
- `outlet_transfer_requests`;
- `audit_logs`.

Urutan, tipe SQL aktual, index default, foreign key hasil AutoMigrate, engine, charset, dan collation belum dibuktikan tanpa database.

## 6. Baseline SQL migration yang sudah ada

Tidak ditemukan baseline SQL untuk tabel aplikasi existing. Satu-satunya file SQL adalah migration chat:

`migrations/000001_create_chat_tables.up.sql`

File tersebut bukan baseline karena:

- tidak membuat `users`;
- tidak membuat `outlets` atau `product_categories`;
- mengasumsikan `users` telah tersedia;
- tidak merepresentasikan operasi kompatibilitas legacy `products.outlet_id`;
- tidak menetapkan charset/collation.

## 7. Risiko AutoMigrate untuk database test

Menggunakan `database.Migrate`/AutoMigrate untuk baseline test berisiko karena:

- schema dibuat berdasarkan perilaku runtime GORM, bukan migration versioned yang immutable;
- operasi `ALTER TABLE`, drop index, dan backfill dapat mengubah schema/data;
- backfill memilih outlet pertama berdasarkan ID, yang belum tentu mapping bisnis yang benar;
- hasil FK, index, default, dan tipe dapat berbeda dari migration SQL chat;
- startup tidak memisahkan target aplikasi dan target test secara otomatis;
- perubahan model di masa depan dapat mengubah schema tanpa review migration;
- tidak ada catatan versi schema yang setara dengan migration berversi.

AutoMigrate tidak boleh dipanggil untuk menyiapkan database test chat.

## 8. Risiko Seed untuk database test

`cmd/server/main.go` menjalankan `database.Seed` setelah koneksi dan migration. Seed berisiko karena:

- bergantung pada startup aplikasi;
- memakai password default jika `ADMIN_PASSWORD` kosong;
- membuat/mengubah data contoh akun, outlet, kategori, dan produk;
- tidak merupakan fixture integration test yang terisolasi dan deterministik untuk skenario chat;
- dapat menambahkan data yang tidak diperlukan atau mengganggu test;
- dapat menyentuh database yang salah jika konfigurasi target salah.

`Seed` tidak boleh digunakan sebagai baseline atau fixture integration test chat.

## 9. Rekomendasi menyiapkan baseline tanpa menyentuh database MOVARA

1. Pertahankan database aplikasi sebagai sumber yang tidak disentuh.
2. Verifikasi manual target `movara_chat_test`, instance, `users.id`, engine, charset/collation, dan privilege sesuai Stage 6D/6G.
3. Tentukan dan setujui scope baseline: apakah hanya parent `users` minimal atau seluruh schema aplikasi untuk test harness.
4. Jika test hanya memerlukan persistence chat, sediakan baseline versioned terpisah yang membuat parent `users` minimal sesuai kontrak yang disetujui, lalu jalankan migration chat.
5. Jika test memakai login/authorization aplikasi, baseline harus mencakup tabel dan relasi yang benar-benar dibutuhkan alur tersebut, termasuk `outlets`/`product_categories` bila fixture user memerlukannya.
6. Buat fixture chat sintetis terpisah dari `database.Seed` dan tanpa password default.
7. Jalankan migration dan fixture hanya pada database test setelah target lolos policy dan approval.
8. Verifikasi metadata hasil migration sebelum integration test.

Pembuatan baseline parent `users` belum dilakukan karena itu memperluas scope dari migration chat dan memerlukan keputusan schema yang belum disetujui.

## 10. Keputusan yang masih memerlukan persetujuan

- Apakah baseline test akan mencakup hanya tabel parent yang diperlukan chat atau seluruh schema aplikasi.
- Definisi SQL versioned untuk `users` jika database test kosong.
- Apakah `outlets` dan `product_categories` termasuk baseline atau hanya fixture/relasi test tertentu.
- Tipe SQL final `users.id` dan strategi foreign key.
- Charset/collation database test.
- Engine, index, default, dan foreign key baseline.
- Akun migration dan privilege DDL minimum.
- Tool/command migration yang dijalankan terpisah dari startup.
- Strategi fixture dan cleanup tanpa `Seed`.

## 11. Kesimpulan

Migration chat sudah memiliki dependency langsung yang jelas, yaitu tabel `users` dengan `id` kompatibel. Namun, source code tidak membuktikan struktur database aktual, dan repository belum memiliki baseline SQL schema aplikasi. Status persiapan tetap **BLOCKED** sampai baseline scope dan struktur parent disetujui serta database test diverifikasi manual.

Tidak ada file selain laporan ini yang dibuat atau diubah. Tidak ada database MOVARA, database test, SQL, server, AutoMigrate, Seed, migration, atau integration test yang dijalankan.
