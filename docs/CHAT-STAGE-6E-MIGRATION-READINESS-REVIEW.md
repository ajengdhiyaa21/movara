# MOVARA Chat — Stage 6E: Migration Readiness Review

Tanggal: 28 Oktober 2026  
Keputusan akhir: **BLOCKED** untuk menjalankan migration; **READY FOR MANUAL VERIFICATION** untuk pemeriksaan operator Stage 6D.

## 1. Scope pemeriksaan

Pemeriksaan dilakukan secara statis terhadap:

- `migrations/000001_create_chat_tables.up.sql`;
- `docs/CHAT-STAGE-6D-READONLY-VERIFICATION-PLAN.md`;
- `internal/integration/chat_database_test.go`;
- `internal/config/testdb.go`;
- `internal/config/testdb_policy.go`;
- `internal/models/models.go`;
- `internal/repository/mysql/chat.go`.

Tidak ada SQL, koneksi, migration, server, AutoMigrate, Seed, atau integration test yang dijalankan.

## 2. Hasil pemeriksaan migration

### Tipe foreign key

Migration menggunakan `BIGINT UNSIGNED` untuk:

- `chat_conversations.pm_user_id → users.id`;
- `chat_messages.conversation_id → chat_conversations.id`;
- `chat_messages.sender_user_id → users.id`;
- `chat_read_states.conversation_id → chat_conversations.id`;
- `chat_read_states.user_id → users.id`.

Model Go memakai `uint` untuk `User.ID` dan ID model chat. Tag model tidak menuliskan tipe SQL secara eksplisit. `BIGINT UNSIGNED` konsisten dengan pemetaan MySQL yang diharapkan untuk `uint`, tetapi tipe aktual `users.id` di database belum dapat dibuktikan dari source code. Verifikasi `INFORMATION_SCHEMA.COLUMNS` dari Stage 6D tetap wajib.

Status: **BELUM TERVERIFIKASI** pada database nyata.

### Nama kolom dan constraint

Secara statis konsisten:

- `pm_user_id` unik untuk satu conversation per PM;
- `client_message_id` diwajibkan dan unik bersama `conversation_id`;
- read state unik bersama `conversation_id` dan `user_id`;
- FK conversation, sender, dan user mengarah ke tabel/kolom yang sesuai;
- seluruh FK menggunakan `ON DELETE RESTRICT ON UPDATE RESTRICT`;
- primary key memakai auto-increment;
- timestamp menggunakan `DATETIME(6)`;
- `body` memiliki kapasitas 3.000 karakter;
- index `(conversation_id, id)` tersedia untuk pagination.

### Engine dan collation

Semua tabel migration menggunakan `ENGINE=InnoDB`, yang sesuai kebutuhan foreign key.

Migration sengaja tidak menetapkan charset atau collation. Ini mencegah keputusan collation spekulatif, tetapi berarti target database harus sudah memiliki charset/collation yang telah diverifikasi dan disetujui sebelum migration dijalankan.

Status engine parent `users`, charset, dan collation target: **BELUM TERVERIFIKASI**.

## 3. Kompatibilitas sintaks MariaDB 10.4

Berdasarkan pemeriksaan sintaks statis, statement yang digunakan adalah fitur umum yang kompatibel dengan MariaDB 10.4:

- `CREATE TABLE`;
- `BIGINT UNSIGNED AUTO_INCREMENT`;
- `DATETIME(6)`;
- `VARCHAR`;
- named `CONSTRAINT` dan foreign key;
- `ON DELETE/UPDATE RESTRICT`;
- `ENGINE=InnoDB`;
- `PRIMARY KEY`, `UNIQUE KEY`, dan `KEY`.

Ini adalah penilaian sintaks, bukan hasil eksekusi terhadap MariaDB aktif. Parser/server belum dijalankan.

## 4. Risiko kegagalan parsial

Migration membuat tiga tabel dalam beberapa statement DDL terpisah. Jika statement kedua atau ketiga gagal, tabel sebelumnya dapat sudah terbuat karena DDL MariaDB tidak boleh diasumsikan atomic sebagai satu migration penuh.

Contoh penyebab kegagalan:

- `users` belum ada di database target;
- tipe `users.id` berbeda signedness atau ukuran;
- engine parent tidak mendukung FK;
- database target salah atau tidak terpilih;
- tabel chat sudah ada sebagian;
- akun migration tidak memiliki DDL;
- collation/charset atau panjang index tidak kompatibel;
- nama constraint/index bentrok dengan objek existing.

Karena tidak ada down migration destruktif, operator harus menghentikan proses dan melakukan review/forward-fix terkontrol jika terjadi kegagalan. Jangan menjalankan ulang secara membabi buta dan jangan menghapus tabel chat otomatis.

## 5. Cara menjalankan secara eksplisit

Setelah pemeriksaan Stage 6D lulus dan persetujuan diberikan:

1. pastikan sesi/tool menunjuk ke `movara_chat_test` yang benar;
2. gunakan akun migration terpisah dari runtime `movara_test`;
3. jalankan migration melalui `golang-migrate/migrate` atau runner SQL berversi yang disetujui;
4. arahkan runner hanya ke `TEST_DATABASE_URL` yang lolos `ApprovedTestDatabasePolicy()`;
5. jangan menjalankan `cmd/server/main.go`;
6. jangan memanggil `AutoMigrate` atau `Seed`;
7. setelah selesai, lakukan pemeriksaan metadata tabel, index, FK, dan migration version;
8. hentikan proses jika target, privilege, atau schema tidak sesuai.

Tool migration belum dijalankan dan belum ada dependency migration baru pada repository.

## 6. Review integration test fail-closed

`internal/integration/chat_database_test.go` memiliki build tag `integration`, sehingga tidak masuk test default. Saat dijalankan eksplisit, kerangka tersebut:

- menolak `APP_ENV` selain `test`;
- menolak `TEST_DATABASE_URL` kosong;
- memvalidasi URL dengan `ApprovedTestDatabasePolicy()`;
- tidak membaca `DATABASE_URL` sebagai fallback;
- baru membuka `sql.DB` setelah validasi target berhasil;
- melakukan `Ping` hanya pada eksekusi integration test eksplisit.

Status: guardrail target sudah tersedia secara statis. Namun test belum membuktikan schema, privilege, migration version, atau perilaku MariaDB karena belum boleh dijalankan.

## 7. Blocker sebelum migration

Migration tetap **BLOCKED** sampai operator menyelesaikan:

1. konfirmasi instance MariaDB dan database aktif adalah target test;
2. verifikasi database `movara_chat_test` terlihat dengan hak metadata yang cukup;
3. verifikasi tabel `users` ada;
4. verifikasi `users.id` kompatibel dengan `BIGINT UNSIGNED`;
5. verifikasi engine `users` mendukung foreign key;
6. verifikasi charset dan collation target;
7. konfirmasi tabel chat belum ada atau ada rencana penanganan existing object;
8. verifikasi akun migration terpisah dan memiliki DDL minimum;
9. verifikasi runtime `movara_test` tidak memiliki DDL, global privilege, atau schema aplikasi;
10. persetujuan menjalankan migration secara eksplisit.

Hasil kosong dari phpMyAdmin tidak boleh langsung dianggap sebagai bukti database/user tidak ada; operator harus membedakan objek tidak ada dari objek tidak terlihat karena privilege atau instance yang salah.

## 8. Urutan aman tahap berikutnya

1. Operator menjalankan query read-only Stage 6D.
2. Operator menyerahkan ringkasan non-rahasia hasil pemeriksaan.
3. Tim meninjau mismatch tipe, engine, FK, collation, atau existing table.
4. Jika seluruh blocker lulus, minta approval migration.
5. Jalankan migration hanya dengan akun migration pada database test.
6. Verifikasi metadata hasil migration.
7. Siapkan fixture terpisah dari Seed.
8. Baru kemudian jalankan kerangka integration test secara eksplisit.

## 9. Keputusan akhir

**BLOCKED** untuk eksekusi migration dan integration test karena database target, akun, privilege, tipe `users.id`, engine, FK, dan collation belum diverifikasi pada database nyata.

**READY FOR MANUAL VERIFICATION** karena source code dan migration sudah memiliki rencana pemeriksaan read-only yang dapat dijalankan operator sesuai Stage 6D.

## 10. Konfirmasi batasan

Hanya laporan ini yang dibuat. Tidak ada file source/migration lain yang diubah. Tidak ada koneksi database, query SQL, migration, server aplikasi, AutoMigrate, Seed, atau integration test yang dijalankan.
