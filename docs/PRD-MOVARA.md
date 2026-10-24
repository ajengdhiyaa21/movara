# Product Requirements Document (PRD)

## MOVARA — Sistem Mutasi Barang Manna Kampus

**Versi:** 1.0  
**Tanggal:** 19 Oktober 2026  
**Status:** Draft untuk pengembangan dan validasi stakeholder

---

## 1. Ringkasan Produk

MOVARA adalah aplikasi internal untuk mencatat, memantau, dan melaporkan perpindahan barang antara area konter dan market pada setiap outlet Manna Kampus.

Aplikasi ini menggantikan pencatatan manual dengan proses berbasis barcode sehingga mutasi barang dapat dicatat lebih cepat, mengurangi input ganda, dan menghasilkan laporan yang terstruktur.

## 2. Latar Belakang dan Masalah

Proses mutasi barang membutuhkan pencatatan yang akurat berdasarkan:

- Outlet.
- Barcode dan kode produk.
- Nama produk.
- Kategori produk.
- Arah perpindahan barang.
- Jumlah barang.
- Beauty Advisor atau BA.
- Petugas yang melakukan input.
- Waktu transaksi.

Masalah yang ingin diselesaikan:

1. Input mutasi masih berisiko salah atau ganda.
2. Data produk dan kategori belum terpusat.
3. Riwayat mutasi sulit dicari jika data sudah banyak.
4. Admin membutuhkan laporan, filter, pagination, dan export.
5. Akses PM harus dibatasi berdasarkan outlet dan kategori produk.
6. Perubahan data jurnal perlu dikontrol dan tercatat dalam audit log.

## 3. Tujuan Produk

### Tujuan utama

- Mempercepat pencatatan mutasi barang menggunakan barcode.
- Memastikan PM hanya dapat mencatat produk sesuai outlet dan kategori akunnya.
- Menyediakan laporan mutasi yang dapat difilter dan diekspor.
- Memisahkan fungsi operasional PM dari fungsi administrasi.
- Menyediakan jejak audit untuk aktivitas penting.

### Indikator keberhasilan

- PM dapat menyimpan satu transaksi mutasi dalam waktu kurang dari 30 detik setelah barcode tersedia.
- Data mutasi yang tersimpan memiliki outlet, arah, jumlah, produk, kategori, user, dan waktu yang benar.
- Admin dapat menemukan transaksi berdasarkan periode, outlet, kategori, lokasi, atau kata kunci.
- Tidak ada transaksi PM yang tersimpan di luar outlet atau kategori yang diizinkan.
- Seluruh aksi administratif penting tercatat di audit log.

## 4. Ruang Lingkup

### Termasuk dalam produk

- Login dan session user.
- Manajemen role Admin, Manager, dan PM.
- Manajemen outlet.
- Manajemen kategori produk.
- Manajemen produk.
- Import produk melalui Excel.
- Template Excel produk.
- Akun PM dan penempatan outlet/kategori.
- Input mutasi dengan barcode.
- Dua arah mutasi: Konter ke Market dan Market ke Konter.
- Riwayat mutasi.
- Rekapitulasi mutasi.
- Filter, pagination, dan export Excel.
- Edit dan hapus jurnal oleh Admin.
- Audit log aktivitas sistem.
- Pengajuan produk baru oleh PM.

### Tidak termasuk dalam versi ini

- Integrasi langsung dengan sistem ERP atau POS.
- Perhitungan stok real-time sebagai sumber stok resmi.
- Pengadaan barang dan purchase order.
- Approval berjenjang untuk setiap mutasi normal.
- Aplikasi mobile native.
- Notifikasi push.

## 5. Pengguna dan Hak Akses

### 5.1 Admin

Admin memiliki akses penuh untuk:

- Melihat rekapitulasi seluruh outlet.
- Export laporan.
- Menambah, mengubah, mengaktifkan, dan menonaktifkan produk.
- Mengelola kategori.
- Mengelola outlet.
- Membuat dan mengubah akun PM.
- Mengubah password akun PM.
- Mengubah dan menghapus jurnal mutasi.
- Melihat audit log.
- Menyetujui data produk yang diajukan PM.

### 5.2 Manager

Manager dapat:

- Melihat rekapitulasi mutasi.
- Menggunakan filter laporan.
- Melihat statistik mutasi.
- Mencetak atau melihat laporan sesuai kewenangan.

Manager tidak dapat:

- Mengubah atau menghapus jurnal.
- Mengelola produk, outlet, kategori, atau akun PM.

### 5.3 PM / Pramuniaga

PM dapat:

- Melihat dashboard operasional outlet sendiri.
- Memilih arah mutasi.
- Scan atau mengetik barcode.
- Menyimpan mutasi barang.
- Melihat riwayat mutasi outletnya.
- Menghubungi Admin.
- Mengajukan produk yang belum terdaftar.
- Mengajukan perpindahan outlet.
- Mengubah profil dan password sendiri.

PM tidak dapat:

- Melihat data outlet lain.
- Menginput produk di luar outlet yang ditetapkan.
- Menginput produk di luar kategori akun PM.
- Mengubah atau menghapus jurnal.
- Mengelola master data.

## 6. Struktur Navigasi

### Navigasi umum

- Login.
- Profil.
- Ganti Password.
- Logout.

### Navigasi Admin dan Manager

- Rekap & Laporan.
- Master Data — Admin saja.
- Kelola Outlet & PM — Admin saja.

### Navigasi PM

- Dashboard/Input Mutasi.
- Pengajuan Pindah Outlet.
- Profil.
- Ganti Password.

## 7. Kebutuhan Fungsional

## FR-01 — Autentikasi

1. User dapat login menggunakan username dan password.
2. Username diproses secara konsisten, termasuk normalisasi huruf kecil.
3. Password disimpan dalam bentuk hash, bukan plaintext.
4. User nonaktif tidak dapat login.
5. Session menggunakan cookie HttpOnly.
6. Sistem mencatat login dan logout pada audit log.
7. Sistem menampilkan pesan umum ketika login gagal tanpa membocorkan detail akun.
8. Sistem harus memiliki rate limiting untuk percobaan login berulang.

## FR-02 — Dashboard PM

Dashboard PM harus menampilkan:

- Outlet dan kategori akun PM.
- Total mutasi hari ini.
- Total barang Konter ke Market.
- Total barang Market ke Konter.
- Tombol aksi cepat untuk dua arah mutasi.
- Riwayat mutasi terbaru.
- Tombol Chat Admin jika nomor Admin tersedia.

Riwayat PM sebaiknya menggunakan tabel responsif dengan kolom utama:

- Jam.
- Nama produk.
- Kategori.
- Qty.
- Lokasi.
- Status.
- Tombol Detail.

Detail tambahan dapat dibuka melalui popup atau drawer agar tidak menyebabkan scroll horizontal.

## FR-03 — Input Mutasi

1. PM memilih salah satu arah:
   - Konter ke Market.
   - Market ke Konter.
2. Sistem membuka form scan setelah arah dipilih.
3. PM dapat scan atau mengetik barcode.
4. Sistem mencari produk berdasarkan barcode dan outlet PM.
5. Sistem memvalidasi kategori produk dengan kategori akun PM.
6. Jika produk valid, sistem mengisi kode, nama, dan kategori secara otomatis.
7. PM mengisi qty dengan batas 1–999.
8. PM dapat mengisi nama BA secara opsional.
9. Sistem memberi peringatan jika barcode sudah diinput pada hari yang sama.
10. Sistem menyimpan outlet dan user berdasarkan session, bukan berdasarkan input form.
11. Setelah berhasil, form kembali siap digunakan untuk scan berikutnya.
12. Sistem mencatat transaksi dan audit log.

## FR-04 — Pengajuan Produk Baru

1. PM dapat mengajukan barcode yang belum ada di master data.
2. Pengajuan menyimpan barcode, nama produk, outlet, kategori PM, user, waktu, dan status pending.
3. Admin dapat melengkapi data produk dari master data.
4. Setelah produk dibuat, transaksi pending dihubungkan ke produk tersebut.
5. Sistem mencatat proses pengajuan dan approval pada audit log.

## FR-05 — Master Data Produk

Data produk minimal terdiri dari:

- Barcode.
- Kode produk.
- Nama produk.
- Kategori.
- Outlet.
- Status aktif/nonaktif.

Fitur:

- Tambah produk.
- Edit produk.
- Aktifkan/nonaktifkan produk.
- Cari produk.
- Filter outlet.
- Filter kategori.
- Pagination.
- Pilihan jumlah data per halaman.
- Import Excel.
- Download template Excel.

Validasi:

- Barcode wajib diisi.
- Kode wajib diisi.
- Nama wajib diisi.
- Outlet wajib valid dan aktif.
- Kategori wajib valid dan aktif.
- Barcode tidak boleh duplikat pada outlet yang sama.

## FR-06 — Master Data Kategori

Admin dapat:

- Melihat daftar kategori.
- Menambah kategori baru.
- Mengaktifkan atau menonaktifkan kategori.
- Menggunakan kategori pada produk.
- Menetapkan kategori pada akun PM.

Kategori nonaktif tidak boleh digunakan untuk produk atau akun PM baru.

## FR-07 — Manajemen Outlet

Admin dapat:

- Melihat daftar outlet.
- Menambah outlet.
- Menetapkan kode, nama, dan tipe outlet.
- Mengaktifkan atau menonaktifkan outlet.
- Melihat produk dan PM pada outlet terkait.

Kode outlet harus unik.

## FR-08 — Manajemen Akun PM

Admin dapat:

- Membuat akun PM.
- Menetapkan outlet PM.
- Menetapkan kategori PM.
- Mengubah nama dan username.
- Mengubah password.
- Mengaktifkan atau menonaktifkan akun.

Ketentuan:

- Username harus unik.
- Password minimum 8 karakter.
- Kategori dan outlet wajib dipilih.
- Password default tetap tidak boleh digunakan.
- Perubahan akun dicatat pada audit log.

## FR-09 — Rekapitulasi Mutasi

Halaman rekap harus menyediakan:

- Pencarian barcode, kode, dan nama produk.
- Filter outlet.
- Filter kategori.
- Filter lokasi perpindahan.
- Filter tanggal mulai dan tanggal akhir.
- Preset periode: Hari ini, 7 hari, 30 hari, Bulan ini, Semua.
- Pagination.
- Jumlah data per halaman.
- Total transaksi.
- Total quantity.
- Grafik volume mutasi per outlet.
- Export Excel.

Seluruh kartu ringkasan, total, grafik, tabel, pagination, dan export harus menggunakan filter yang sama agar tidak terjadi perbedaan angka.

Kolom jurnal:

- No.
- Outlet.
- Lokasi perpindahan.
- Tanggal dan jam.
- Barcode.
- Kode.
- Nama produk.
- Kategori.
- Qty.
- PM.
- BA.
- Aksi.

## FR-10 — Edit dan Hapus Jurnal

1. Hanya Admin yang dapat melihat tombol Edit dan Hapus.
2. Endpoint harus tetap dilindungi middleware role Admin.
3. Edit harus memvalidasi qty, nama produk, dan arah mutasi.
4. Hapus harus menampilkan popup konfirmasi.
5. Sistem mencatat user, waktu, dan ID mutasi yang diedit atau dihapus.
6. Untuk kebutuhan audit yang kuat, penghapusan sebaiknya diganti menjadi soft delete atau status void.

## FR-11 — Audit Log

Audit log minimal mencatat:

- Login.
- Logout.
- Perubahan password.
- Perubahan profil.
- Tambah/edit/aktifkan/nonaktifkan produk.
- Tambah/edit/aktifkan/nonaktifkan PM.
- Tambah kategori.
- Tambah outlet.
- Input mutasi.
- Edit mutasi.
- Hapus mutasi.
- Export laporan.
- Approval produk pending.

Audit log menyediakan:

- Filter tanggal.
- Pagination.
- Jumlah data per halaman.
- Waktu yang konsisten dengan timezone aplikasi.

## 8. Aturan Bisnis

1. Satu PM hanya dapat memiliki satu outlet aktif.
2. PM hanya dapat melihat dan memproses data outletnya.
3. Jika PM memiliki kategori, produk yang diproses harus memiliki kategori yang sama.
4. Produk nonaktif tidak boleh digunakan untuk mutasi baru.
5. Kategori nonaktif tidak boleh dipilih untuk data baru.
6. Qty mutasi harus antara 1 dan 999.
7. Arah mutasi hanya boleh bernilai `counter_to_market` atau `market_to_counter`.
8. Outlet, kategori, dan produk harus divalidasi ulang di server.
9. Data dari session menjadi sumber user dan outlet, bukan data dari browser.
10. Data jurnal menyimpan snapshot produk agar histori lama tidak berubah saat master produk diedit.

## 9. Kebutuhan Nonfungsional

### Keamanan

- Wajib menggunakan CSRF token pada seluruh request yang mengubah data.
- Session cookie menggunakan HttpOnly dan Secure pada production.
- Session secret production minimal 32 karakter dan wajib berasal dari environment variable.
- Password menggunakan bcrypt atau algoritma hashing yang setara.
- Login memiliki rate limiting.
- Semua endpoint administratif dilindungi role middleware.
- Semua input divalidasi di server.
- Error database tidak ditampilkan langsung ke user.

### Performa

- Halaman dengan ribuan data harus tetap menggunakan pagination server-side.
- Total dan grafik menggunakan query agregasi database.
- Tidak memuat seluruh tabel mutasi ke memory hanya untuk menghitung total.
- Query utama memiliki index yang sesuai.
- Target waktu respons halaman normal: di bawah 2 detik pada dataset operasional.

### Usability

- Layout responsif tanpa scroll horizontal untuk informasi utama.
- Tabel panjang menggunakan kolom prioritas dan detail popup/drawer.
- Scanner otomatis fokus setelah halaman input dibuka.
- Tombol aksi memiliki status loading/disabled saat diproses.
- Pesan berhasil dan gagal harus jelas serta dapat dipahami user nonteknis.

### Maintainability

- Handler, service, repository, model, dan template memiliki tanggung jawab terpisah.
- Logic bisnis tidak diletakkan di template.
- Semua operasi multi-langkah menggunakan transaksi database.
- Gunakan migration berversi untuk production, bukan hanya AutoMigrate.
- Tambahkan unit test dan integration test untuk aturan bisnis utama.

## 10. Kriteria Penerimaan Utama

### Login

- Login dengan akun aktif berhasil.
- Login dengan akun nonaktif gagal.
- Password salah menghasilkan pesan umum.
- Request POST tanpa CSRF token ditolak.

### PM

- PM tidak dapat mengakses halaman Admin.
- PM tidak dapat memproses produk di luar kategori akunnya.
- PM tidak dapat memproses produk dari outlet lain.
- Mutasi valid tersimpan dengan outlet dan user dari session.
- Barcode yang sama pada hari yang sama menghasilkan peringatan.

### Admin

- Admin dapat menambah kategori, outlet, produk, dan akun PM.
- Admin dapat mengedit dan menghapus jurnal.
- Popup konfirmasi muncul sebelum penghapusan.
- Semua aksi administratif tercatat di audit log.

### Rekap

- Filter outlet, kategori, lokasi, periode, dan pencarian bekerja bersamaan.
- Tabel dan total mengikuti filter yang sama.
- Pagination mempertahankan semua parameter filter.
- Export menghasilkan data sesuai filter aktif.
- Tabel tidak terpotong pada ukuran layar umum.

## 11. Risiko dan Mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| CSRF belum menggunakan token | Perubahan data tidak sah | Implementasi CSRF token dan validasi server |
| Password default mudah ditebak | Pengambilalihan akun | Password acak dan wajib ganti password |
| Data mutasi bertambah besar | Rekap lambat atau timeout | Agregasi SQL dan index |
| Filter tabel dan ringkasan berbeda | Keputusan laporan salah | Gunakan satu objek filter untuk seluruh komponen |
| Penghapusan permanen | Histori audit hilang | Gunakan soft delete/void |
| Kategori produk berubah | Akses PM dan histori tidak konsisten | Simpan snapshot kategori pada mutasi |
| Perbedaan timezone server | Laporan harian salah | Standarisasi ke WIB melalui clock service |
| Tidak ada test bisnis | Bug regresi sulit terdeteksi | Tambahkan test matrix berdasarkan role dan fitur |

## 12. Rencana Pengembangan

### Fase 1 — Stabilitas dan keamanan

- CSRF token.
- Validasi role dan scope.
- Penghapusan password default.
- Error handling database.
- Standarisasi timezone.

### Fase 2 — Konsistensi data dan performa

- Filter kategori PM pada riwayat.
- Sinkronisasi filter rekap dengan kartu dan grafik.
- Query agregasi SQL.
- Index database.
- Soft delete jurnal.

### Fase 3 — Penyempurnaan pengalaman pengguna

- Riwayat PM responsif dengan kolom prioritas.
- Detail transaksi melalui popup/drawer.
- Toast dan loading state.
- Perbaikan encoding ikon.
- Empty state dan error state yang konsisten.

### Fase 4 — Quality assurance

- Unit test service.
- Integration test route dan middleware.
- Test role matrix.
- Test import Excel.
- Test pagination dan filter kombinasi.
- Test keamanan dasar.

## 13. Matriks Pengujian Role

| Fitur | Admin | Manager | PM |
|---|---:|---:|---:|
| Login | Ya | Ya | Ya |
| Input mutasi | Tidak wajib | Tidak | Ya |
| Lihat rekap | Ya | Ya | Tidak |
| Export rekap | Ya | Tidak | Tidak |
| Edit jurnal | Ya | Tidak | Tidak |
| Hapus jurnal | Ya | Tidak | Tidak |
| Kelola produk | Ya | Tidak | Tidak |
| Kelola kategori | Ya | Tidak | Tidak |
| Kelola outlet | Ya | Tidak | Tidak |
| Kelola akun PM | Ya | Tidak | Tidak |
| Lihat audit log | Ya | Tidak | Tidak |
| Ubah profil sendiri | Ya | Ya | Ya |
| Ganti password sendiri | Ya | Ya | Ya |

## 14. Definisi Selesai

Fitur dianggap selesai apabila:

- Kebutuhan fungsional telah diimplementasikan.
- Validasi server dan role telah diuji.
- Tidak ada error kritis atau high yang terbuka.
- Test otomatis berhasil.
- Filter dan pagination telah diuji dalam kombinasi.
- Data audit tercatat dengan benar.
- UI dapat digunakan pada desktop dan mobile.
- Dokumentasi penggunaan dan deployment tersedia.

