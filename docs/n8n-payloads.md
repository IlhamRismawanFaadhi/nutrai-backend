# Kontrak Payload Go <-> n8n

Semua request dari Go membawa header `X-N8N-SECRET: <N8N_WEBHOOK_SECRET>`.
Di n8n, tambahkan node **IF** / **Header Auth** pada Webhook untuk memvalidasinya.

---
## 1. AI Scan  (`N8N_SCAN_WEBHOOK_URL`)

### Go -> n8n  (`multipart/form-data`, Webhook node: Method POST, Binary Property Name = `image`)
| field        | tipe   | contoh                           |
|--------------|--------|----------------------------------|
| `image`      | file   | foto makanan (jpeg/png/webp)     |
| `request_id` | text   | `a1b2c3d4e5f60718`               |
| `user_id`    | text   | `12`                             |
| `allergies`  | text   | `["peanut","shellfish"]` (JSON string) |
| `is_halal`   | text   | `true`                           |
| `diet`       | text   | `high_protein`                   |

### n8n -> Go  (node "Respond to Webhook", JSON; objek tunggal atau array berisi 1 objek, keduanya diterima)
```json
{
  "item_name": "Chicken rice bowl",
  "estimated_calories": 640,
  "protein_g": 46,
  "carbs_g": 62,
  "fat_g": 18,
  "tags": ["indonesian", "rice", "chicken", "high protein"],
  "confidence": 0.82
}
```
Jika bukan makanan, balas `{"item_name": ""}` -> Go mengembalikan HTTP 422 `no_food_detected`.

### Respons Go -> frontend
```json
{ "success": true, "data": { "item_name": "Chicken rice bowl", "estimated_calories": 640,
  "protein_g": 46, "carbs_g": 62, "fat_g": 18, "tags": ["indonesian","rice","chicken","high protein"] } }
```
Frontend lalu mengisi form "Food or drink details"; user menekan **Add to today** (`POST /logs/meal`)
atau **Save as favorite meal** (`POST /favorites`).

Saran workflow n8n: Webhook -> (opsional) Compress/Resize image -> AI Vision node (prompt: "Return strict JSON ...")
-> Code node (validasi & normalisasi angka, cocokkan dengan dataset makanan Indonesia) -> Respond to Webhook.

---
## 2. AI Chat  (`N8N_CHAT_WEBHOOK_URL`)

### Go -> n8n  (`application/json`)
```json
{
  "request_id": "9f3c1a77be02d5e1",
  "timestamp": "2026-08-24T07:36:10Z",
  "message": "Suggest dinner",
  "intent": "suggest_dinner",
  "session_id": "optional-client-session",
  "history": [
    { "role": "user", "content": "What am I low on?" },
    { "role": "assistant", "content": "Protein and water." }
  ],
  "user": { "id": 12, "name": "Ari Rahman" },
  "context": {
    "date": "2026-08-24",
    "local_time": "14:36",
    "next_meal_type": "Dinner",
    "targets":   { "calories": 2200, "protein_g": 140, "carbs_g": 250, "fat_g": 70, "water_l": 2.5 },
    "consumed":  { "calories": 1420, "protein_g": 92, "carbs_g": 138, "fat_g": 46 },
    "remaining": { "calories_left": 780, "protein_gap_g": 48, "carbs_gap_g": 112, "fat_gap_g": 24, "water_gap_l": 2.5 },
    "water":     { "total_ml": 0, "target_l": 2.5, "left_ml": 2500 },
    "profile": {
      "diet_preference": "high_protein",
      "is_halal": true,
      "allergies": ["peanuts", "shellfish"],
      "budget_preference": "moderate"
    },
    "candidate_foods": [
      { "id": 24, "name": "Salmon quinoa bowl", "category": "Dinner", "serving": "1 mangkuk", "calories": 520,
        "protein_g": 38, "carbs_g": 46, "fat_g": 18, "tags": ["fish","halal","high protein"],
        "budget": "premium", "is_halal": true, "source": "estimate",
        "labels": ["Best dinner fit","Protein fit"], "reason": "Ranked for your protein goal...", "score": 78.4, "is_favorite": false }
    ],
    "today_logs": [
      { "item_name": "Greek yogurt parfait",  "meal_category": "Breakfast", "calories": 360, "protein_g": 28, "eaten_at": "08:00" },
      { "item_name": "Chicken rice bowl",     "meal_category": "Lunch",     "calories": 640, "protein_g": 46, "eaten_at": "12:30" },
      { "item_name": "Banana peanut toast",   "meal_category": "Snack",     "calories": 420, "protein_g": 18, "eaten_at": "16:30" }
    ]
  }
}
```

### n8n -> Go
```json
{
  "reply": "Kamu masih butuh ~48 g protein dan 780 kkal. Coba Salmon quinoa bowl (520 kkal, 38 g protein).",
  "suggestions": [
    { "item_name": "Salmon quinoa bowl", "category": "Dinner", "calories": 520,
      "protein_g": 38, "carbs_g": 46, "fat_g": 18, "tags": ["fish","high protein"],
      "reason": "Menutup gap protein, cocok halal." }
  ]
}
```
`reply` wajib tidak kosong. `suggestions` boleh kosong/tidak ada (Go menormalisasi menjadi `[]`).

`context.candidate_foods` = menu dari dataset di database, sudah difilter alergi/halal/diet/budget dan diurutkan. Minta LLM memilih/menjelaskan dari daftar ini saja (jangan mengarang menu).

Penting di workflow n8n: **filter hasil dataset** dengan `context.profile.allergies` & `is_halal`
sebelum/sesudah LLM, dan masukkan `remaining` ke system prompt agar rekomendasi sesuai sisa kalori/protein.

---
## 3. Catatan sync vs async
Implementasi saat ini **sync**: Go menunggu respons n8n (timeout per percobaan `N8N_TIMEOUT_SECONDS`,
retry `N8N_MAX_RETRIES` untuk error jaringan/5xx). Gunakan node **Respond to Webhook** (bukan "Immediately").
Jika kelak workflow > 60 detik, ubah ke pola async: Go membuat `job_id`, n8n membalas `202` lalu memanggil
callback `POST /internal/ai/callback`, dan frontend polling `GET /ai/jobs/:id`.
