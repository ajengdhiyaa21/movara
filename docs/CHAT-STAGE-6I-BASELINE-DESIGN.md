# MOVARA Chat — Stage 6I: Minimal Test Database Baseline Design

Tanggal: 28 Oktober 2026  
Status: **DESIGN ONLY — BLOCKED UNTIL DATABASE VERIFICATION**

## 1. Scope dan sumber

Dokumen ini hanya merancang baseline minimal untuk database test terisolasi. Tidak ada SQL baseline yang dibuat atau dijalankan.

Sumber yang ditinjau:

- `docs/CHAT-STAGE-6H-BASELINE-AUDIT.md`
- `migrations/000001_create_chat_tables.up.sql`
- `internal/models/models.go`
- `internal/database/database.go`
- `internal/config/testdb_policy.go`
- `docs/CHAT-STAGE-6D-READONLY-VERIFICATION-PLAN.md`
- `docs/CHAT-STAGE-6E-MIGRATION-READINESS-REVIEW.md`

Tidak ada koneksi database, SQL, AutoMigrate, Seed, server, migration, atau integration test yang dijalankan.

## 2. Dua jenis baseline

### 2.1 Baseline minimal untuk repository chat

Tujuannya hanya menyediakan parent table agar `migrations/000001_create_chat_tables.up.sql` dapat dibuat dan repository chat dapat diuji secara persistence.

Kebutuhan langsungnya:

- database test yang terisolasi dan telah lolos policy;
- tabel `users`;
- kolom `users.id` yang kompatibel dengan seluruh foreign key migration chat;
- engine parent yang mendukung foreign key;
- privilege akun migration untuk membuat tabel, index, dan constraint;
- data fixture user sintetis untuk PM, Admin, dan skenario authorization yang diuji.

Migration chat kemudian membuat:

- `chat_conversations`;
- `chat_messages`;
- `chat_read_states`.

Untuk repository chat murni, tabel `outlets`, `product_categories`, `products`, dan `mutations` tidak dirujuk oleh migration chat.

### 2.2 Baseline untuk autentikasi dan alur aplikasi

Baseline ini lebih luas. Selain `users`, alur aplikasi dapat membutuhkan tabel yang dibuat oleh `database.Migrate`, antara lain:

- `outlets`;
- `product_categories`;
- `products`;
- `mutations`;
- `mutation_edit_logs`;
- `outlet_transfer_requests`;
- `audit_logs`.

Kebutuhan tepatnya harus ditentukan dari test yang akan dijalankan. Baseline penuh tidak boleh dibuat hanya karena model tersedia. `database.Migrate` bukan baseline versioned dan tidak boleh dipakai untuk menyiapkan test database.

## 3. Kontrak `users` dari migration chat

Migration saat ini memiliki tiga referensi ke `users.id`:

| Kolom child | Kolom parent | Sifat |
|---|---|---|
| `chat_conversations.pm_user_id` | `users.id` | `NOT NULL`, FK `RESTRICT` |
| `chat_messages.sender_user_id` | `users.id` | `NOT NULL`, FK `RESTRICT` |
| `chat_read_states.user_id` | `users.id` | `NOT NULL`, FK `RESTRICT` |

Migration mendefinisikan kolom ID chat sebagai `BIGINT UNSIGNED AUTO_INCREMENT`. Karena foreign key harus kompatibel, `users.id` harus memiliki ukuran dan signedness yang kompatibel dengan `BIGINT UNSIGNED`.

Hal berikut masih **BELUM TERVERIFIKASI**:

- tipe SQL aktual `users.id`;
- `UNSIGNED` pada `users.id`;
- engine tabel `users`;
- charset dan collation tabel/database;
- apakah key yang dirujuk benar-benar primary key atau unique key yang kompatibel;
- foreign key dan index aktual pada `users`;
- apakah tabel `users` berada pada database target yang benar.

Migration tidak merujuk `users.role`, `users.is_active`, `users.outlet_id`, atau `users.category_id`. Field tersebut tetap diperlukan oleh fixture/service authorization aplikasi, bukan oleh pembuatan FK chat secara langsung.

## 4. Perbandingan dengan `models.User`

Model `User` menggunakan `ID uint` dengan tag primary key. Field domain lainnya mencakup `Name`, `Username`, `PasswordHash`, `Role`, nullable `OutletID`, nullable `CategoryID`, `IsActive`, `CreatedAt`, dan `UpdatedAt`.

Model memberi kontrak aplikasi berikut:

- `Username` unik dan wajib;
- `Role` wajib dan di-index;
- `PasswordHash` wajib;
- `IsActive` wajib dengan default aktif;
- `OutletID` dan `CategoryID` nullable serta di-index.

Namun, model dan konfigurasi GORM tidak cukup untuk membuktikan seluruh schema SQL aktual. Secara khusus, mapping `uint` ke SQL, panjang/unsignedness ID, engine, collation, default aktual, serta constraint hasil AutoMigrate harus diverifikasi pada target test atau melalui baseline yang disetujui.

Karena migration chat sudah menetapkan `BIGINT UNSIGNED`, perbedaan dengan mapping `User.ID` harus diselesaikan sebelum migration dijalankan. Jangan mengubah migration atau model berdasarkan dugaan.

## 5. Apakah baseline minimal cukup?

### Cukup untuk repository chat persistence

Baseline minimal dapat cukup jika test hanya:

- membuat dan membaca conversation;
- menyimpan dan mengambil pesan;
- menguji unique `client_message_id`;
- menguji cursor pagination;
- menyimpan read state monotonic;
- memakai row user sintetis yang langsung dibuat oleh fixture.

### Tidak cukup untuk seluruh alur autentikasi aplikasi

Baseline minimal belum tentu cukup untuk test yang menjalankan login, middleware, handler aplikasi, atau pembuatan user melalui service existing. Alur tersebut dapat mengharapkan tabel/relasi outlet dan kategori, serta field user lengkap.

Rekomendasi: mulai dengan scope repository chat persistence yang eksplisit. Tambahkan parent schema lain hanya ketika test autentikasi/alur aplikasi benar-benar disetujui dan kebutuhannya dipetakan.

## 6. Prasyarat sebelum migration

Migration tetap **BLOCKED** sampai seluruh prasyarat berikut terpenuhi:

1. `TEST_DATABASE_URL` tersedia secara eksplisit dan `APP_ENV=test` digunakan.
2. Validasi menggunakan `ApprovedTestDatabasePolicy()` dan tidak fallback ke `DATABASE_URL` atau `.env` aplikasi.
3. Database target terisolasi dan identitas instance telah diverifikasi operator.
4. `users.id` aktual telah diperiksa dan kompatibel dengan FK `BIGINT UNSIGNED`.
5. Engine `users` mendukung foreign key.
6. Charset/collation target telah diverifikasi dan keputusan kompatibilitas disetujui.
7. Akun runtime test dan akun migration dipisahkan.
8. Akun runtime tidak diberi privilege DDL permanen.
9. Privilege migration minimum telah disetujui dan diverifikasi.
10. Tidak ada target ke database MOVARA yang sudah ada.
11. Error sistem MariaDB `#1034 - Index for table 'db' is corrupt` telah ditangani oleh operator/administrator database melalui prosedur resmi.

Error `#1034` adalah blocker operasional. Tahap ini tidak memperbaiki tabel sistem dan tidak menyarankan penghapusan file database, penghapusan `.ibd`, reinstall XAMPP, atau perbaikan spekulatif.

## 7. Fixture sintetis tanpa `database.Seed`

Fixture test harus:

- dijalankan hanya setelah database test lolos policy;
- dibuat khusus untuk test dan tidak dipanggil dari startup aplikasi;
- tidak memakai password default aplikasi;
- menggunakan identifier yang diketahui test, bukan data production atau hasil copy database MOVARA;
- membuat minimal Admin aktif, PM aktif, PM nonaktif, dan Manager bila authorization test membutuhkannya;
- menjaga foreign key dan unique constraint;
- memiliki cleanup/reset yang hanya dapat menargetkan database yang sudah tervalidasi.

`database.Seed` tidak digunakan karena bersifat startup-oriented, membuat data aplikasi yang lebih luas, dan memiliki perilaku default password yang tidak sesuai dengan fixture terisolasi.

## 8. Strategi reset database test

Reset hanya boleh dipertimbangkan setelah:

- target diverifikasi sebagai `movara_chat_test` pada instance yang disetujui;
- policy guardrail lulus;
- akun yang digunakan bukan akun production;
- operator menyetujui operasi reset;
- mekanisme reset memeriksa target kembali sebelum operasi destruktif.

Reset tidak boleh menyentuh database MOVARA, tidak boleh memakai wildcard target, dan tidak boleh dijalankan dari startup server. Jika belum ada prosedur reset yang tervalidasi, pilihan aman adalah membuat fixture idempotent atau menggunakan database test baru yang disetujui, bukan menghapus objek secara spekulatif.

## 9. Langkah berikutnya yang paling kecil dan aman

1. Operator menyelesaikan blocker MariaDB `#1034` melalui administrator yang berwenang.
2. Operator menjalankan pemeriksaan read-only Stage 6D untuk instance, target database, `users.id`, engine, charset/collation, dan privilege.
3. Pengembang dan operator menyetujui scope baseline repository chat minimal.
4. Setelah persetujuan, siapkan baseline parent `users` secara versioned dan terpisah dari migration chat.
5. Review hasil baseline secara statis, lalu jalankan migration chat hanya pada target test yang telah divalidasi.
6. Buat fixture sintetis chat dan jalankan integration test secara eksplisit.

Tidak ada langkah di atas yang dijalankan pada Stage 6I.

## 10. Status akhir

- Desain baseline minimal: tersedia sebagai rancangan.
- SQL baseline: belum dibuat.
- Database test: belum dianggap siap.
- Migration chat: belum boleh dijalankan.
- Blocker utama: struktur `users` dan target database belum terverifikasi, privilege belum terbukti, serta error MariaDB `#1034` belum terselesaikan.

Tidak ada file selain laporan ini yang dibuat atau diubah. Tidak ada koneksi database, SQL, migration, AutoMigrate, Seed, server, atau integration test yang dijalankan.
