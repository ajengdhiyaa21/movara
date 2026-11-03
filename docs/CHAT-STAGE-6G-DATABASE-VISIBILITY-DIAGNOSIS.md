# MOVARA Chat — Stage 6G: Database Target Visibility Diagnosis

Tanggal: 28 Oktober 2026  
Status: panduan operator; tidak ada koneksi atau query yang dijalankan.

## 1. Tujuan

Panduan ini membedakan empat kondisi yang dapat terlihat sama di phpMyAdmin:

1. database `movara_chat_test` memang belum dibuat;
2. phpMyAdmin terhubung ke instance MariaDB yang berbeda;
3. akun pemeriksa tidak memiliki hak melihat schema/metadata;
4. database target ada, tetapi tabel `users` belum tersedia.

Daftar database kosong atau database target tidak terlihat **tidak cukup** untuk menyimpulkan database pasti tidak ada.

## 2. Aturan keselamatan operator

- Semua query di dokumen ini bersifat read-only dan hanya instruksi manual.
- Jangan mencoba password default atau credential alternatif.
- Jangan memakai DSN aplikasi sebagai cara pemeriksaan.      
- Jangan menjalankan `CREATE DATABASE`, `CREATE USER`, `GRANT`, `REVOKE`, `DROP`, `TRUNCATE`, migration, AutoMigrate, atau Seed.
- Jangan menyalin password, DSN lengkap, kolom password, atau output credential ke laporan/chat.
- Jika hasil tidak jelas, berhenti dan minta administrator database memeriksa.

## 3. Langkah 1 — pastikan instance phpMyAdmin benar

### [READ-ONLY, hak biasa biasanya cukup]

Di phpMyAdmin, catat tanpa credential:

- host yang ditampilkan;
- metode koneksi TCP/IP atau socket;
- port yang ditampilkan;
- versi server;
- nama user aktif tanpa menyalin authentication data.

Kemudian, pada tab SQL, operator dapat menjalankan query berikut secara manual:

```sql
SELECT VERSION() AS server_version,
       @@version_comment AS version_comment,
       @@hostname AS server_host,
       @@port AS server_port,
       DATABASE() AS current_database;
```

Interpretasi:

| Hasil | Keputusan |
|---|---|
| Host/port/versi sesuai instance target | Lanjut ke pemeriksaan visibility |
| Host/port berbeda | Ini kemungkinan instance berbeda; hentikan diagnosis target dan perbaiki koneksi phpMyAdmin |
| Query ditolak | Identitas server belum terverifikasi; minta administrator |
| `DATABASE()` kosong | Sesi belum memilih database; ini bukan bukti database target tidak ada |

Versi binary lokal saja tidak membuktikan versi server yang sedang dipakai phpMyAdmin.

## 4. Langkah 2 — bedakan database belum dibuat dan tidak terlihat

### [READ-ONLY, hak metadata diperlukan]

```sql
SELECT SCHEMA_NAME,
       DEFAULT_CHARACTER_SET_NAME,
       DEFAULT_COLLATION_NAME
FROM INFORMATION_SCHEMA.SCHEMATA
WHERE SCHEMA_NAME = 'movara_chat_test';
```

### [READ-ONLY, hak metadata diperlukan]

```sql
SELECT SCHEMA_NAME
FROM INFORMATION_SCHEMA.SCHEMATA
ORDER BY SCHEMA_NAME;
```

Interpretasi:

### Kondisi A — database ditemukan

Jika query target mengembalikan satu baris, database ada dan terlihat. Catat charset/collation tanpa mengubahnya. Lanjutkan pemeriksaan tabel `users`, engine, tipe ID, dan grants.

### Kondisi B — database tidak terlihat, hak akses cukup

Database dapat disebut “belum ditemukan pada instance target” hanya jika:

- host/port/versi sudah sesuai;
- query metadata berhasil;
- operator memiliki hak melihat schema;
- administrator mengonfirmasi hasil visibility.

Tetap jangan langsung membuat database. Pembuatan memerlukan approval terpisah.

### Kondisi C — query kosong tetapi hak akses tidak diketahui

Statusnya **BELUM TERVERIFIKASI**. Hasil kosong bisa berarti database belum dibuat atau akun tidak memiliki visibility.

### Kondisi D — query ditolak

Statusnya **TIDAK DAPAT DIVERIFIKASI** karena privilege metadata tidak cukup. Minta administrator menjalankan pemeriksaan yang sama atau menyediakan ringkasan non-rahasia.

## 5. Langkah 3 — periksa user dan grants

### [READ-ONLY, hak administratif mungkin diperlukan]

```sql
SELECT User, Host, plugin, account_locked
FROM mysql.user
WHERE User = 'movara_test';
```

Kolom password tidak boleh dipilih atau disalin.

### [READ-ONLY, hak administratif]

```sql
SHOW GRANTS FOR 'movara_test'@'127.0.0.1';
```

Hasil yang perlu dipastikan oleh administrator:

- akun host sesuai policy;
- akses hanya pada `movara_chat_test`;
- tidak ada `ON *.*`;
- tidak ada `GRANT OPTION`;
- runtime test tidak memiliki DDL permanen;
- tidak ada akses ke database aplikasi.

Interpretasi:

| Hasil | Keputusan |
|---|---|
| User ditemukan dan grants sesuai | Kandidat runtime, tetapi belum otomatis READY |
| User tidak terlihat | User belum terverifikasi; jangan membuat ulang tanpa pemeriksaan admin |
| `SHOW GRANTS` ditolak | Privilege belum terverifikasi; jangan melanjutkan |
| Grants terlalu luas | BLOCKED; minta review admin, jangan revoke otomatis |

User `root@localhost` yang terlihat di phpMyAdmin tidak membuktikan `movara_test@127.0.0.1` tersedia atau aman.

## 6. Langkah 4 — periksa tabel `users`

### [READ-ONLY, hak metadata diperlukan]

Jika database target sudah dipastikan dan dipilih oleh operator:

```sql
SELECT TABLE_SCHEMA,
       TABLE_NAME,
       ENGINE,
       TABLE_COLLATION
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = 'movara_chat_test'
  AND TABLE_NAME = 'users';
```

### [READ-ONLY, hak metadata diperlukan]

```sql
SELECT TABLE_SCHEMA,
       TABLE_NAME,
       COLUMN_NAME,
       COLUMN_TYPE,
       IS_NULLABLE,
       COLUMN_KEY,
       EXTRA
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = 'movara_chat_test'
  AND TABLE_NAME = 'users'
  AND COLUMN_NAME = 'id';
```

Interpretasi:

### Kondisi E — database ada, `users` ditemukan

Bandingkan `COLUMN_TYPE` dengan `BIGINT UNSIGNED` dan pastikan engine mendukung foreign key. Jika tipe berbeda, migration chat tetap BLOCKED dan harus ditinjau ulang.

### Kondisi F — database ada, `users` tidak ditemukan

Ini berarti database target terlihat tetapi dependency parent belum tersedia, atau nama/schema berbeda. Migration chat tidak boleh dijalankan karena FK ke `users.id` akan gagal. Jangan menjalankan baseline aplikasi otomatis.

### Kondisi G — database atau tabel tidak terlihat

Jangan menyimpulkan tabel tidak ada sebelum hak metadata dan instance dipastikan. Minta administrator mengulang pemeriksaan dengan hak yang sesuai.

## 7. Alur diagnosis singkat

```text
Pastikan host/port/versi
        |
        +-- berbeda --> instance salah, hentikan
        |
        +-- sesuai --> query metadata schema
                         |
                         +-- ditolak --> privilege tidak cukup
                         |
                         +-- kosong --> belum dibedakan ada/tidak terlihat
                         |
                         +-- ditemukan --> periksa users
                                             |
                                             +-- tidak ada --> dependency belum siap
                                             |
                                             +-- ada --> periksa tipe ID, engine, grants
```

## 8. Keputusan berdasarkan kemungkinan hasil

| Kemungkinan | Status | Tindakan berikutnya |
|---|---|---|
| Instance berbeda | BLOCKED | Perbaiki sesi phpMyAdmin/client; jangan membuat objek |
| Schema belum ditemukan dengan visibility terbukti | NEEDS APPROVAL | Minta persetujuan provisioning database test |
| Schema tidak terlihat karena privilege | BLOCKED | Administrator melakukan verifikasi |
| Schema ada, user tidak terverifikasi | BLOCKED | Verifikasi akun/grants oleh administrator |
| Schema ada, `users` belum ada | BLOCKED | Tentukan dependency schema secara terpisah; jangan baseline otomatis |
| Schema dan users ada, tipe/engine tidak kompatibel | BLOCKED | Review migration/schema, jangan memaksa FK |
| Schema, users, tipe, engine, grants sesuai | READY FOR MIGRATION APPROVAL | Minta approval eksplisit sebelum migration |

Tidak ada kondisi di atas yang memberikan izin otomatis untuk membuat database, user, atau privilege.

## 9. Pemeriksaan minimum sebelum Stage 6H

Operator harus menyerahkan ringkasan non-rahasia mengenai:

1. instance/version/host/port;
2. status visibility `movara_chat_test`;
3. charset dan collation target;
4. status `users` dan tipe `users.id`;
5. engine `users`;
6. status `movara_test@127.0.0.1`;
7. ringkasan grants runtime dan migration;
8. keberadaan tabel chat existing.

Jika satu item penting tidak dapat diverifikasi, status tetap **BLOCKED**.

## 10. Pembuatan objek memerlukan persetujuan

Jika diagnosis menyimpulkan database atau user memang belum dibuat, operator tidak boleh langsung menjalankan provisioning. Pembuatan:

- `movara_chat_test`;
- akun runtime `movara_test`;
- akun migration;
- grants;
- schema atau privilege apa pun;

memerlukan konfirmasi operator dan persetujuan terpisah. Tidak ada tindakan tersebut dilakukan pada Stage 6G.

## 11. Konfirmasi batasan

Dokumen ini dibuat sebagai panduan diagnosis. Tidak ada koneksi database, autentikasi, query SQL, server, migration, AutoMigrate, Seed, integration test, pembuatan database/user, perubahan grants, source code, migration, konfigurasi, atau dependency yang dijalankan/diubah.
