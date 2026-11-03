# MOVARA Chat — Stage 6C: Migration dan Integration Test Preparation Report

Tanggal: 28 Oktober 2026  
Status: persiapan selesai; migration dan integration test belum dijalankan.

## File dibuat

- `migrations/000001_create_chat_tables.up.sql`
- `internal/integration/chat_database_test.go`
- `docs/CHAT-STAGE-6C-MIGRATION-PREP-REPORT.md`

Tidak ada dependency baru, konfigurasi, `.env`, privilege, atau startup code yang diubah.

## Migration yang disiapkan

Migration hanya membuat tiga tabel chat:

- `chat_conversations` dengan unique `pm_user_id`;
- `chat_messages` dengan unique `(conversation_id, client_message_id)`;
- `chat_read_states` dengan unique `(conversation_id, user_id)`.

Constraint dan index yang disiapkan:

- foreign key ke `users.id`, conversation, dan sender/user;
- `ON DELETE RESTRICT` dan `ON UPDATE RESTRICT`;
- index histori `(conversation_id, id)`;
- index sender dan timestamp;
- index read state per user/conversation;
- timestamp `DATETIME(6)`;
- engine `InnoDB` untuk dukungan foreign key.

Tipe foreign key menggunakan `BIGINT UNSIGNED`, konsisten dengan pemetaan MySQL yang diharapkan untuk field `uint` pada model existing, tetapi tipe aktual schema `users.id` tetap harus diverifikasi sebelum migration dijalankan.

Tidak ada down migration. Menghapus tiga tabel akan bersifat destruktif terhadap histori chat dan tidak aman sebagai rollback default. Pemulihan harus memakai backup/forward-fix yang disetujui.

Migration tidak menetapkan charset atau collation. Ini disengaja karena collation database test belum diverifikasi. Migration menjadi **BLOCKED** sampai operator menyetujui dan memverifikasi charset/collation target.

## Kerangka integration test

`internal/integration/chat_database_test.go` memakai build tag `integration`, sehingga tidak masuk `go test ./...` biasa.

Saat dijalankan secara eksplisit, test:

1. mewajibkan `APP_ENV=test`;
2. mewajibkan `TEST_DATABASE_URL`;
3. memvalidasi target dengan `ApprovedTestDatabasePolicy()`;
4. tidak membaca `DATABASE_URL` atau `.env` sebagai fallback;
5. baru memanggil `sql.Open` setelah validasi target lulus;
6. melakukan `Ping` hanya pada eksekusi eksplisit test integration.

Test ini belum menjalankan migration, AutoMigrate, Seed, fixture, atau skenario chat. Ia hanya menjadi entry point fail-closed untuk tahap integration berikutnya.

## Review kompatibilitas statis

- Nama tabel/kolom sesuai model Stage 6A dan query repository MySQL Stage 6B.
- `client_message_id` memakai composite unique constraint per conversation.
- Pagination didukung index `(conversation_id, id)`.
- Read state memakai unique conversation/user.
- `DATETIME(6)` sesuai rancangan timestamp presisi.
- `InnoDB` dipilih karena membutuhkan foreign key.
- `ON DELETE RESTRICT` mencegah penghapusan user/conversation yang masih dirujuk.
- Repository MySQL masih harus diuji terhadap duplicate-key nyata dari driver MariaDB.

Belum diverifikasi: engine dan tipe aktual `users.id`, foreign key enforcement, collation, panjang efektif `VARCHAR(3000)`, serta privilege akun test/migration.

## Verifikasi yang dijalankan

Belum ada SQL, migration, koneksi, `sql.Open`, `Ping`, server, AutoMigrate, Seed, atau integration test yang dijalankan.

Setelah file dibuat, verifikasi statis project perlu dilakukan dengan `go test ./...`, `go vet ./...`, dan `go build ./...`. Test integration bertag `integration` tidak termasuk dalam perintah default tersebut dan tidak boleh dijalankan pada tahap ini.

## Blocker sebelum eksekusi

1. Verifikasi database test dan target instance.
2. Verifikasi tipe `users.id` pada schema target.
3. Verifikasi engine yang mendukung foreign key.
4. Persetujuan charset/collation database test.
5. Verifikasi akun runtime hanya memiliki DML pada schema test.
6. Akun migration harus terpisah dan memiliki DDL minimum.
7. Persetujuan untuk menjalankan tool migration secara eksplisit.
8. Fixture dan integration scenario harus disiapkan setelah migration berhasil.

## Konfirmasi keselamatan

Tidak ada database, akun, privilege, schema, data, SQL, koneksi, migration, AutoMigrate, Seed, server, atau integration test yang disentuh. Migration hanya dibuat sebagai file untuk review dan kerangka test tidak dijalankan.
