// Command importer mengimpor data gizi dari file CSV sumber rujukan (mis. dataset Kaggle turunan TKPI)
// dan menghitung ulang gizi menu berdasarkan resep (bahan x gram).
//
//	go run ./cmd/importer ingredients -file tkpi.csv -source tkpi [-dry-run]
//	go run ./cmd/importer find -q "nasi"
//	go run ./cmd/importer components -file recipes.csv [-dry-run]
package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"nutrai/internal/config"
	"nutrai/internal/database"
	"nutrai/internal/models"
	"nutrai/internal/repository"
)

func main() {
	_ = godotenv.Load()
	if len(os.Args) < 2 {
		usage()
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	db, err := database.Connect(cfg.DSN)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	switch os.Args[1] {
	case "ingredients":
		cmdIngredients(db, os.Args[2:])
	case "components":
		cmdComponents(db, os.Args[2:])
	case "find":
		cmdFind(db, os.Args[2:])
	case "export":
		cmdExport(db, os.Args[2:])
	case "sync-review":
		cmdSyncReview(db, os.Args[2:])
	case "sync-recipes":
		cmdSyncRecipes(db, os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Println(`Usage:
  importer ingredients -file <csv> [-source tkpi|usda|custom] [-energy-unit kcal|kj] [-dry-run]
  importer find        -q <kata kunci> [-limit 20]
  importer components  -file <recipes.csv> [-dry-run]
  importer export       [-dir docs]                      (DB -> ingredients-review.csv, food-recipes.csv, foods.csv)
  importer sync-review  [-file docs/ingredients-review.csv] [-dry-run]   (CSV bahan -> DB, menu dihitung ulang)
  importer sync-recipes [-file docs/food-recipes.csv] [-dry-run]         (CSV gramasi menu -> DB)

recipes.csv kolom: food_name, ingredient (atau ingredient_id), grams`)
	os.Exit(2)
}

// ---------- helpers CSV ----------
func readCSV(path string) (map[string]int, [][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	hdr := map[string]int{}
	var rows [][]string
	first := true
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		if first {
			for i, h := range rec {
				h = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, "\ufeff")))
				hdr[h] = i
			}
			first = false
			continue
		}
		rows = append(rows, rec)
	}
	return hdr, rows, nil
}

func colIdx(h map[string]int, names ...string) int {
	for _, n := range names {
		if i, ok := h[n]; ok {
			return i
		}
	}
	return -1
}

func get(rec []string, i int) string {
	if i < 0 || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

func num(s string) (float64, error) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	if s == "" || s == "-" || strings.EqualFold(s, "tr") || strings.EqualFold(s, "t") {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }

func headerKeys(h map[string]int) []string {
	k := make([]string, 0, len(h))
	for n := range h {
		k = append(k, n)
	}
	sort.Strings(k)
	return k
}

// ---------- ingredients ----------
func cmdIngredients(db *gorm.DB, args []string) {
	fs := flag.NewFlagSet("ingredients", flag.ExitOnError)
	file := fs.String("file", "", "path CSV (wajib)")
	source := fs.String("source", "tkpi", "tkpi | usda | custom")
	unit := fs.String("energy-unit", "kcal", "satuan energi di CSV: kcal | kj")
	dry := fs.Bool("dry-run", false, "validasi saja, tidak menulis ke DB")
	_ = fs.Parse(args)
	if *file == "" {
		log.Fatal("-file wajib diisi")
	}
	hdr, rows, err := readCSV(*file)
	if err != nil {
		log.Fatalf("baca CSV: %v", err)
	}
	iName := colIdx(hdr, "name", "nama", "nama bahan", "description")
	iCal := colIdx(hdr, "calories", "kalori", "energi", "energy", "kcal")
	iProt := colIdx(hdr, "proteins", "protein")
	iFat := colIdx(hdr, "fat", "lemak")
	iCarb := colIdx(hdr, "carbohydrate", "carbohydrates", "carbs", "karbohidrat")
	iID := colIdx(hdr, "id")
	if iName < 0 || iCal < 0 || iProt < 0 || iFat < 0 || iCarb < 0 {
		log.Fatalf("kolom wajib (name, calories, protein, fat, carbohydrate) tidak lengkap. Header ditemukan: %v", headerKeys(hdr))
	}

	var added, updated, skipped int
	seen := map[string]bool{}
	for n, rec := range rows {
		name := get(rec, iName)
		if name == "" || seen[strings.ToLower(name)] {
			skipped++
			continue
		}
		seen[strings.ToLower(name)] = true
		cal, e1 := num(get(rec, iCal))
		p, e2 := num(get(rec, iProt))
		f, e3 := num(get(rec, iFat))
		c, e4 := num(get(rec, iCarb))
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
			log.Printf("baris %d (%s): angka tidak valid, dilewati", n+2, name)
			skipped++
			continue
		}
		if *unit == "kj" {
			cal /= 4.184
		}
		if cal < 0 || p < 0 || f < 0 || c < 0 || cal > 950 || p > 100 || f > 100 || c > 100 {
			log.Printf("baris %d (%s): nilai di luar batas wajar per 100 g (cek satuan), dilewati", n+2, name)
			skipped++
			continue
		}
		ing := models.Ingredient{Name: name, Calories100: round2(cal), Protein100: round2(p),
			Carbs100: round2(c), Fat100: round2(f), Source: *source, SourceRef: get(rec, iID)}
		if *dry {
			added++
			continue
		}
		var existing models.Ingredient
		err := db.Where("source = ? AND name = ?", *source, name).First(&existing).Error
		switch {
		case err == nil:
			ing.ID = existing.ID
			if err := db.Save(&ing).Error; err != nil {
				log.Fatal(err)
			}
			updated++
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := db.Create(&ing).Error; err != nil {
				log.Fatal(err)
			}
			added++
		default:
			log.Fatal(err)
		}
	}
	fmt.Printf("selesai: %d baru, %d diperbarui, %d dilewati (dry-run=%v)\n", added, updated, skipped, *dry)
	fmt.Println("PERIKSA SATUAN: lihat beberapa baris di bawah. Nasi matang harusnya sekitar 130-180 kkal/100 g.")
	repo := repository.New(db)
	if sample, err := repo.SearchIngredients("nasi", 3); err == nil {
		for _, s := range sample {
			fmt.Printf("  %-40s %.0f kkal  P%.1f  C%.1f  F%.1f  [%s]\n", s.Name, s.Calories100, s.Protein100, s.Carbs100, s.Fat100, s.Source)
		}
	}
}

// ---------- find ----------
func cmdFind(db *gorm.DB, args []string) {
	fs := flag.NewFlagSet("find", flag.ExitOnError)
	q := fs.String("q", "", "kata kunci nama bahan")
	limit := fs.Int("limit", 20, "jumlah hasil")
	_ = fs.Parse(args)
	rows, err := repository.New(db).SearchIngredients(*q, *limit)
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range rows {
		fmt.Printf("id=%-6d %-45s %6.1f kkal  P%5.1f  C%5.1f  F%5.1f  [%s]\n", r.ID, r.Name, r.Calories100, r.Protein100, r.Carbs100, r.Fat100, r.Source)
	}
	if len(rows) == 0 {
		fmt.Println("tidak ada hasil")
	}
}

// ---------- components ----------
type compRow struct {
	line  int
	name  string
	ingID uint
	grams float64
}

func resolveIngredient(db *gorm.DB, r compRow) (*models.Ingredient, error) {
	var ing models.Ingredient
	if r.ingID > 0 {
		if err := db.First(&ing, r.ingID).Error; err != nil {
			return nil, fmt.Errorf("ingredient_id %d tidak ditemukan", r.ingID)
		}
		return &ing, nil
	}
	var found []models.Ingredient
	if err := db.Where("LOWER(name) = ?", strings.ToLower(r.name)).Limit(5).Find(&found).Error; err != nil {
		return nil, err
	}
	switch len(found) {
	case 0:
		return nil, fmt.Errorf("bahan %q tidak ditemukan (cari nama persis dengan: importer find -q \"%s\")", r.name, r.name)
	case 1:
		return &found[0], nil
	}
	ids := make([]string, 0, len(found))
	for _, f := range found {
		ids = append(ids, strconv.Itoa(int(f.ID)))
	}
	return nil, fmt.Errorf("bahan %q ambigu (id: %s); pakai kolom ingredient_id", r.name, strings.Join(ids, ","))
}

func cmdComponents(db *gorm.DB, args []string) {
	fs := flag.NewFlagSet("components", flag.ExitOnError)
	file := fs.String("file", "", "path recipes.csv (wajib)")
	dry := fs.Bool("dry-run", false, "tampilkan perbandingan saja, tidak menulis ke DB")
	_ = fs.Parse(args)
	if *file == "" {
		log.Fatal("-file wajib diisi")
	}
	hdr, rows, err := readCSV(*file)
	if err != nil {
		log.Fatalf("baca CSV: %v", err)
	}
	iFood := colIdx(hdr, "food_name", "food", "menu", "nama_menu")
	iIng := colIdx(hdr, "ingredient", "ingredient_name", "bahan")
	iIngID := colIdx(hdr, "ingredient_id")
	iG := colIdx(hdr, "grams", "gram", "berat_g")
	if iFood < 0 || iG < 0 || (iIng < 0 && iIngID < 0) {
		log.Fatalf("kolom wajib: food_name, grams, ingredient atau ingredient_id. Header ditemukan: %v", headerKeys(hdr))
	}

	groups := map[string][]compRow{}
	hasErr := map[string]bool{}
	var order []string
	for n, rec := range rows {
		food := get(rec, iFood)
		if food == "" {
			continue
		}
		if _, ok := groups[food]; !ok {
			order = append(order, food)
		}
		g, err := num(get(rec, iG))
		if err != nil || g <= 0 {
			fmt.Printf("✗ baris %d (%s): grams tidak valid\n", n+2, food)
			hasErr[food] = true
			continue
		}
		row := compRow{line: n + 2, name: get(rec, iIng), grams: g}
		if iIngID >= 0 {
			if v, err := strconv.Atoi(get(rec, iIngID)); err == nil && v > 0 {
				row.ingID = uint(v)
			}
		}
		if row.name == "" && row.ingID == 0 {
			fmt.Printf("✗ baris %d (%s): bahan kosong\n", n+2, food)
			hasErr[food] = true
			continue
		}
		groups[food] = append(groups[food], row)
	}

	applied, skipped := 0, 0
	for _, foodName := range order {
		var food models.FoodItem
		if err := db.Where("LOWER(name) = ?", strings.ToLower(foodName)).First(&food).Error; err != nil {
			fmt.Printf("✗ %s: menu tidak ada di tabel foods\n", foodName)
			skipped++
			continue
		}
		ok := !hasErr[foodName]
		var cal, p, c, f float64
		merged := map[uint]float64{}
		sources := map[string]bool{}
		for _, r := range groups[foodName] {
			ing, err := resolveIngredient(db, r)
			if err != nil {
				fmt.Printf("✗ %s (baris %d): %v\n", foodName, r.line, err)
				ok = false
				continue
			}
			merged[ing.ID] += r.grams
			k := r.grams / 100
			cal += ing.Calories100 * k
			p += ing.Protein100 * k
			c += ing.Carbs100 * k
			f += ing.Fat100 * k
			sources[ing.Source] = true
		}
		if !ok {
			fmt.Printf("✗ %s dilewati (ada baris bermasalah)\n", foodName)
			skipped++
			continue
		}
		diff := math.Abs(float64(food.Calories)-cal) / math.Max(float64(food.Calories), 1) * 100
		flag := ""
		if diff > 25 {
			flag = "  <-- REVIEW (selisih >25%)"
		}
		fmt.Printf("%-32s lama %4d kkal P%5.1f C%5.1f F%5.1f | baru %4.0f kkal P%5.1f C%5.1f F%5.1f%s\n",
			food.Name, food.Calories, food.Protein, food.Carbs, food.Fat, cal, p, c, f, flag)
		if *dry {
			continue
		}
		srcs := make([]string, 0, len(sources))
		for s := range sources {
			srcs = append(srcs, s)
		}
		sort.Strings(srcs)
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("food_id = ?", food.ID).Delete(&models.FoodComponent{}).Error; err != nil {
				return err
			}
			for id, g := range merged {
				if err := tx.Create(&models.FoodComponent{FoodID: food.ID, IngredientID: id, Grams: round1(g)}).Error; err != nil {
					return err
				}
			}
			return tx.Model(&models.FoodItem{}).Where("id = ?", food.ID).Updates(map[string]any{
				"calories": int(math.Round(cal)), "protein": round1(p), "carbs": round1(c), "fat": round1(f),
				"source": "computed:" + strings.Join(srcs, "+"),
			}).Error
		})
		if err != nil {
			log.Fatalf("%s: %v", food.Name, err)
		}
		applied++
	}
	fmt.Printf("\nselesai: %d menu diperbarui, %d dilewati (dry-run=%v)\n", applied, skipped, *dry)
}
