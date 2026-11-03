# MOVARA Internal Chat — Requirements

Status: Draft untuk persetujuan stakeholder. Belum diimplementasikan.

## 1. Tujuan

Menyediakan komunikasi teks internal antara Admin dan masing-masing PM melalui aplikasi MOVARA, sehingga komunikasi operasional tidak bergantung sepenuhnya pada WhatsApp dan tetap memiliki konteks akun pengguna.

## 2. Dasar dari PRD MOVARA

PRD secara eksplisit menyebutkan:

- PM dapat menghubungi Admin.
- Dashboard PM menampilkan tombol Chat Admin jika nomor Admin tersedia.
- Role Admin, Manager, dan PM harus dipisahkan.
- Aksi penting perlu dicatat dalam audit log.
- PM dibatasi berdasarkan outlet dan kategori.

PRD belum secara eksplisit menentukan:

- chat internal tersimpan atau tidak;
- satu atau banyak percakapan per PM;
- status terbaca;
- notifikasi;
- real-time;
- lampiran;
- retensi pesan;
- akses Manager;
- pembagian percakapan antar Admin.

## 3. Kondisi saat ini

Sudah tersedia:

- login/session;
- role Admin, Manager, PM;
- outlet dan kategori pada user;
- tombol Chat Admin berbasis WhatsApp;
- audit log umum.

Belum tersedia:

- conversation;
- message;
- chat API;
- koneksi real-time;
- unread count;
- halaman/popup chat internal;
- read receipt;
- moderasi atau penghapusan pesan.

## 4. Functional requirements yang diusulkan

### MVP wajib

FR-C01. Admin dapat melihat daftar PM yang aktif.

FR-C02. PM dapat membuka percakapan dengan Admin.

FR-C03. Admin dapat membuka percakapan dengan PM tertentu.

FR-C04. User hanya dapat membaca percakapan yang diizinkan oleh role dan participant.

FR-C05. Pesan teks memiliki pengirim, isi, waktu dibuat, dan conversation ID.

FR-C06. Pesan disimpan ke database.

FR-C07. Riwayat pesan menggunakan pagination atau cursor.

FR-C08. Pesan kosong, terlalu panjang, atau tidak valid ditolak server.

FR-C09. Pesan baru memiliki status unread/read jika keputusan ini disetujui.

FR-C10. Akun PM nonaktif tidak dapat mengirim pesan baru.

FR-C11. Histori percakapan tetap tersedia untuk Admin sesuai kebijakan retensi.

FR-C12. Pengiriman dan pembacaan pesan dicatat secukupnya untuk audit teknis, tanpa memasukkan seluruh isi pesan ke audit log umum kecuali disetujui.

### Fitur yang menunggu persetujuan

- indikator online/offline;
- typing indicator;
- pesan terkirim/terbaca;
- unread badge pada topbar;
- pencarian percakapan;
- lampiran gambar/file;
- penghapusan atau edit pesan;
- broadcast Admin ke banyak PM;
- akses Manager;
- notifikasi browser/suara.

## 5. Matriks akses usulan

| Aksi | Admin | Manager | PM |
|---|---:|---:|---:|
| Melihat daftar PM aktif | Ya | Menunggu keputusan | Hanya dirinya/percakapan sendiri |
| Membuat pesan ke PM | Ya | Menunggu keputusan | Tidak |
| Membuat pesan ke Admin | Ya | Menunggu keputusan | Ya |
| Membaca percakapan PM lain | Ya, sesuai kebijakan | Menunggu keputusan | Tidak |
| Menghapus pesan | Menunggu keputusan | Menunggu keputusan | Tidak |
| Mengirim lampiran | Menunggu keputusan | Menunggu keputusan | Menunggu keputusan |
| Melihat histori setelah PM nonaktif | Menunggu keputusan | Menunggu keputusan | Tidak login |

## 6. Alur utama

### PM

1. PM login.
2. PM membuka tombol Chat Admin.
3. Sistem mengambil conversation PM–Admin yang diizinkan.
4. PM membaca histori terbaru.
5. PM mengirim pesan teks.
6. Sistem memvalidasi session, role, participant, ukuran, dan isi.
7. Pesan disimpan dan dikirim ke Admin melalui mekanisme komunikasi yang dipilih.

### Admin

1. Admin membuka menu chat.
2. Admin melihat daftar PM dan unread count jika disetujui.
3. Admin memilih PM.
4. Admin membaca histori dan mengirim balasan.

## 7. Non-functional requirements

- Tidak boleh ada akses lintas percakapan melalui manipulasi ID.
- Session dan role tetap diverifikasi di server.
- Pesan teks tidak boleh dieksekusi sebagai HTML/JavaScript.
- Batas pesan awal disarankan 2.000–4.000 karakter.
- Pagination histori wajib dilakukan di server.
- Pengiriman berulang harus memiliki idempotency atau deduplication strategy.
- Koneksi real-time harus dapat reconnect tanpa menggandakan pesan.
- Error koneksi harus memiliki fallback yang mudah dipahami.
- Semua perubahan schema harus menggunakan migration yang dapat diulang dan diaudit.

## 8. Acceptance criteria MVP

- PM A tidak dapat membaca percakapan PM B.
- PM tidak dapat mengirim pesan dengan user ID palsu.
- Admin dapat membalas PM tertentu.
- User nonaktif tidak dapat membuat pesan baru.
- Pesan tetap muncul setelah refresh.
- Pesan tidak hilang ketika koneksi sementara terputus.
- Payload berisi HTML/script tidak dieksekusi.
- Pesan melebihi batas ditolak.
- Histori lama memiliki pagination.
- Test authorization berhasil untuk Admin, Manager, dan PM.

## 9. Pertanyaan terbuka

1. Apakah satu PM memiliki satu conversation privat dengan semua Admin?
2. Apakah setiap Admin hanya melihat PM outlet tertentu?
3. Apakah Manager boleh membaca atau mengirim chat?
4. Apakah histori chat tetap disimpan setelah akun PM dinonaktifkan?
5. Berapa lama retensi pesan?
6. Apakah isi chat termasuk data yang harus masuk audit log?
7. Apakah chat hanya teks?
8. Apakah dibutuhkan unread count dan read receipt?
9. Apakah WhatsApp tetap ditampilkan sebagai alternatif?
10. Apakah pesan dapat dikirim ketika user offline dan diterima saat login kembali?
