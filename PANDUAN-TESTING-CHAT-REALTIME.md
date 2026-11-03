# Panduan Testing Chat Realtime MOVARA

Dokumen ini adalah panduan pengujian manual untuk integrasi Chat HTTP, ChatHub, WebSocket, notifier, dan frontend. Pengujian harus dilakukan hanya pada lingkungan test yang sudah disetujui.

## Prasyarat keselamatan

- Jangan menggunakan database `movara`.
- Pastikan operator memiliki target test terpisah `movara_chat_test` dan akun runtime yang hanya memiliki hak pada database tersebut.
- Jangan menjalankan startup normal sebelum target database diverifikasi. `cmd/server/main.go` membaca `DATABASE_URL`, lalu memanggil `database.Migrate()` dan `database.Seed()`.
- Siapkan dua akun aktif: satu Admin dan satu PM. Siapkan Manager untuk pengujian penolakan akses.
- Catat URL dan port aplikasi dari konfigurasi `PORT`; jangan mengasumsikan port selain nilai konfigurasi.
- Buka DevTools browser pada tab Network dan Console.

## Cara memastikan target database aman

Pengujian manual aplikasi tidak membuktikan database target secara otomatis. Sebelum menjalankan aplikasi:

1. Pastikan konfigurasi runtime yang disetujui menunjuk ke database test, bukan database utama.
2. Jangan menyalin password atau DSN lengkap ke laporan.
3. Pastikan database target adalah `movara_chat_test` dan akun runtime bukan akun production.
4. Pastikan privilege akun runtime hanya mencakup kebutuhan runtime pada `movara_chat_test`.
5. Pastikan `DATABASE_URL` tidak diarahkan ke `movara` pada sesi test.
6. Catat hanya nama database, host, port, dan akun non-rahasia untuk bukti test.

`TEST_DATABASE_URL` digunakan oleh runner/test integration secara eksplisit. `cmd/server/main.go` tetap menggunakan `DATABASE_URL`, sehingga menjalankan server normal bukan cara untuk menjalankan integration test database.

## Skenario 1 — Login Admin dan PM

### Persiapan

- Siapkan akun Admin aktif dan PM aktif.
- Buka aplikasi pada dua browser profile atau satu browser dan satu incognito window.

### Langkah

1. Login sebagai Admin.
2. Buka halaman Chat.
3. Login sebagai PM pada profile kedua.
4. Buka halaman Chat.

### Hasil yang diharapkan

- Admin melihat daftar PM.
- PM langsung melihat percakapannya sendiri.
- DevTools menunjukkan koneksi WebSocket ke `/chat/ws`.
- Tidak ada akses database utama yang digunakan.

### Lulus/gagal

- Lulus jika kedua akun membuka Chat tanpa error dan koneksi WebSocket aktif.
- Gagal jika Manager/PM lain muncul sebagai percakapan PM, atau handshake gagal untuk akun valid.

## Skenario 2 — Pengiriman dan penerimaan realtime

### Persiapan

- Admin dan PM sudah login.
- Admin membuka conversation PM tersebut.

### Langkah

1. Kirim pesan dari Admin.
2. Amati halaman PM tanpa reload.
3. Balas dari PM.
4. Amati halaman Admin tanpa reload.

### Hasil yang diharapkan

- Pesan tersimpan melalui HTTP POST `/chat/messages`.
- Penerima menerima notifikasi WebSocket.
- Isi pesan aktual dimuat melalui `GET /chat`, bukan dari payload WebSocket.
- Event hanya berisi tipe event, conversation ID, dan message ID.

### Lulus/gagal

- Lulus jika pesan muncul satu kali tanpa reload.
- Gagal jika pesan hanya muncul setelah reload, muncul dua kali, atau payload WebSocket memuat isi pesan.

## Skenario 3 — Percakapan PM tidak tercampur

### Persiapan

- Siapkan PM-A dan PM-B aktif.
- Admin memiliki akses ke keduanya.

### Langkah

1. Admin membuka conversation PM-A.
2. Kirim pesan khusus PM-A.
3. Pindah ke conversation PM-B.
4. Kirim pesan khusus PM-B.
5. Periksa kedua conversation sebagai Admin.

### Hasil yang diharapkan

- Pesan PM-A hanya muncul pada conversation PM-A.
- Pesan PM-B hanya muncul pada conversation PM-B.
- PM-A tidak menerima event PM-B.
- PM-B tidak menerima event PM-A.

### Lulus/gagal

- Lulus jika tidak ada pesan lintas conversation.
- Gagal jika event atau isi pesan muncul pada PM yang salah.

## Skenario 4 — Multi-tab Admin

### Persiapan

- Login sebagai Admin.
- Buka tiga tab Chat menggunakan session yang sama.

### Langkah

1. Pastikan ketiga tab membuka WebSocket.
2. Kirim pesan dari PM.
3. Amati ketiga tab Admin.

### Hasil yang diharapkan

- Ketiga tab dapat menerima notifikasi yang sama.
- Tidak ada tab yang menutup tab lain.
- Tab dengan conversation berbeda tidak mengubah isi conversation aktifnya.

### Lulus/gagal

- Lulus jika tiga koneksi aktif dan masing-masing bekerja.
- Gagal jika salah satu tab memutus tab lain atau Hub membatasi di bawah tiga koneksi.

## Skenario 5 — Reconnect setelah koneksi terputus

### Persiapan

- Buka Chat dan pastikan WebSocket aktif.
- Gunakan DevTools Network atau matikan sementara koneksi jaringan.

### Langkah

1. Putuskan jaringan atau tutup koneksi WebSocket melalui DevTools.
2. Amati Console dan Network.
3. Pulihkan jaringan.
4. Tunggu proses reconnect.

### Hasil yang diharapkan

- Client mencoba reconnect otomatis.
- Jeda reconnect meningkat bertahap dan dibatasi.
- Setelah tersambung, client memanggil `GET /chat` dengan `after_id` terakhir.
- Pesan yang belum terlihat dimuat tanpa duplikasi.

### Lulus/gagal

- Lulus jika koneksi pulih dan histori tersinkronisasi.
- Gagal jika reconnect membuka banyak koneksi bersamaan atau pesan hilang.

## Skenario 6 — Resync pesan ketika offline

### Persiapan

- Admin dan PM sudah membuka conversation.
- Putuskan koneksi WebSocket Admin tetapi biarkan session tetap aktif.

### Langkah

1. Kirim pesan dari PM ketika Admin offline realtime.
2. Pulihkan koneksi Admin.
3. Amati request `GET /chat?conversation_id=...&after_id=...`.

### Hasil yang diharapkan

- Pesan muncul setelah resync.
- Pesan tidak perlu dikirim ulang.
- Message ID terakhir bertambah secara monoton.

### Lulus/gagal

- Lulus jika pesan masuk setelah reconnect.
- Gagal jika pesan hanya muncul setelah reload penuh atau muncul ganda.

## Skenario 7 — Deduplikasi pesan

### Persiapan

- Buka DevTools Network.
- Siapkan satu conversation aktif.

### Langkah

1. Kirim satu pesan.
2. Amati response HTTP dan event WebSocket.
3. Simulasikan reconnect atau event yang sama datang lebih dari sekali jika alat browser memungkinkan.
4. Kirim ulang request dengan `client_message_id` yang sama.

### Hasil yang diharapkan

- Retry HTTP tidak membuat baris pesan baru.
- Event tidak menambah tampilan pesan kedua.
- Pesan dirender berdasarkan `message_id`.

### Lulus/gagal

- Lulus jika pesan tampil tepat satu kali.
- Gagal jika retry menggandakan pesan atau event lama diperlakukan sebagai pesan baru.

## Skenario 8 — Perpindahan conversation Admin

### Persiapan

- Admin memiliki minimal dua PM.

### Langkah

1. Buka PM-A.
2. Tunggu histori selesai dimuat.
3. Pindah ke PM-B.
4. Kirim pesan dari PM-A dan PM-B secara bergantian.

### Hasil yang diharapkan

- State message dan cursor PM-A tidak digunakan untuk PM-B.
- Event PM-A tidak mengubah isi panel PM-B.
- Saat kembali ke PM-A, histori benar dan tidak menggandakan pesan.

### Lulus/gagal

- Lulus jika panel selalu sesuai PM yang dipilih.
- Gagal jika pesan lama terbawa atau cursor salah conversation.

## Skenario 9 — Manager dan user tanpa session

### Persiapan

- Siapkan akun Manager.
- Gunakan browser profile tanpa cookie session.

### Langkah

1. Coba buka `/chat/page` sebagai Manager.
2. Coba buka `/chat/ws` sebagai Manager.
3. Coba buka kedua endpoint tanpa login.

### Hasil yang diharapkan

- Manager ditolak.
- User tanpa session diarahkan ke login atau menerima status unauthorized sesuai middleware.
- Tidak ada koneksi Manager yang masuk Hub.

### Lulus/gagal

- Lulus jika semua akses ditolak dan tidak ada event chat diterima.
- Gagal jika Manager dapat handshake atau melihat histori.

## Skenario 10 — Batas tiga koneksi per user

### Persiapan

- Login sebagai satu Admin.
- Buka Chat pada empat tab.

### Langkah

1. Buka tab pertama sampai ketiga dan pastikan WebSocket aktif.
2. Buka tab keempat.
3. Tutup salah satu dari tiga tab pertama.
4. Amati tab keempat atau lakukan reconnect.

### Hasil yang diharapkan

- Maksimal tiga koneksi aktif per user.
- Koneksi keempat ditolak/ditutup tanpa mengganggu tiga koneksi yang sudah ada.
- Setelah satu koneksi ditutup, slot dapat digunakan kembali.

### Lulus/gagal

- Lulus jika batas tiga koneksi dipertahankan.
- Gagal jika koneksi keempat menggantikan koneksi aktif atau jumlah koneksi tidak berkurang setelah tab ditutup.

## Skenario 11 — Disconnect dan cleanup

### Persiapan

- Buka satu atau lebih tab Chat.
- Siapkan akses log server bila tersedia.

### Langkah

1. Tutup tab.
2. Tunggu beberapa detik.
3. Buka tab baru dan lakukan handshake kembali.
4. Ulangi sampai batas koneksi pernah tercapai.

### Hasil yang diharapkan

- Koneksi tertutup dihapus dari Hub.
- Tidak ada koneksi zombie yang menghabiskan slot.
- Koneksi baru dapat menggunakan slot yang sudah kosong.

### Lulus/gagal

- Lulus jika cleanup konsisten.
- Gagal jika slot tetap penuh setelah semua tab ditutup.

## Risiko startup normal

`cmd/server/main.go` menjalankan urutan berikut:

```text
config.Load()
database.Connect(cfg.DatabaseURL)
database.Migrate(db)
database.Seed(db, ADMIN_PASSWORD)
router.NewWithHub(...)
```

Risikonya:

- Jika `DATABASE_URL` menunjuk ke `movara`, startup dapat melakukan AutoMigrate pada database utama.
- Seeder dapat membuat atau memperbarui data contoh pada database yang dituju.
- Menjalankan server normal bukan pengganti migration runner test yang fail-closed.
- `TEST_DATABASE_URL` tidak otomatis dipakai oleh startup server.

Karena itu pengujian manual harus dilakukan hanya setelah target `DATABASE_URL` dan hak aksesnya diverifikasi oleh operator. Jangan mengandalkan nama database saja sebagai bukti isolasi.

## Checklist akhir

- [ ] Admin dan PM berhasil login.
- [ ] WebSocket Admin dan PM tersambung.
- [ ] Pesan realtime diterima satu kali.
- [ ] Conversation PM tidak tercampur.
- [ ] Multi-tab maksimal tiga koneksi.
- [ ] Reconnect berhasil.
- [ ] Resync `after_id` berhasil.
- [ ] Retry idempoten tidak menggandakan pesan.
- [ ] Manager dan anonymous ditolak.
- [ ] Disconnect membersihkan slot Hub.
- [ ] Origin kosong/asing/port salah ditolak.
- [ ] Target database test telah diverifikasi terpisah dari `movara`.

