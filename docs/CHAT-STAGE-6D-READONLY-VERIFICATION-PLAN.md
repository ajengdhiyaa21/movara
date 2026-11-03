# MOVARA Chat — Stage 6D: Read-only Database Verification Plan

Tanggal: 28 Oktober 2026  
Status: rencana pemeriksaan; tidak ada query atau koneksi yang dijalankan.

## 1. Temuan statis

### Mapping ID

`models.User.ID` bertipe Go `uint` dengan tag `gorm:"primaryKey"`. Kode tidak menuliskan tipe SQL secara eksplisit. Migration chat menggunakan `BIGINT UNSIGNED` untuk foreign key ke `users.id`, mengikuti konvensi pemetaan integer `uint` pada driver MySQL yang diharapkan.

Namun, tipe aktual `users.id` belum dapat dibuktikan hanya dari source code. Operator wajib memeriksa `INFORMATION_SCHEMA.COLUMNS` sebelum migration. Jika hasil bukan `BIGINT UNSIGNED` yang kompatibel, migration harus dihentikan dan schema migration ditinjau ulang.

### Konsistensi migration

`migrations/000001_create_chat_tables.up.sql` konsisten dengan model/repository Stage 6A–6B:

- `chat_conversations.pm_user_id` unik;
- `chat_messages` memakai `conversation_id`, `sender_user_id`, `client_message_id`, `body`, `created_at`;
- unique `(conversation_id, client_message_id)`;
- index `(conversation_id, id)` untuk cursor;
- `chat_read_states` memakai unique `(conversation_id, user_id)`;
- foreign key memakai `ON DELETE RESTRICT` dan `ON UPDATE RESTRICT`;
- semua tabel memakai `ENGINE=InnoDB`;
- timestamp memakai `DATETIME(6)`.

Sintaks hanya diperiksa secara tekstual pada file. Tidak ada SQL parser/server MariaDB yang dijalankan pada tahap ini.

## 2. Cara penggunaan

Query di bawah adalah **template untuk operator** melalui phpMyAdmin setelah sesi target yang sah dibuka. Query tersebut tidak boleh dijalankan oleh agent pada Stage 6D.

Sebelum memulai, operator harus memastikan phpMyAdmin menunjuk ke MariaDB target `127.0.0.1:3306`. Jangan menaruh password atau DSN pada laporan.

## 3. Identitas dan versi server

### [READ-ONLY] Versi dan identitas server

```sql
SELECT VERSION() AS server_version,
       @@version_comment AS version_comment,
       @@hostname AS server_host,
       @@port AS server_port;
```

Jika query gagal karena hak akses atau sesi tidak tersedia, status versi server SQL adalah **BELUM TERVERIFIKASI**. Versi executable lokal tidak menggantikan versi server aktif.

## 4. Database aktif dan target

### [READ-ONLY] Sesi aktif

```sql
SELECT DATABASE() AS current_database;
```

### [READ-ONLY] Daftar schema yang terlihat

```sql
SELECT SCHEMA_NAME,
       DEFAULT_CHARACTER_SET_NAME,
       DEFAULT_COLLATION_NAME
FROM INFORMATION_SCHEMA.SCHEMATA
ORDER BY SCHEMA_NAME;
```

### [READ-ONLY] Target test secara spesifik

```sql
SELECT SCHEMA_NAME,
       DEFAULT_CHARACTER_SET_NAME,
       DEFAULT_COLLATION_NAME
FROM INFORMATION_SCHEMA.SCHEMATA
WHERE SCHEMA_NAME = 'movara_chat_test';
```

Baris kosong berarti schema tidak terlihat oleh akun pada sesi tersebut. Itu belum membedakan schema tidak ada dari schema yang tidak terlihat karena privilege.

## 5. Tipe `users.id` dan engine users

### [READ-ONLY] Tipe kolom parent

```sql
SELECT TABLE_SCHEMA,
       TABLE_NAME,
       COLUMN_NAME,
       COLUMN_TYPE,
       IS_NULLABLE,
       COLUMN_KEY,
       EXTRA
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'users'
  AND COLUMN_NAME = 'id';
```

Untuk migration ini, hasil `COLUMN_TYPE` harus dibandingkan dengan `BIGINT UNSIGNED`. Perbedaan tipe, signedness, atau ukuran harus menjadi blocker.

### [READ-ONLY] Engine dan collation tabel users

```sql
SELECT TABLE_SCHEMA,
       TABLE_NAME,
       ENGINE,
       TABLE_COLLATION,
       TABLE_ROWS
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'users';
```

Foreign key InnoDB membutuhkan parent dan child table menggunakan engine yang kompatibel. `TABLE_ROWS` bersifat estimasi dan bukan validasi data.

## 6. Pemeriksaan tabel chat jika sudah ada

### [READ-ONLY] Keberadaan tabel

```sql
SELECT TABLE_NAME, ENGINE, TABLE_COLLATION
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = 'movara_chat_test'
  AND TABLE_NAME IN (
    'chat_conversations',
    'chat_messages',
    'chat_read_states'
  )
ORDER BY TABLE_NAME;
```

### [READ-ONLY] Kolom dan tipe

```sql
SELECT TABLE_NAME,
       ORDINAL_POSITION,
       COLUMN_NAME,
       COLUMN_TYPE,
       IS_NULLABLE,
       COLUMN_DEFAULT,
       EXTRA
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = 'movara_chat_test'
  AND TABLE_NAME IN (
    'chat_conversations',
    'chat_messages',
    'chat_read_states'
  )
ORDER BY TABLE_NAME, ORDINAL_POSITION;
```

### [READ-ONLY] Index dan unique constraint

```sql
SELECT TABLE_NAME,
       INDEX_NAME,
       NON_UNIQUE,
       SEQ_IN_INDEX,
       COLUMN_NAME,
       INDEX_TYPE
FROM INFORMATION_SCHEMA.STATISTICS
WHERE TABLE_SCHEMA = 'movara_chat_test'
  AND TABLE_NAME IN (
    'chat_conversations',
    'chat_messages',
    'chat_read_states'
  )
ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX;
```

### [READ-ONLY] Foreign key

```sql
SELECT TABLE_NAME,
       CONSTRAINT_NAME,
       COLUMN_NAME,
       REFERENCED_TABLE_NAME,
       REFERENCED_COLUMN_NAME,
       UPDATE_RULE,
       DELETE_RULE
FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = 'movara_chat_test'
  AND TABLE_NAME IN (
    'chat_conversations',
    'chat_messages',
    'chat_read_states'
  )
  AND REFERENCED_TABLE_NAME IS NOT NULL
ORDER BY TABLE_NAME, CONSTRAINT_NAME, ORDINAL_POSITION;
```

Jika tabel chat belum ada, hasil kosong adalah expected dan tidak menunjukkan migration sudah dijalankan. Jika tabel ada tetapi sebagian metadata tidak terlihat, pemeriksaan harus dihentikan sampai hak akses operator dikonfirmasi.

## 7. Akun database dan grants

### [READ-ONLY, HAK ADMINISTRATIF MUNGKIN DIPERLUKAN] Daftar akun tanpa credential

```sql
SELECT User, Host, plugin, account_locked
FROM mysql.user
ORDER BY User, Host;
```

Query ini membaca metadata sistem dan dapat ditolak untuk akun biasa. Jangan memilih, menyalin, atau menampilkan kolom password.

### [READ-ONLY, HAK ADMINISTRATIF MUNGKIN DIPERLUKAN] Akun target

```sql
SELECT User, Host, plugin, account_locked
FROM mysql.user
WHERE User = 'movara_test';
```

### [READ-ONLY, HAK ADMINISTRATIF] Grants akun runtime

```sql
SHOW GRANTS FOR 'movara_test'@'127.0.0.1';
```

Hasil yang aman harus dibaca oleh operator tanpa memasukkan output credential ke laporan. Runtime test tidak boleh memiliki `ON *.*`, `GRANT OPTION`, atau privilege pada database aplikasi.

### [READ-ONLY, HAK ADMINISTRATIF MUNGKIN DIPERLUKAN] Ringkasan privilege yang terlihat

```sql
SELECT GRANTEE, TABLE_SCHEMA, PRIVILEGE_TYPE
FROM INFORMATION_SCHEMA.SCHEMA_PRIVILEGES
WHERE GRANTEE LIKE '''movara_test''@''%'''
ORDER BY GRANTEE, TABLE_SCHEMA, PRIVILEGE_TYPE;
```

Jika `mysql.user`, `SHOW GRANTS`, atau `SCHEMA_PRIVILEGES` tidak dapat dibaca, status grants adalah **BELUM TERVERIFIKASI**, bukan “aman”.

## 8. Membedakan objek tidak ada dan objek tidak terlihat

Operator harus membedakan tiga kondisi:

1. **Objek ditemukan**: metadata mengembalikan baris dan detail dapat dibaca.
2. **Objek tidak terlihat**: query menghasilkan kosong atau phpMyAdmin tidak menampilkan objek, tetapi akun mungkin tidak memiliki visibility.
3. **Pemeriksaan ditolak**: server mengembalikan error privilege/akses.

Hasil kosong hanya dapat disebut “tidak ditemukan” jika operator memiliki hak metadata yang cukup dan sesi target telah dipastikan benar. Error permission harus dicatat sebagai “tidak dapat diverifikasi”. Jangan membuat objek berdasarkan hasil kosong yang belum dibedakan.

## 9. Pemeriksaan minimum sebelum migration

Migration tetap **BLOCKED** sampai semua poin berikut selesai:

1. versi server aktif dan host/port terkonfirmasi;
2. database target terkonfirmasi sebagai `movara_chat_test` pada instance yang benar;
3. `users.id` kompatibel dengan `BIGINT UNSIGNED` atau migration disesuaikan melalui review;
4. tabel `users` ada dan engine-nya mendukung foreign key;
5. charset dan collation database target diketahui dan disetujui;
6. akun migration terpisah dari runtime test;
7. grants runtime hanya pada schema test dan tidak memiliki DDL/global privilege;
8. tabel chat belum ada, atau jika sudah ada maka migration tidak dijalankan ulang tanpa review;
9. prosedur migration eksplisit disetujui;
10. hasil pemeriksaan disimpan hanya sebagai ringkasan non-rahasia.

## 10. Keputusan yang masih membutuhkan operator

- memberikan akses phpMyAdmin/client kepada operator yang berwenang;
- memverifikasi `users.id`, engine, FK, charset, dan collation;
- memverifikasi keberadaan `movara_chat_test` dan `movara_test` tanpa menganggap hasil kosong sebagai bukti mutlak;
- memverifikasi grants runtime dan migration;
- menyetujui migration pertama setelah seluruh blocker selesai.

## 11. Konfirmasi batasan

Dokumen ini adalah rencana query read-only. Tidak ada koneksi database, autentikasi, query SQL, migration, server, AutoMigrate, Seed, integration test, pembuatan database/user, perubahan privilege, schema, data, konfigurasi, atau source code yang dilakukan.
