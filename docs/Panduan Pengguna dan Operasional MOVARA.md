# Panduan Pengguna dan Operasional MOVARA

## 1. Tujuan Aplikasi

MOVARA digunakan untuk mengelola produk, akun pengguna, outlet, dan pencatatan mutasi barang antara konter dan market. Data mutasi menjadi dasar jurnal, rekapitulasi, export Excel, dan cetak PDF.

## 2. Pembagian Hak Akses

| Role | Fungsi utama |
|---|---|
| Pramuniaga | Input mutasi pada outlet dan kategori yang ditetapkan, melihat riwayat mutasi, serta mengajukan perpindahan outlet bila tersedia. |
| Manager | Melihat rekapitulasi mutasi, memfilter data, export Excel, dan mencetak PDF. |
| Admin B | Mengakses fungsi administrasi sesuai hak akses akun, terutama pemantauan data. Fitur sensitif akun dibatasi. |
| Admin A / Super Admin | Mengelola master data, outlet, akun, pengajuan produk baru, jurnal, dan aktivitas aplikasi. |

Catatan keamanan: semua akun hanya dapat aktif pada satu perangkat atau browser dalam satu waktu. Login kedua akan ditolak sampai session pertama logout atau kedaluwarsa.

## 3. Alur Singkat Sistem

```text
Login dengan ID pengguna
        ↓
Sistem memeriksa akun, password, status aktif, dan session perangkat
        ↓
  ┌───────────────┬──────────────────┐
  │ Pramuniaga    │ Manager / Admin  │
  └──────┬────────┴─────────┬────────┘
         ↓                  ↓
 Input mutasi          Rekap & laporan
         ↓                  ↓
 Jurnal mutasi        Excel / Cetak PDF

Admin A mengelola master data, outlet, akun, pengajuan produk, dan audit aktivitas.
```

## 4. Login dan Session

1. Buka halaman login.
2. Masukkan **ID Pengguna**.
3. Username akan tampil otomatis apabila ID terdaftar.
4. Masukkan password lalu login.
5. Jika akun sedang aktif di perangkat lain, sistem menampilkan warning dan login tidak dilanjutkan.
6. Untuk pindah perangkat, logout dari perangkat pertama terlebih dahulu.
7. Gunakan tombol **Keluar** setelah selesai bekerja.

Session berlaku maksimal 8 jam. Jika browser ditutup tanpa logout, akun dapat tetap dianggap aktif sampai session kedaluwarsa.

## 5. Operasional Pramuniaga

### 5.1 Input Mutasi

1. Masuk ke menu **Mutasi**.
2. Pilih atau buka form input mutasi.
3. Scan atau ketik barcode produk.
4. Jika produk terdaftar di master data, SKU/PLU dan nama produk terisi otomatis serta terkunci.
5. Jika produk belum terdaftar, lengkapi nama produk dan gunakan alur **Ajukan Produk Baru**.
6. Pilih arah perpindahan sesuai kondisi barang:
   - Konter → Market
   - Market → Konter
7. Masukkan jumlah mutasi. Jumlah tidak boleh 0.
8. Isi nama Beauty Advisor (BA) jika diperlukan.
9. Klik **Simpan Mutasi**.
10. Pastikan notifikasi berhasil muncul dan data masuk ke riwayat.

### 5.2 Aturan Input

- Outlet dan kategori mengikuti akun Pramuniaga yang sedang login.
- Nama produk dari master data tidak dapat diubah saat produk ditemukan.
- Barcode atau produk yang belum ada dapat diajukan untuk ditinjau Admin.
- Jangan menyimpan form sebelum jumlah mutasi diisi.
- Setelah berhasil menyimpan atau mengajukan, form harus di-reset sebelum input berikutnya.

### 5.3 Riwayat Mutasi

Gunakan menu riwayat untuk mencari data berdasarkan tanggal, barcode, produk, atau arah perpindahan. Perubahan jumlah hanya dilakukan melalui fungsi edit yang tersedia dan harus sesuai prosedur operasional.

## 6. Pengajuan Produk Baru

Pengajuan digunakan bila barcode hasil scan belum tersedia di Master Data.

1. Isi barcode dan nama produk.
2. Lengkapi jumlah mutasi dan data BA bila diperlukan.
3. Kirim pengajuan produk baru.
4. Admin meninjau data pada menu **Produk Baru**.
5. Admin dapat melengkapi PLU/SKU dan kategori, lalu menyetujui atau menolak pengajuan.
6. Produk yang disetujui masuk ke Master Data dan dapat digunakan pada input berikutnya.

## 7. Operasional Manager

1. Login menggunakan ID pengguna Manager.
2. Buka menu **Rekap & Laporan**.
3. Gunakan filter periode, outlet, kategori, arah perpindahan, atau pencarian bila diperlukan.
4. Periksa kartu ringkasan, jurnal mutasi, dan volume mutasi per outlet.
5. Pilih **Export Excel (.xlsx)** untuk pengolahan lanjutan.
6. Pilih **Cetak / PDF** untuk membuka tampilan cetak ukuran A4 portrait.

Manager bersifat read-only terhadap data operasional dan tidak mengelola master data maupun akun.

## 8. Operasional Admin A / Super Admin

### 8.1 Master Data Produk

- Tambah produk secara manual melalui **Tambah Produk**.
- Import produk menggunakan template Excel.
- Edit barcode, PLU/SKU, nama produk, outlet, dan kategori sesuai kewenangan.
- Nonaktifkan produk yang sudah tidak digunakan.
- Gunakan filter outlet, kategori, pencarian, dan pagination untuk pemeriksaan data.

### 8.2 Import Produk Massal

Format kolom yang didukung:

```text
barcode | plu_sku | nama_produk | kategori
```

Alur yang disarankan:

1. Download template Excel.
2. Isi data tanpa mengubah nama header.
3. Pilih outlet tujuan dan kategori default.
4. Unggah file `.xlsx`.
5. Periksa ringkasan jumlah berhasil dan dilewati.
6. Bila ada baris dilewati, perbaiki file sumber lalu import ulang hanya data tersebut.

Batas operasional saat ini adalah maksimal 20.000 baris per file dan ukuran file maksimal 25 MB. Import disimpan secara batch dalam transaksi; bila penyimpanan gagal, data baru dari proses tersebut dibatalkan.

### 8.3 Kelola Kategori dan Outlet

- Tambahkan kategori sebelum import bila kategori belum tersedia.
- Pastikan kode outlet unik dan nama outlet sesuai standar operasional.
- Nonaktifkan outlet hanya setelah memastikan tidak ada proses operasional aktif yang bergantung padanya.

### 8.4 Kelola Akun dan Aktivitas

Menu **Kelola Akun & Aktivitas** memiliki tab:

- **Akun Pramuniaga**: melihat akun, outlet, kategori, status, dan pengelolaan akun.
- **Akun Manager**: melihat dan mengelola akun Manager.
- **Aktivitas Aplikasi**: memeriksa log login, perubahan data, export, reset password, dan aktivitas penting lainnya.

Gunakan tombol **Tambah Akun** untuk memilih jenis akun Pramuniaga atau Manager. Fitur lihat/reset password hanya tersedia untuk Super Admin, memerlukan konfirmasi, dan tercatat di log aktivitas.

## 9. Pemeriksaan Harian Operasional

### Awal hari

- Pastikan outlet aktif.
- Pastikan akun pengguna yang bertugas aktif dan dapat login.
- Periksa Master Data untuk produk yang akan digunakan.

### Selama operasional

- Input mutasi segera setelah perpindahan barang terjadi.
- Pastikan arah perpindahan dan jumlah PCS benar.
- Jangan menggunakan akun pengguna lain.
- Jika produk tidak ditemukan, gunakan pengajuan produk baru; jangan memasukkan nama produk master secara sembarang.

### Akhir hari

- Periksa riwayat mutasi dan total PCS.
- Bandingkan arah Konter → Market dan Market → Konter.
- Tindak lanjuti data pending atau produk baru yang belum diproses.
- Logout dari aplikasi.

## 10. Penanganan Masalah Umum

| Masalah | Tindakan |
|---|---|
| ID pengguna tidak ditemukan | Periksa kembali ID. Jika tetap tidak ditemukan, hubungi Admin. |
| Login ditolak karena perangkat lain | Logout dari perangkat pertama atau tunggu session maksimal 8 jam berakhir. |
| Password salah | Pastikan password benar. Hubungi Super Admin bila perlu reset. |
| Produk tidak ditemukan | Pastikan barcode benar, lalu ajukan produk baru bila memang belum ada. |
| Jumlah mutasi ditolak | Isi jumlah lebih dari 0 sebelum menyimpan. |
| Import Excel gagal | Gunakan template resmi, pastikan format `.xlsx`, header lengkap, kategori valid, ukuran file maksimal 25 MB, dan jumlah maksimal 20.000 baris. |
| Banyak baris import dilewati | Baca detail baris yang ditampilkan, perbaiki duplikat atau field wajib, lalu import ulang data yang bermasalah. |
| PDF tidak muncul | Pastikan popup diizinkan, buka tombol Cetak/PDF di tab baru, lalu gunakan dialog print browser untuk memilih Save as PDF. |
| Data tidak terlihat setelah perubahan | Refresh halaman dan periksa filter, outlet, periode, serta status aktif. |

## 11. Prinsip Data dan Keamanan

- Setiap pengguna memakai akun sendiri.
- Jangan membagikan password.
- Logout setelah selesai.
- Jangan menghapus atau mengubah data historis tanpa otorisasi.
- Lakukan export atau backup sesuai kebijakan organisasi sebelum perubahan massal.
- Untuk import besar, gunakan file yang sudah divalidasi dan simpan salinan file sumber.
- Periksa log aktivitas bila terjadi perubahan yang tidak dikenali.

## 12. Checklist Sebelum Go-Live

- [ ] Semua outlet dan kategori sudah dibuat.
- [ ] Produk utama sudah tersedia di Master Data.
- [ ] ID pengguna dan role sudah sesuai.
- [ ] Password awal sudah disampaikan melalui kanal aman.
- [ ] Pengguna memahami arah perpindahan dan kewajiban mengisi jumlah.
- [ ] Uji login setiap role berhasil.
- [ ] Uji import file kecil berhasil sebelum import massal.
- [ ] Export Excel dan cetak PDF sudah diuji.
- [ ] Prosedur backup dan penanggung jawab support sudah ditetapkan.
