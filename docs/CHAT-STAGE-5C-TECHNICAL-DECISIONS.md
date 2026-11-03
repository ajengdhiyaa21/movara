# MOVARA Internal Chat — Stage 5C: Finalisasi Keputusan Teknis

Tanggal: 28 Oktober 2026  
Status: keputusan teknis MVP; implementasi belum dimulai.

## 1. Dasar keputusan

Kode aktual yang ditinjau:

- `internal/models/models.go`
- `internal/middleware/auth.go`
- `internal/router/router.go`
- `internal/config/testdb.go`
- `internal/config/testdb_policy.go`
- `go.mod`

Dokumen yang ditinjau:

- `docs/CHAT-REQUIREMENTS.md`
- `docs/CHAT-TECHNICAL-PLAN.md`
- keputusan Stage 5A dan 5B yang diberikan sebagai konteks percakapan.

Laporan Stage 5A dan 5B tidak ditemukan sebagai file terpisah di repository; keputusan final yang ditulis di bawah mengikuti keputusan Stage 5A/5B pada instruksi proyek.

Belum ada model, repository, service, handler, route, migration, atau tabel chat di kode aktual.

## 2. Keputusan final MVP

- Tiga tabel: `chat_conversations`, `chat_messages`, `chat_read_states`.
- Satu conversation untuk setiap PM melalui unique `pm_user_id`.
- Semua Admin aktif berbagi conversation PM.
- Manager tidak memiliki akses ke chat.
- PM hanya boleh mengakses conversation miliknya.
- PM nonaktif tidak dapat login, membaca, mengirim pesan, atau membuat read state baru.
- Admin aktif dapat membaca histori PM nonaktif, tetapi tidak dapat mengirim pesan baru kepadanya.
- Admin nonaktif kehilangan akses setelah status nonaktif dimuat ulang oleh middleware.
- Pesan maksimal 3.000 karakter dan tidak dihapus otomatis.
- Tidak ada attachment, broadcast, typing indicator, edit pesan, atau hapus pesan.
- HTTP adalah sumber kebenaran untuk histori, pengiriman, dan read state.
- WebSocket hanya mengirim event pembaruan.
- Migration berupa SQL berversi dan dijalankan terpisah dari startup.
- CSRF tetap pekerjaan terpisah dan tidak diubah pada tahap ini.

## 3. `client_message_id`

Client membuat UUID sebelum request pertama, lalu menyimpan UUID tersebut selama seluruh siklus retry. Server memvalidasi:

- format UUID yang disetujui;
- panjang maksimum input sebelum parsing;
- tidak kosong;
- hanya digunakan pada conversation yang sedang dituju.

Database mempertahankan unique `(conversation_id, client_message_id)`.

Perilaku duplicate-key:

1. Jika request pertama berhasil, kembalikan pesan yang dibuat.
2. Jika retry memakai UUID yang sama, ambil pesan existing dan kembalikan hasil idempotent.
3. Jika UUID sama tetapi payload isi berbeda, tolak sebagai konflik idempotency.
4. Jangan membuat pesan kedua.

`sender_user_id` selalu berasal dari session server, bukan payload client.

## 4. Kontrak HTTP yang diusulkan

### Membuka atau memperoleh conversation

`POST /api/chat/conversations`

Request Admin menyebutkan `pm_user_id`. Request PM tidak boleh menentukan PM lain; server menggunakan user dari session.

Response mengembalikan conversation yang sudah ada atau baru dibuat. Pembuatan harus aman terhadap request paralel menggunakan unique constraint dan retry lookup setelah duplicate-key.

### Daftar conversation

`GET /api/chat/conversations`

- Admin aktif: PM aktif ditampilkan sebagai daftar utama; histori PM nonaktif dapat dicari sesuai policy.
- PM aktif: hanya conversation miliknya.
- Manager/nonaktif/tanpa session: ditolak.

### Histori pesan

`GET /api/chat/conversations/:id/messages?before_id=<id>&limit=<n>`

Request berikutnya untuk pesan yang lebih baru dapat memakai:

`GET /api/chat/conversations/:id/messages?after_id=<id>&limit=<n>`

Server selalu memeriksa hak akses conversation sebelum mengambil pesan.

### Kirim pesan

`POST /api/chat/conversations/:id/messages`

Payload konseptual:

```json
{
  "client_message_id": "<uuid>",
  "body": "Pesan teks"
}
```

Server memvalidasi session, role, status aktif, conversation ownership, content type, body kosong, panjang maksimal 3.000 karakter, UUID, dan idempotency.

### Tandai terbaca

`POST /api/chat/conversations/:id/read`

Payload membawa `message_id`. Server memvalidasi bahwa pesan berada pada conversation yang sama dan hanya memajukan read state, tidak memundurkannya.

## 5. Kontrak WebSocket minimal

Endpoint konseptual: `GET /api/chat/ws`.

Handshake wajib memvalidasi session, user aktif, role, Origin, dan conversation access. Manager dan user nonaktif ditolak.

Event server minimal:

```json
{
  "type": "message.created",
  "conversation_id": 12,
  "message_id": 45,
  "created_at": "server-time"
}
```

Event hanya menjadi sinyal bahwa client perlu mengambil data melalui HTTP. Event tidak menjadi sumber kebenaran dan tidak perlu membawa seluruh isi pesan.

Reconnect:

1. Client menyimpan `after_id` terakhir yang berhasil diproses.
2. Setelah reconnect, client mengambil pesan melalui HTTP dengan `after_id`.
3. Server memeriksa authorization kembali.
4. Client melakukan deduplikasi berdasarkan `message_id`.
5. WebSocket baru dipakai untuk event setelah histori tersinkron.

Jika WebSocket gagal, alur HTTP tetap harus berfungsi.

## 6. Pagination dan validasi

Gunakan cursor berbasis ID, bukan offset sebagai mekanisme utama:

- `before_id`: histori lebih lama;
- `after_id`: pesan setelah pesan terakhir yang diterima;
- `limit`: default kecil, misalnya 30;
- maksimum server, misalnya 100.

Nilai maksimum limit adalah keputusan implementasi yang dapat dimulai dengan 100 dan diuji berdasarkan beban. Client tidak boleh menentukan limit tanpa batas.

Query wajib memakai conversation scope dan index `(conversation_id, id)`. `message_id`, `conversation_id`, dan `pm_user_id` dari URL/payload bukan bukti akses; semuanya harus diverifikasi server-side.

## 7. Schema dan constraint yang diterapkan

### `chat_conversations`

```text
id          BIGINT UNSIGNED PRIMARY KEY
pm_user_id  BIGINT UNSIGNED NOT NULL UNIQUE
created_at  DATETIME(6) NOT NULL
updated_at  DATETIME(6) NOT NULL
```

FK `pm_user_id → users.id` dengan `ON DELETE RESTRICT` dan `ON UPDATE RESTRICT`.

### `chat_messages`

```text
id                  BIGINT UNSIGNED PRIMARY KEY
conversation_id     BIGINT UNSIGNED NOT NULL
sender_user_id      BIGINT UNSIGNED NOT NULL
client_message_id   VARCHAR(100) NOT NULL
body                VARCHAR(3000) NOT NULL
created_at          DATETIME(6) NOT NULL
```

Unique `(conversation_id, client_message_id)`, index `(conversation_id, id)`, dan index `(sender_user_id, created_at)`. FK conversation dan sender memakai `RESTRICT`.

### `chat_read_states`

```text
id                    BIGINT UNSIGNED PRIMARY KEY
conversation_id       BIGINT UNSIGNED NOT NULL
user_id               BIGINT UNSIGNED NOT NULL
last_read_message_id  BIGINT UNSIGNED NULL
last_read_at          DATETIME(6) NULL
created_at            DATETIME(6) NOT NULL
updated_at            DATETIME(6) NOT NULL
```

Unique `(conversation_id, user_id)`, index `(user_id, conversation_id)`, dan FK conversation/user dengan `RESTRICT`.

`last_read_message_id` harus diverifikasi berada pada conversation yang sama. Update read state hanya boleh maju.

## 8. Rekomendasi migration

Rekomendasi awal: SQL migration berversi dengan tool `golang-migrate/migrate` sebagai CLI atau mekanisme terpisah. Repository saat ini belum memiliki dependency migration dan `go.mod` belum mencantumkan library migration.

Keputusan ini berarti:

- dependency tidak ditambahkan pada Stage 5C;
- tool/CLI dan format version table harus disetujui sebelum implementasi;
- migration dijalankan manual/terkontrol terhadap database test yang sudah divalidasi;
- migration tidak dipanggil dari `cmd/server/main.go`;
- startup tetap tidak menjalankan migration chat;
- akun migration/DDL harus berbeda dari akun runtime test;
- migration tidak boleh menyentuh database aplikasi.

SQL migration harus dimulai dari schema chat yang telah disetujui, bukan baseline seluruh aplikasi. Kebutuhan tabel `users` hanya berupa dependency existing yang harus diverifikasi pada database test.

## 9. Strategi pengujian

### Unit test tanpa database

Uji:

- role Admin/Manager/PM;
- PM hanya dapat mengakses conversation sendiri;
- Admin aktif dapat membaca semua PM;
- Admin tidak dapat mengirim ke PM nonaktif;
- PM nonaktif ditolak untuk read state dan pengiriman;
- validasi 3.000 karakter;
- UUID invalid/kosong/terlalu panjang;
- duplicate-key/idempotency dengan payload sama dan berbeda;
- read state monotonic;
- pagination cursor;
- validasi `after_id` dan deduplikasi event.

### Integration test database

Hanya memakai database test terisolasi melalui `APP_ENV=test`, `TEST_DATABASE_URL`, dan `ApprovedTestDatabasePolicy()`.

Skenario minimum:

- concurrent conversation creation untuk PM yang sama;
- FK dan `RESTRICT`;
- unique client message ID;
- pagination berdasarkan ID;
- read state per Admin dan PM;
- data PM A tidak terlihat PM B;
- schema/privilege tidak mengarah ke database aplikasi.

Integration test tidak boleh memanggil `cmd/server/main.go`, `AutoMigrate`, atau `database.Seed`.

### WebSocket test

Setelah persistence dan HTTP stabil:

- handshake Admin/PM valid;
- Manager ditolak;
- session expired menutup koneksi;
- Origin invalid ditolak;
- reconnect dengan `after_id` tidak menggandakan pesan;
- event yang terlambat tidak merusak urutan histori.

## 10. Verifikasi database yang masih tertunda

Belum boleh dianggap terbukti:

- database test tersedia dan benar-benar terpisah;
- user runtime tersedia;
- grants runtime hanya DML pada schema test;
- akun migration berbeda dan memiliki DDL minimum;
- engine tabel mendukung foreign key;
- foreign key benar-benar aktif pada MariaDB target;
- collation database test;
- perilaku case sensitivity `client_message_id`;
- tipe aktual hasil migration untuk `uint` existing;
- konfigurasi backup/retensi pesan.

Tidak ada pemeriksaan database yang dilakukan pada Stage 5C.

## 11. Urutan implementasi paling pendek dan aman

1. Persetujuan kontrak API, event, limit, dan migration tool.
2. Verifikasi manual database test, akun, grants, engine, dan collation.
3. Tambahkan SQL migration chat berversi.
4. Tambahkan model dan repository chat.
5. Tambahkan unit test authorization, idempotency, pagination, dan read state.
6. Jalankan migration hanya terhadap database test yang telah disetujui.
7. Tambahkan fixture sintetis terpisah dari Seed.
8. Tambahkan integration test repository/service.
9. Tambahkan service dan HTTP API.
10. Tambahkan CSRF pada pekerjaan terpisah setelah approval eksplisit.
11. Tambahkan WebSocket event adapter dan test reconnect.
12. Tambahkan UI chat terakhir.

Jika validasi database atau authorization gagal, hentikan tahap berikutnya.

## 12. Persetujuan yang masih diperlukan

1. Tool migration SQL berversi dan cara eksekusinya.
2. Nilai limit pagination awal/maksimum.
3. Format UUID yang diterima dan batas panjang `client_message_id`.
4. Engine tabel dan collation final setelah verifikasi database test.
5. Kebijakan error duplicate idempotency dengan payload berbeda.
6. Format autentikasi dan Origin validation WebSocket.
7. Prosedur fixture/cleanup database test.
8. Approval terpisah untuk perubahan CSRF.

## 13. Konfirmasi batasan

Laporan ini dibuat sebagai keputusan teknis. Tidak ada kode, dependency, konfigurasi, migration, model, tabel, server, AutoMigrate, Seed, SQL, koneksi database, atau integration test yang diubah/dijalankan.
