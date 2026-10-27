# NutrAI Backend (Go + Gin + GORM + MySQL + n8n)

## Struktur
```
cmd/server/main.go          entrypoint + graceful shutdown
internal/config             env -> Config (support MYSQL_URL Railway)
internal/database           koneksi (retry) + AutoMigrate
internal/models             struct GORM (users, user_profiles, meal_logs, water_logs, favorite_meals)
internal/dto                request/response + payload n8n
internal/repository         akses DB
internal/service            logika bisnis (auth, profile, logs, ai)
internal/handler            HTTP handler (Gin)
internal/middleware         API key auth, CORS
internal/n8n                HTTP client webhook (retry, timeout, multipart)
internal/router             wiring DI + routes
internal/seed               dataset awal (foods.json 95 menu + ingredients.json 69 bahan, di-embed)
tools/gen_foods.py          generator: bahan x gram -> kalori/makro/tag menu
migrations/001_init.sql     DDL MySQL
migrations/002_foods.sql    DDL tabel foods
migrations/003_ingredients.sql  DDL ingredients + food_components
cmd/importer                CLI impor CSV sumber rujukan + hitung ulang gizi menu
data/recipes_template.csv   contoh format resep (bahan x gram)
docs/n8n-payloads.md        kontrak payload Go <-> n8n
```

## Jalankan lokal
```bash
cp .env.example .env        # isi DB_DSN & URL n8n
go mod tidy                 # menghasilkan go.sum (WAJIB sebelum docker build)
go run ./cmd/server
```
DDL manual (opsional, jika AUTO_MIGRATE=false): `mysql < migrations/001_init.sql`

## Dataset makanan & rekomendasi
- Tabel `foods` otomatis terisi 95 menu (Breakfast/Lunch/Dinner/Snack) saat server start; seed idempotent (menu yang sudah ada tidak ditimpa).
- **Cara angka dihitung:** kalori & makro tiap menu = jumlah (gram bahan x nilai per 100 g / 100) dari 69 bahan di tabel `ingredients` (resep di `food_components`, `source = "recipe"`). Tidak ada lagi angka per porsi yang ditebak langsung. `GET /foods/:id` menampilkan rincian per bahan.
- **Status verifikasi (jujur):** nilai per 100 g adalah nilai rujukan USDA yang saya masukkan dan sebagian saya turunkan sendiri (kolom `source` = `usda` / `derived` / `product`; semua `verified = false`). Hanya tempe yang saya cocokkan langsung ke USDA FoodData Central selama pengerjaan. Gramasi per porsi tetap asumsi porsi umum.
- **Review yang dibutuhkan hanya 69 baris bahan**, bukan 95 menu: buka `docs/ingredients-review.csv`. Setelah ahli gizi memverifikasi sebuah bahan, set `verified = 1`; seed tidak akan menimpanya lagi. Gramasi tiap menu: `docs/food-recipes.csv`.
- **Mengubah data:** edit `tools/gen_foods.py` (tabel bahan + resep), jalankan `python3 tools/gen_foods.py`, lalu deploy; seed otomatis memperbarui menu ber-source `recipe`/`estimate`. Menu ber-source `tkpi`/`custom`/`computed:*` tidak disentuh.
- **Sinkronisasi CSV <-> database** (agar keduanya selalu identik):
  ```bash
  go run ./cmd/importer export                       # DB -> docs/ingredients-review.csv, food-recipes.csv, foods.csv
  go run ./cmd/importer sync-review -dry-run         # lihat perubahan dari CSV bahan yang sudah direview
  go run ./cmd/importer sync-review                  # CSV bahan -> DB (+ verified), kalori menu dihitung ulang
  go run ./cmd/importer sync-recipes -dry-run        # CSV gramasi menu -> DB
  go run ./cmd/importer sync-recipes                 # menu yang diubah ditandai source=custom (tidak ditimpa seed)
  ```
  CSV tidak membuat bahan baru (hanya mengubah yang sudah ada, kunci = kolom `key`). Nilai tidak wajar ditolak. Setelah bahan diubah, seed saat server start selalu menghitung ulang kalori menu dari tabel bahan di DB, jadi hasil review tidak tertimpa.
- Tag alergen/vegan/vegetarian/halal kini **diturunkan otomatis dari bahan** (bukan ketikan manual). Alergen tersembunyi (mis. bumbu, kaldu instan, kontaminasi silang) tidak terdeteksi, jadi untuk alergi serius tetap tampilkan peringatan di aplikasi. "Halal" hanya berarti tidak ada bahan babi; sertifikasi/penyembelihan tidak diketahui.
- **Menaikkan akurasi data:** lihat `docs/DATA_SOURCES.md` (sumber rujukan TKPI/Kaggle/USDA + importer `go run ./cmd/importer`). `GET /ingredients?q=` mencari bahan per 100 g hasil impor.
- `GET /foods?q=&category=&tag=&safe=true&limit=&offset=` cari/jelajahi menu. `safe=true` (default) memfilter alergi, halal, diet, dan budget dari profil.
- `GET /suggestions?meal_type=Dinner&limit=6` rekomendasi untuk tab Suggest. Aturan (tanpa LLM): filter keras profil -> muat sisa kalori -> skor kecocokan kalori + gap protein, penalti karbo/lemak berlebih, bonus favorit. Respons berisi label ("Best dinner fit", "Protein fit", "Light option") dan teks alasan.
- **Gambar menu:** tiap menu punya `slug` (mis. `nasi-uduk-telur`). Taruh file `static/foods/<slug>.jpg` (disajikan di `/static/foods/...` dan ikut ter-copy ke Docker image) atau unggah ke CDN/R2/S3 dengan nama sama, lalu set `FOOD_IMAGE_BASE_URL`. Respons `/foods` & `/suggestions` memuat `image_url`. Untuk gambar khusus satu menu isi kolom `image_url` (dan `image_credit`). Daftar nama file + contoh prompt: `docs/food-images-manifest.csv`. Frontend perlu fallback placeholder karena file belum tentu ada.
- `POST /ai/chat` otomatis menyertakan `context.candidate_foods` (8 menu teratas dari dataset) agar jawaban AI berbasis data.

## Contoh cURL
```bash
# 1) Register -> simpan api_key
curl -X POST localhost:8080/api/v1/auth/register -H 'Content-Type: application/json' \
  -d '{"name":"Ari Rahman","email":"ari@example.com"}'

export KEY=nutrai_xxx

# 2) Profil
curl -X PUT localhost:8080/api/v1/profile -H "X-API-KEY: $KEY" -H 'Content-Type: application/json' \
  -d '{"calorie_target":2200,"protein_target":140,"carbs_target":250,"fat_target":70,"water_target_l":2.5,
       "diet_preference":"high_protein","is_halal":true,"allergies":["peanuts"],"budget_preference":"moderate"}'

# 3) Log makanan & air
curl -X POST localhost:8080/api/v1/logs/meal -H "Authorization: Bearer $KEY" -H 'Content-Type: application/json' \
  -d '{"item_name":"Chicken rice bowl","meal_category":"Lunch","calories":640,"protein_g":46,"carbs_g":62,"fat_g":18}'
curl -X POST localhost:8080/api/v1/logs/water -H "X-API-KEY: $KEY" -H 'Content-Type: application/json' -d '{"amount_ml":250}'
curl -X POST localhost:8080/api/v1/logs/water -H "X-API-KEY: $KEY" -H 'Content-Type: application/json' -d '{"reset":true}'

# 4) Today & history
curl localhost:8080/api/v1/logs/today -H "X-API-KEY: $KEY"
curl "localhost:8080/api/v1/logs/history?days=7" -H "X-API-KEY: $KEY"

# 4b) Dataset & rekomendasi
curl "localhost:8080/api/v1/foods?q=ayam&limit=10" -H "X-API-KEY: $KEY"
curl "localhost:8080/api/v1/suggestions?meal_type=Dinner" -H "X-API-KEY: $KEY"

# 5) AI
curl -X POST localhost:8080/api/v1/ai/scan -H "X-API-KEY: $KEY" -F image=@makanan.jpg
curl -X POST localhost:8080/api/v1/ai/chat -H "X-API-KEY: $KEY" -H 'Content-Type: application/json' \
  -d '{"message":"Suggest dinner","intent":"suggest_dinner"}'
```

## Deploy ke Railway
1. Push repo ke GitHub (pastikan `go.sum` ikut ter-commit).
2. Railway -> New Project -> Deploy from GitHub (otomatis memakai `Dockerfile` + `railway.json`; `Procfile` tidak diperlukan).
3. Tambah plugin **MySQL**, lalu di service app set variable: `MYSQL_URL=${{MySQL.MYSQL_URL}}`.
4. Set variable lain dari `.env.example` (`N8N_*`, `APP_TIMEZONE`, `CORS_ALLOWED_ORIGINS`, ...). Railway menyuplai `PORT` otomatis.
5. Healthcheck: `/healthz`.

## Catatan keamanan
- API key disimpan sebagai hash SHA-256; plaintext hanya muncul sekali saat register / rotate.
- `POST /auth/generate-key` merotasi key (butuh key lama), karena tanpa password endpoint publik akan memungkinkan pengambilalihan akun. Untuk "lupa key" tambahkan verifikasi email/OTP.
- Tambahkan rate limiting (mis. di Railway proxy/Cloudflare) terutama untuk `/auth/register` dan `/ai/*`.
