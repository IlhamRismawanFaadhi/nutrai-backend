# Sumber Data Gizi & Cara Mengimpor

Status saat ini: 95 menu di `foods` **dihitung dari 69 bahan** (`tools/gen_foods.py`, `source = "recipe"`), bukan lagi tebakan per porsi.
Nilai per 100 g bahan masih belum terverifikasi penuh (lihat `docs/ingredients-review.csv`, kolom `source` dan `verified`).
Dokumen ini menjelaskan cara menggantinya dengan data resmi (TKPI/USDA) lewat importer, sehingga setiap angka bisa ditelusuri.
`foods` ber-source `tkpi`/`custom`/`computed:*` tidak ditimpa oleh seed otomatis.

## Sumber rujukan (hasil pencarian, Oktober 2026)

| Sumber | Isi | Lisensi / catatan |
|---|---|---|
| **TKPI 2017** (Tabel Komposisi Pangan Indonesia), Kemenkes RI | Data resmi nasional. Buku 135 hal., ISBN 9786024164072; cetakan revisi 2020 juga ada di repository.kemkes.go.id. Sebuah makalah Poltekkes menyebut TKPI 2017 memuat 1.113 jenis pangan. Versi web: panganku.org | **Syarat pemakaian ulang belum saya verifikasi.** Cek halaman sumber dan cantumkan atribusi "Kemenkes RI, TKPI 2017". |
| **Kaggle: "Indonesian Food And Drink Nutrition Dataset"** (anasfikrihanif) | 1.346 baris, kolom `id, calories, proteins, fat, carbohydrate, name, image`, nilai **per 100 g**. Menurut deskripsinya diambil dari TKPI via panganku.org, lalu dibersihkan dan ditambah link gambar | **Lisensi tidak terlihat di halaman yang bisa saya akses** (saya hanya melihat mirror). Buka halaman Kaggle-nya dan cek bagian License. **Jangan pakai kolom `image`** (link gambar pihak ketiga, hak pakainya tidak jelas). |
| **Mendeley Data: "Nutritional Analysis and Macro-Micro Nutrient Profiling of Indonesian Culinary Recipes"** (DOI 10.17632/8b4ztns76h.3, terbit 11 Mei 2026) | Resep masakan Indonesia dengan porsi; tiap bahan dikonversi ke gram lalu dihitung gizinya memakai TKPI | Kolom lisensi kosong pada halaman yang saya baca; **cek di halaman Mendeley**. Penulisnya mencatat keterbatasan: bahan yang tidak ada di TKPI dan konversi takaran "secukupnya". Paling dekat dengan kebutuhan kita (level masakan). |
| **USDA FoodData Central** | Bahan umum/Internasional (salmon, quinoa, oat, dll.) | Domain publik, CC0 1.0. Tidak perlu izin, tapi mereka meminta FoodData Central dicantumkan sebagai sumber. Unduh: fdc.nal.usda.gov/download-datasets |

## Alur yang disarankan (dua tingkat)

```
dataset bahan per 100 g (TKPI/Kaggle, USDA)  -->  tabel ingredients
resep menu (bahan x gram)                    -->  tabel food_components
gizi menu = sum(gram x nilai per 100 g / 100) -->  foods.calories/protein/carbs/fat, source = "computed:tkpi"
```

### 1. Unduh dataset (saya tidak bisa melakukannya: Kaggle butuh login)
Unduh CSV dari Kaggle (atau ekspor dari sumber lain) ke mesin Anda. Pastikan lisensinya mengizinkan penggunaan Anda.

### 2. Impor bahan per 100 g
```bash
# cek dulu tanpa menulis ke DB
go run ./cmd/importer ingredients -file indonesian_food_nutrition.csv -source tkpi -dry-run
go run ./cmd/importer ingredients -file indonesian_food_nutrition.csv -source tkpi
```
Header yang dikenali: `name/nama`, `calories/kalori/energi/kcal`, `proteins/protein`, `fat/lemak`, `carbohydrate/karbohidrat`, `id`.
Jika energi dalam kJ: tambahkan `-energy-unit kj`. **Periksa satuan** pada contoh yang dicetak (nasi matang sekitar 130-180 kkal/100 g);
deskripsi Kaggle menulis "cal", yang bisa berarti kkal.
Baris dengan nilai tidak wajar (kalori > 950 atau makro > 100 g per 100 g) dilewati dan dilaporkan.

### 3. Cari nama bahan yang persis
```bash
go run ./cmd/importer find -q "nasi"
go run ./cmd/importer find -q "dada ayam"
```
API juga menyediakan `GET /api/v1/ingredients?q=nasi` untuk pencarian di aplikasi.

### 4. Isi resep dan hitung ulang
Salin `data/recipes_template.csv`, ganti nama bahan dengan nama persis (atau isi `ingredient_id`) dan isi gram per porsi.
```bash
go run ./cmd/importer components -file data/recipes.csv -dry-run   # lihat perbandingan lama vs baru
go run ./cmd/importer components -file data/recipes.csv
```
Output menandai `REVIEW` jika hasil hitung berbeda lebih dari 25% dari estimasi lama (kemungkinan salah pilih bahan atau gram).
Menu yang punya baris bermasalah dilewati seluruhnya (tidak ada update parsial).

## Jebakan akurasi yang perlu diwaspadai
1. **Keadaan bahan**: "beras" (mentah) vs "nasi" (matang) berbeda sekitar 2x lipat per 100 g. Pilih entri yang sesuai bentuk yang dimakan.
2. **Minyak dan santan**: gorengan dan masakan bersantan menyerap lemak yang jarang tercatat. Tambahkan sebagai bahan tersendiri (gram minyak).
3. **Gram per porsi** tetap asumsi. Idealnya diambil dari resep baku atau ditimbang. Dataset Mendeley menyediakan porsi untuk banyak masakan.
4. **Bahan tidak ada di TKPI** (mis. salmon, quinoa): gunakan USDA dengan `-source usda`, atau `custom` untuk data label kemasan.
5. **Tag alergen, halal, dan diet** pada `foods` tetap kurasi manual dan bukan hasil impor. Untuk alergi serius, tampilkan peringatan di aplikasi.
6. Ubah tampilan di aplikasi: tandai angka `estimate` sebagai "perkiraan", dan `computed:*` sebagai "dihitung dari <sumber>".

## Atribusi (disarankan di layar About/Profile aplikasi)
- Kementerian Kesehatan RI, Tabel Komposisi Pangan Indonesia (TKPI) 2017.
- U.S. Department of Agriculture, Agricultural Research Service. FoodData Central. https://fdc.nal.usda.gov/
