# MOVARA Chat — Stage 6J: Safe MariaDB Privilege Diagnosis Plan

Tanggal: 28 Oktober 2026  
Status: **READ-ONLY PLAN — PROVISIONING BLOCKED**

## 1. Konteks dan sumber

Konteks yang diberikan operator:

- MariaDB XAMPP 10.4.32 pada port 3306;
- database `movara_chat_test` sudah dibuat dan masih kosong;
- akun `movara_test` dan `movara_migrator` pernah dibuat;
- pemberian privilege pernah gagal dengan `#1034 - Index for table 'db' is corrupt`;
- terdapat log tablespace pada database lain `memo_supplier_mk`;
- database aplikasi `movara` tidak boleh diubah.

Rujukan:

- `docs/CHAT-STAGE-6I-BASELINE-DESIGN.md`
- `docs/CHAT-STAGE-6G-DATABASE-VISIBILITY-DIAGNOSIS.md`
- `docs/CHAT-STAGE-6D-READONLY-VERIFICATION-PLAN.md`

Tidak ada koneksi MariaDB, query, server, migration, AutoMigrate, Seed, atau integration test yang dijalankan.

## 2. Arti teknis error `#1034`

Pesan `Index for table 'db' is corrupt` menunjuk pada kegagalan MariaDB membaca atau memperbarui index tabel sistem bernama `db`, yang biasanya berkaitan dengan metadata privilege. Dalam konteks pemberian `GRANT`, error tersebut dapat berarti operasi privilege tidak dapat menyelesaikan pembaruan metadata sistem.

Error itu dapat mendukung kesimpulan berikut:

- pemberian privilege belum berhasil dibuktikan;
- metadata privilege server perlu diperiksa oleh administrator;
- akun dan grants tidak boleh dianggap aman hanya karena akun pernah dibuat;
- provisioning harus dihentikan sampai kondisi metadata sistem dipahami.

Error itu **tidak** membuktikan:

- database `movara` rusak;
- database `movara_chat_test` rusak;
- seluruh MariaDB rusak;
- tabel aplikasi tertentu mengalami kerusakan;
- log tablespace `memo_supplier_mk` memiliki hubungan sebab-akibat dengan tabel sistem `mysql.db`.

Korelasi antara error privilege dan log tablespace belum dapat ditentukan tanpa pemeriksaan administrator. Jangan melakukan perbaikan spekulatif.

## 3. Prinsip diagnosis

Diagnosis harus memisahkan empat hal:

1. **Identitas instance** — server/port/versi yang sedang dipakai phpMyAdmin.
2. **Keberadaan database** — apakah `movara_chat_test` terlihat pada instance tersebut.
3. **Keberadaan akun** — apakah `movara_test@127.0.0.1` dan `movara_migrator@127.0.0.1` tersedia.
4. **Privilege dan kondisi metadata sistem** — apakah grants dapat dibaca dan tabel privilege dapat diakses/dioperasikan.

Database yang terlihat dan akun yang ada belum berarti grants benar. Sebaliknya, hasil kosong dari daftar phpMyAdmin belum membuktikan objek tidak ada jika visibility belum terverifikasi.

## 4. Langkah read-only untuk operator

Semua perintah di bagian ini hanya template untuk operator melalui phpMyAdmin. Jangan menjalankannya sebagai bagian dari tahap ini.

### 4.1 Identitas instance

**READ-ONLY, biasanya dapat dilakukan akun biasa:**

```sql
SELECT VERSION() AS server_version,
       @@version_comment AS version_comment,
       @@hostname AS server_host,
       @@port AS server_port,
       DATABASE() AS current_database;
```

Catat hanya host, port, versi, dan database aktif. Jangan menyalin credential atau DSN.

Jika host/port/versi berbeda dari target, hentikan diagnosis karena phpMyAdmin kemungkinan terhubung ke instance lain.

### 4.2 Visibility database target

**READ-ONLY, memerlukan visibility metadata:**

```sql
SELECT SCHEMA_NAME,
       DEFAULT_CHARACTER_SET_NAME,
       DEFAULT_COLLATION_NAME
FROM INFORMATION_SCHEMA.SCHEMATA
WHERE SCHEMA_NAME = 'movara_chat_test';
```

Interpretasi:

- satu baris: database terlihat pada instance itu;
- kosong: belum dapat dibedakan antara belum dibuat dan tidak terlihat;
- error privilege: visibility tidak dapat diverifikasi.

Jangan menjalankan `CREATE DATABASE` berdasarkan hasil kosong.

### 4.3 Keberadaan akun tanpa credential

**READ-ONLY, umumnya memerlukan hak administratif:**

```sql
SELECT User, Host, plugin, account_locked
FROM mysql.user
WHERE User IN ('movara_test', 'movara_migrator');
```

Jangan memilih atau menampilkan kolom password/authentication string.

Jika query ditolak, akun tidak dapat dianggap tidak ada; statusnya **BELUM TERVERIFIKASI**.

### 4.4 Pemeriksaan grants

**READ-ONLY, membutuhkan hak untuk melihat grants akun:**

```sql
SHOW GRANTS FOR 'movara_test'@'127.0.0.1';
SHOW GRANTS FOR 'movara_migrator'@'127.0.0.1';
```

Administrator harus memeriksa ringkasan berikut tanpa memasukkan output sensitif ke laporan:

- akses hanya pada `movara_chat_test`;
- tidak ada `ON *.*`;
- tidak ada `GRANT OPTION`;
- runtime tidak memiliki privilege DDL permanen;
- migration account dipisahkan dari runtime;
- tidak ada akses ke database `movara`.

Kegagalan `SHOW GRANTS` berarti privilege belum terverifikasi, bukan berarti grants aman atau akun tidak ada.

## 5. Pemeriksaan kondisi tabel sistem MariaDB

Pemeriksaan ini memerlukan administrator database. Tujuannya hanya membaca status metadata; bukan memperbaiki atau mengoptimalkan tabel sistem.

### 5.1 Baca metadata objek privilege

Administrator dapat memeriksa secara read-only apakah objek metadata privilege dapat dibaca, termasuk tabel sistem yang relevan dengan database-level grants. Perintah dan detail output harus dipilih oleh administrator sesuai versi MariaDB 10.4.32 dan hak akses yang tersedia.

Jika pemeriksaan metadata sistem menghasilkan error korupsi, error tablespace, atau error I/O, hentikan provisioning. Jangan mencoba `REPAIR TABLE`, `OPTIMIZE TABLE`, `ALTER TABLE`, `mysql_upgrade`, atau tindakan lain dari rencana ini.

### 5.2 Bedakan kegagalan privilege dari kerusakan objek aplikasi

Administrator harus mencatat secara non-rahasia:

- apakah error terjadi saat membaca atau saat mengubah privilege;
- nama objek sistem yang disebut error;
- apakah query read-only terhadap metadata gagal;
- apakah query metadata database target berhasil;
- apakah error yang sama muncul untuk database aplikasi atau hanya saat grant.

Keterangan log `memo_supplier_mk` harus diperlakukan sebagai temuan terpisah sampai administrator membuktikan hubungan teknisnya.

## 6. Klasifikasi hak dan tindakan

| Pemeriksaan/tindakan | Klasifikasi | Keterangan |
|---|---|---|
| Identitas versi/host/port | Read-only, hak biasa biasanya cukup | Tidak mengubah state |
| Visibility `INFORMATION_SCHEMA` | Read-only | Hasil kosong perlu dibedakan dari kurang privilege |
| Baca daftar akun sistem | Read-only administratif | Jangan membaca credential |
| `SHOW GRANTS` | Read-only administratif | Ringkas hasil tanpa secret |
| Membuat database/user | Mengubah state, approval wajib | Tidak dilakukan pada Stage 6J |
| `GRANT`/`REVOKE` | Mengubah privilege, approval wajib | Terblokir oleh error `#1034` |
| Perbaikan tabel sistem | Operasi administratif berisiko tinggi | Di luar scope; harus ditangani prosedur resmi administrator |
| Migration chat | Mengubah schema, approval wajib | Tidak boleh sebelum blocker selesai |

## 7. Kondisi berhenti

Operator harus berhenti dan tidak melanjutkan provisioning atau migration jika salah satu kondisi berikut terjadi:

- host, port, atau versi instance tidak sesuai;
- target database tidak terlihat dan visibility belum terbukti;
- akun runtime/migration tidak dapat diverifikasi;
- `SHOW GRANTS` ditolak atau hasilnya terlalu luas;
- pemberian privilege masih menghasilkan `#1034`;
- pemeriksaan metadata privilege menghasilkan error korupsi/tablespace/I/O;
- tipe `users.id`, engine, charset, atau collation belum diverifikasi;
- akun runtime masih memiliki privilege DDL atau akses ke `movara`;
- administrator tidak dapat memastikan target bukan database aplikasi;
- terdapat indikasi kerusakan sistem MariaDB tetapi belum ada diagnosis resmi.

Dalam kondisi berhenti, jangan melakukan retry `GRANT` berulang-ulang dan jangan menjalankan migration untuk menguji keadaan.

## 8. Keputusan berdasarkan hasil diagnosis

| Hasil | Status | Langkah berikutnya |
|---|---|---|
| Instance salah | BLOCKED | Perbaiki sesi phpMyAdmin/client setelah konfirmasi operator |
| Database terlihat, grants tidak terbaca | BLOCKED | Administrator memeriksa metadata dan grants |
| Akun tidak terlihat, hak pemeriksaan tidak cukup | BLOCKED | Administrator melakukan verifikasi |
| Grants ada tetapi terlalu luas | BLOCKED | Review privilege resmi; jangan revoke otomatis |
| `#1034` hanya pada operasi grant | BLOCKED | Diagnosis metadata privilege oleh administrator |
| Error read-only pada tabel sistem | BLOCKED | Ikuti prosedur pemulihan resmi; jangan provisioning |
| Semua metadata/grants valid dan database terisolasi | READY FOR SEPARATE APPROVAL | Ajukan persetujuan migration secara terpisah |

Tidak ada hasil yang memberi izin otomatis untuk membuat akun, mengubah grants, menjalankan migration, atau menyentuh database `movara`.

## 9. Tindakan yang tidak boleh dilakukan

Jangan:

- menghapus file `.ibd`;
- menghapus isi direktori data MariaDB;
- menghapus database aplikasi;
- menjalankan `DROP DATABASE`, `DROP USER`, `TRUNCATE`, atau cleanup destruktif;
- menjalankan `REPAIR TABLE` atau memperbaiki tabel sistem berdasarkan dugaan;
- reinstall XAMPP sebagai diagnosis pertama;
- menyalin database `movara` ke test;
- menjalankan AutoMigrate, Seed, atau startup aplikasi.

## 10. Kesimpulan

Error `#1034` cukup untuk menahan provisioning karena keberhasilan pemberian privilege belum dapat dibuktikan. Error tersebut bukan bukti bahwa database `movara` rusak. Langkah aman berikutnya adalah pemeriksaan read-only oleh operator/administrator untuk memastikan instance, visibility, akun, grants, dan kondisi metadata sistem. Sampai pemeriksaan itu selesai dan hasilnya disetujui, database test dan migration chat tetap **BLOCKED**.

File yang dibuat pada tahap ini hanya laporan ini. Tidak ada koneksi database, autentikasi, SQL, perubahan privilege, perubahan schema/data, server, migration, AutoMigrate, Seed, atau integration test yang dilakukan.
