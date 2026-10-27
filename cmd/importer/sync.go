package main

// Sinkronisasi dua arah antara database dan CSV di docs/ agar keduanya selalu identik.
//
//	importer export                    DB  -> CSV
//	importer sync-review               CSV -> DB (nilai bahan + verified), lalu hitung ulang menu terkait
//	importer sync-recipes              CSV -> DB (gramasi per menu), menu ditandai source=custom

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"nutrai/internal/database"
	"nutrai/internal/models"
)

func ff(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func writeCSV(path string, rows [][]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.WriteAll(rows); err != nil {
		return err
	}
	return w.Error()
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "ya", "y":
		return true
	}
	return false
}

// ---------- export ----------
func cmdExport(db *gorm.DB, args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	dir := fs.String("dir", "docs", "folder output")
	_ = fs.Parse(args)

	var ings []models.Ingredient
	if err := db.Where("code IS NOT NULL").Order("id").Find(&ings).Error; err != nil {
		log.Fatal(err)
	}
	rows := [][]string{{"key", "name", "state", "kcal_100g", "protein_100g", "carbs_100g", "fat_100g", "source", "verified", "note"}}
	for _, i := range ings {
		ver := "FALSE"
		if i.Verified {
			ver = "TRUE"
		}
		rows = append(rows, []string{*i.Code, i.Name, i.State, ff(i.Calories100), ff(i.Protein100), ff(i.Carbs100), ff(i.Fat100), i.Source, ver, i.Note})
	}
	if err := writeCSV(filepath.Join(*dir, "ingredients-review.csv"), rows); err != nil {
		log.Fatal(err)
	}

	type recipeRow struct {
		Menu          string  `gorm:"column:menu"`
		Category      string  `gorm:"column:category"`
		Serving       string  `gorm:"column:serving"`
		ServingG      int     `gorm:"column:serving_g"`
		IngredientKey string  `gorm:"column:ingredient_key"`
		Ingredient    string  `gorm:"column:ingredient"`
		Grams         float64 `gorm:"column:grams"`
	}
	var rr []recipeRow
	err := db.Table("food_components AS fc").
		Select("f.name AS menu, f.category AS category, COALESCE(f.serving,'') AS serving, COALESCE(f.serving_g,0) AS serving_g, COALESCE(i.code,'') AS ingredient_key, i.name AS ingredient, fc.grams AS grams").
		Joins("JOIN foods AS f ON f.id = fc.food_id").
		Joins("JOIN ingredients AS i ON i.id = fc.ingredient_id").
		Order("f.id ASC, fc.grams DESC").Scan(&rr).Error
	if err != nil {
		log.Fatal(err)
	}
	rrows := [][]string{{"menu", "category", "serving", "serving_g", "ingredient_key", "ingredient", "grams"}}
	for _, r := range rr {
		rrows = append(rrows, []string{r.Menu, r.Category, r.Serving, strconv.Itoa(r.ServingG), r.IngredientKey, r.Ingredient, ff(r.Grams)})
	}
	if err := writeCSV(filepath.Join(*dir, "food-recipes.csv"), rrows); err != nil {
		log.Fatal(err)
	}

	var foods []models.FoodItem
	if err := db.Order("id").Find(&foods).Error; err != nil {
		log.Fatal(err)
	}
	frows := [][]string{{"id", "name", "category", "serving", "serving_g", "calories", "protein_g", "carbs_g", "fat_g", "source", "tags"}}
	for _, f := range foods {
		frows = append(frows, []string{strconv.Itoa(int(f.ID)), f.Name, f.Category, f.Serving, strconv.Itoa(f.ServingG),
			strconv.Itoa(f.Calories), ff(f.Protein), ff(f.Carbs), ff(f.Fat), f.Source, f.TagsJSON})
	}
	if err := writeCSV(filepath.Join(*dir, "foods.csv"), frows); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("export OK -> %s: %d bahan, %d baris resep, %d menu\n", *dir, len(ings), len(rr), len(foods))
}

// ---------- sync-review (CSV bahan -> DB) ----------
func cmdSyncReview(db *gorm.DB, args []string) {
	fs := flag.NewFlagSet("sync-review", flag.ExitOnError)
	file := fs.String("file", "docs/ingredients-review.csv", "CSV hasil review")
	dry := fs.Bool("dry-run", false, "hanya tampilkan perubahan")
	_ = fs.Parse(args)

	hdr, rows, err := readCSV(*file)
	if err != nil {
		log.Fatalf("baca %s: %v", *file, err)
	}
	iKey := colIdx(hdr, "key")
	iName, iState := colIdx(hdr, "name"), colIdx(hdr, "state")
	iK, iP := colIdx(hdr, "kcal_100g"), colIdx(hdr, "protein_100g")
	iC, iF := colIdx(hdr, "carbs_100g"), colIdx(hdr, "fat_100g")
	iSrc, iVer, iNote := colIdx(hdr, "source"), colIdx(hdr, "verified"), colIdx(hdr, "note")
	if iKey < 0 || iK < 0 || iP < 0 || iC < 0 || iF < 0 {
		log.Fatalf("header wajib: key,kcal_100g,protein_100g,carbs_100g,fat_100g (ditemukan: %v)", headerKeys(hdr))
	}

	valueChanged := []uint{}
	changed, skipped, invalid := 0, 0, 0
	for _, rec := range rows {
		key := get(rec, iKey)
		if key == "" {
			continue
		}
		var ing models.Ingredient
		if err := db.Where("code = ?", key).First(&ing).Error; err != nil {
			fmt.Printf("SKIP    %-16s tidak ada di DB (CSV tidak membuat bahan baru)\n", key)
			skipped++
			continue
		}
		k, e1 := num(get(rec, iK))
		p, e2 := num(get(rec, iP))
		c, e3 := num(get(rec, iC))
		f, e4 := num(get(rec, iF))
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil ||
			k < 0 || k > 950 || p < 0 || p > 100 || c < 0 || c > 100 || f < 0 || f > 100 || p+c+f > 100.5 {
			fmt.Printf("INVALID %-16s angka tidak wajar (kkal 0-950, makro 0-100 g, jumlah makro <=100 g per 100 g)\n", key)
			invalid++
			continue
		}
		name, state, note := get(rec, iName), get(rec, iState), get(rec, iNote)
		source := get(rec, iSrc)
		if source == "" {
			source = ing.Source
		}
		if name == "" {
			name = ing.Name
		}
		verified := parseBool(get(rec, iVer))

		const eps = 0.001
		vals := math.Abs(k-ing.Calories100) > eps || math.Abs(p-ing.Protein100) > eps ||
			math.Abs(c-ing.Carbs100) > eps || math.Abs(f-ing.Fat100) > eps
		meta := verified != ing.Verified || source != ing.Source || name != ing.Name || state != ing.State || note != ing.Note
		if !vals && !meta {
			continue
		}
		changed++
		fmt.Printf("UBAH    %-16s %6.1f/%4.1f/%4.1f/%4.1f -> %6.1f/%4.1f/%4.1f/%4.1f  verified %v -> %v\n", key,
			ing.Calories100, ing.Protein100, ing.Carbs100, ing.Fat100, k, p, c, f, ing.Verified, verified)
		if vals {
			valueChanged = append(valueChanged, ing.ID)
		}
		if *dry {
			continue
		}
		err := db.Model(&ing).Updates(map[string]any{"name": name, "state": state, "calories_100g": k, "protein_100g": p,
			"carbs_100g": c, "fat_100g": f, "source": source, "verified": verified, "note": note}).Error
		if err != nil {
			log.Fatalf("update %s: %v", key, err)
		}
	}

	menus := 0
	if len(valueChanged) > 0 {
		var foodIDs []uint
		if err := db.Table("food_components").Where("ingredient_id IN ?", valueChanged).Distinct().Pluck("food_id", &foodIDs).Error; err != nil {
			log.Fatal(err)
		}
		menus = len(foodIDs)
		if !*dry {
			for _, id := range foodIDs {
				if err := database.RecomputeFood(db, id); err != nil {
					log.Fatalf("hitung ulang menu %d: %v", id, err)
				}
			}
		}
	}
	mode := ""
	if *dry {
		mode = " (DRY-RUN, tidak ada yang ditulis)"
	}
	fmt.Printf("\nselesai%s: %d bahan berubah, %d dilewati, %d tidak valid, %d menu dihitung ulang\n", mode, changed, skipped, invalid, menus)
}

// ---------- sync-recipes (CSV gramasi -> DB) ----------
func cmdSyncRecipes(db *gorm.DB, args []string) {
	fs := flag.NewFlagSet("sync-recipes", flag.ExitOnError)
	file := fs.String("file", "docs/food-recipes.csv", "CSV resep")
	dry := fs.Bool("dry-run", false, "hanya tampilkan perubahan")
	_ = fs.Parse(args)

	hdr, rows, err := readCSV(*file)
	if err != nil {
		log.Fatalf("baca %s: %v", *file, err)
	}
	iMenu, iKey, iGram := colIdx(hdr, "menu"), colIdx(hdr, "ingredient_key"), colIdx(hdr, "grams")
	if iMenu < 0 || iKey < 0 || iGram < 0 {
		log.Fatalf("header wajib: menu,ingredient_key,grams (ditemukan: %v)", headerKeys(hdr))
	}

	type line struct {
		key   string
		grams float64
	}
	order := []string{}
	byMenu := map[string][]line{}
	bad := map[string]string{}
	for _, rec := range rows {
		m := get(rec, iMenu)
		if m == "" {
			continue
		}
		if _, ok := byMenu[m]; !ok {
			order = append(order, m)
		}
		g, err := num(get(rec, iGram))
		key := get(rec, iKey)
		if err != nil || g <= 0 || g > 2000 || key == "" {
			bad[m] = fmt.Sprintf("baris tidak valid (ingredient_key=%q grams=%q)", key, get(rec, iGram))
		}
		byMenu[m] = append(byMenu[m], line{key, g})
	}

	changed, same, skipped := 0, 0, 0
	for _, m := range order {
		if why, ok := bad[m]; ok {
			fmt.Printf("SKIP    %-34s %s\n", m, why)
			skipped++
			continue
		}
		var food models.FoodItem
		if err := db.Where("name = ?", m).First(&food).Error; err != nil {
			fmt.Printf("SKIP    %-34s menu tidak ada di DB\n", m)
			skipped++
			continue
		}
		if food.Source != "recipe" && food.Source != "estimate" && food.Source != "custom" {
			fmt.Printf("SKIP    %-34s source=%s (dikelola jalur lain, tidak diubah)\n", m, food.Source)
			skipped++
			continue
		}
		// resolve bahan -> id; satu bahan hanya boleh sekali per menu
		want := map[uint]float64{}
		var ids []uint
		ok := true
		for _, l := range byMenu[m] {
			var ing models.Ingredient
			if err := db.Where("code = ?", l.key).First(&ing).Error; err != nil {
				fmt.Printf("SKIP    %-34s bahan %q tidak ada (menu dilewati seluruhnya)\n", m, l.key)
				ok = false
				break
			}
			if _, dup := want[ing.ID]; dup {
				fmt.Printf("SKIP    %-34s bahan %q muncul dua kali\n", m, l.key)
				ok = false
				break
			}
			want[ing.ID] = l.grams
			ids = append(ids, ing.ID)
		}
		if !ok {
			skipped++
			continue
		}
		var cur []models.FoodComponent
		if err := db.Where("food_id = ?", food.ID).Find(&cur).Error; err != nil {
			log.Fatal(err)
		}
		same2 := len(cur) == len(want)
		for _, c := range cur {
			if g, ok := want[c.IngredientID]; !ok || math.Abs(g-c.Grams) > 0.05 {
				same2 = false
			}
		}
		if same2 {
			same++
			continue
		}
		changed++
		if *dry {
			fmt.Printf("UBAH    %-34s %d bahan -> %d bahan (dry-run)\n", m, len(cur), len(want))
			continue
		}
		oldKcal := food.Calories
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("food_id = ?", food.ID).Delete(&models.FoodComponent{}).Error; err != nil {
				return err
			}
			for _, id := range ids {
				if err := tx.Create(&models.FoodComponent{FoodID: food.ID, IngredientID: id, Grams: want[id]}).Error; err != nil {
					return err
				}
			}
			// source=custom: seed otomatis tidak akan menimpa resep yang diedit lewat CSV
			return tx.Model(&models.FoodItem{}).Where("id = ?", food.ID).Update("source", "custom").Error
		})
		if err != nil {
			log.Fatalf("simpan %s: %v", m, err)
		}
		if err := database.RecomputeFood(db, food.ID); err != nil {
			log.Fatalf("hitung ulang %s: %v", m, err)
		}
		var after models.FoodItem
		db.First(&after, food.ID)
		fmt.Printf("UBAH    %-34s %4d -> %4d kkal (source=custom)\n", m, oldKcal, after.Calories)
	}
	mode := ""
	if *dry {
		mode = " (DRY-RUN, tidak ada yang ditulis)"
	}
	fmt.Printf("\nselesai%s: %d menu berubah, %d sama, %d dilewati\n", mode, changed, same, skipped)
}
