#!/usr/bin/env python3
"""Generator dataset makanan NutrAI.

Kalori & makro menu DIHITUNG dari bahan (per 100 g) x gramasi, bukan ditebak per porsi.
Jalankan dari root repo:  python3 tools/gen_foods.py
Output: internal/seed/{ingredients,foods}.json, docs/{ingredients-review,food-recipes,food-images-manifest}.csv
Perbaiki angka di tabel ING (satu baris per bahan) lalu jalankan ulang -> semua menu ikut ter-update.
"""
import csv, json, re, sys

# ---- meta menu (kategori, porsi, budget, indonesian) ----
# (name, category, serving_label, budget, indonesian)
B, M, P = "budget_friendly", "moderate", "premium"
MENUS = {}
def menu(name, cat, serving, budget, indo=False): MENUS[name] = (cat, serving, budget, indo)

# ---- bahan: key -> (nama, state, kcal, P, C, F, allergens, animal, tags, source, catatan) per 100 g ----
# source: usda = nilai USDA FoodData Central (SR Legacy); derived = turunan/asumsi saya; product = bergantung merek
ING = {}
def ing(key, name, state, kcal, p, c, f, allergens=(), animal=None, tags=(), source="usda", note=""):
    ING[key] = dict(name=name, state=state, kcal=kcal, p=p, c=c, f=f, allergens=list(allergens),
                    animal=animal, tags=list(tags), source=source, note=note)

# karbohidrat & tepung
ing("nasi","Nasi putih","cooked",130,2.7,28.2,0.3,tags=["rice"])
ing("nasi_merah","Nasi merah","cooked",123,2.7,25.6,1.0,tags=["rice"])
ing("lontong","Lontong","cooked",105,2.2,23,0.2,tags=["rice"],source="derived",note="turunan dari nasi putih (kadar air lebih tinggi); tidak ada di USDA")
ing("mie_telur","Mie telur","cooked",138,4.5,25.2,2.1,["wheat","egg"],"egg")
ing("soba","Soba","cooked",99,5.1,21.4,0.1,["wheat"])
ing("spaghetti","Spaghetti","cooked",158,5.8,30.9,0.9,["wheat"])
ing("bihun","Bihun / rice vermicelli","cooked",108,0.9,24.9,0.2)
ing("roti_tawar","Roti tawar putih","as_eaten",265,9.0,49,3.2,["wheat"])
ing("roti_gandum","Roti gandum","as_eaten",252,12.5,43,3.5,["wheat"])
ing("oat","Oat gulung (kering)","dry",379,13.2,67.7,6.5,note="gluten oat bisa terkontaminasi; tidak diberi tag wheat")
ing("quinoa","Quinoa","cooked",120,4.4,21.3,1.9)
ing("kentang","Kentang rebus","cooked",87,1.9,20.1,0.1)
ing("ubi","Ubi jalar rebus","cooked",76,1.4,17.7,0.1)
ing("singkong","Singkong rebus","cooked",160,1.4,38,0.3,source="derived",note="memakai nilai singkong mentah USDA sebagai pendekatan")
ing("jagung","Jagung manis rebus (pipilan)","cooked",96,3.4,21,1.5)
ing("terigu","Tepung terigu","dry",364,10.3,76.3,1.0,["wheat"])
ing("tepung_beras","Tepung beras","dry",366,5.9,80,1.4)
ing("tortilla","Tortilla gandum","as_eaten",312,8.0,51.7,8.0,["wheat"])
ing("crackers","Crackers asin","as_eaten",421,9.0,74,10,["wheat"])
ing("granola","Granola","as_eaten",471,10,64,20,["wheat"],source="product",note="sangat bergantung merek")
# protein hewani
ing("ayam_dada","Dada ayam tanpa kulit","cooked",165,31.0,0,3.6,animal="meat",tags=["chicken"])
ing("ayam_paha","Paha ayam tanpa kulit","cooked",179,24.8,0,8.2,animal="meat",tags=["chicken"])
ing("sapi","Daging sapi (campur lemak)","cooked",215,30,0,10,animal="meat",tags=["beef"],source="derived",note="pendekatan potongan has luar/round dimasak; bervariasi menurut potongan")
ing("sapi_giling","Daging sapi giling","cooked",250,25.9,0,15.4,animal="meat",tags=["beef"])
ing("babi","Daging babi","cooked",220,25,0,13,animal="pork",tags=["pork"],source="derived",note="pendekatan bahu/loin dimasak")
ing("telur","Telur ayam utuh","cooked",155,12.6,1.1,10.6,["egg"],"egg")
ing("salmon","Salmon","cooked",206,22.1,0,12.4,["fish"],"fish")
ing("tuna","Tuna kaleng (air, ditiriskan)","as_eaten",116,25.5,0,0.8,["fish"],"fish")
ing("nila","Ikan nila / ikan putih","cooked",128,26.2,0,2.7,["fish"],"fish",note="dipakai juga untuk gurame/ikan putih lain")
ing("lele","Ikan lele","cooked",152,18.5,0,8.2,["fish"],"fish")
ing("udang","Udang","cooked",99,24.0,0.2,0.3,["shellfish"],"shellfish")
ing("cumi","Cumi-cumi","raw",92,15.6,3.1,1.4,["shellfish"],"shellfish",note="berat mentah")
# protein nabati
ing("tempe","Tempe","as_eaten",192,20.3,7.6,10.8,["soy"])
ing("tahu","Tahu","raw",76,8.1,1.9,4.8,["soy"])
ing("edamame","Edamame","cooked",121,11.9,8.9,5.2,["soy"])
ing("lentil","Lentil","cooked",116,9.0,20.1,0.4)
ing("kacang_hijau","Kacang hijau","cooked",105,7.0,19.2,0.4)
# susu & olahan
ing("keju","Keju cheddar","as_eaten",403,24.9,1.3,33.1,["dairy"],"dairy")
ing("cottage","Cottage cheese","as_eaten",98,11.1,3.4,4.3,["dairy"],"dairy")
ing("yogurt_greek","Greek yogurt 2%","as_eaten",73,9.95,3.94,1.92,["dairy"],"dairy")
ing("susu","Susu sapi full cream","as_eaten",61,3.2,4.8,3.3,["dairy"],"dairy")
ing("susu_kedelai","Susu kedelai tanpa gula","as_eaten",54,3.3,6.3,1.8,["soy"])
ing("whey","Whey protein (bubuk)","as_eaten",400,80,8,6,["dairy"],"dairy",source="product",note="bergantung merek; cek label")
ing("mentega","Mentega","as_eaten",717,0.9,0.1,81.1,["dairy"],"dairy")
ing("krim","Krim kental (heavy cream)","as_eaten",340,2.8,2.8,36,["dairy"],"dairy")
# lemak, bumbu, saus
ing("minyak","Minyak goreng","as_eaten",884,0,0,100)
ing("santan","Santan","as_eaten",197,2.0,2.8,21.3,note="santan kental kaleng; santan segar bervariasi")
ing("selai_kacang","Selai kacang","as_eaten",588,25.1,20,50.4,["peanut"],note="juga dipakai sebagai pendekatan bumbu kacang")
ing("kacang_rebus","Kacang tanah rebus","cooked",318,13.5,21.3,21.8,["peanut"],source="derived",note="nilai dari ingatan, perlu cek")
ing("almond","Almond","as_eaten",579,21.2,21.6,49.9,["tree nuts"])
ing("gula","Gula / gula merah","as_eaten",387,0,100,0)
ing("cokelat_hitam","Cokelat hitam 70-85%","as_eaten",598,7.8,45.9,42.6,["dairy"],note="sering mengandung jejak susu")
ing("selai_cokelat","Selai cokelat hazelnut","as_eaten",539,6.3,57.5,30.9,["tree nuts","dairy"],"dairy",source="product",note="bergantung merek")
ing("mayo","Mayones","as_eaten",680,1.0,0.6,75,["egg"],"egg")
ing("dressing_caesar","Dressing Caesar","as_eaten",460,2.2,3.8,49,["egg","dairy","fish"],"dairy",source="product",note="mengandung anchovy; bergantung merek")
ing("kecap_manis","Kecap manis","as_eaten",245,2.0,57,0.2,["soy","wheat"],source="derived",note="tidak ada di USDA; pendekatan (sebagian besar gula)")
ing("saus_tiram","Saus tiram","as_eaten",51,1.4,11,0.3,["shellfish","soy","wheat"])
ing("teriyaki","Saus teriyaki","as_eaten",89,5.9,15.6,0,["soy","wheat"])
ing("saus_tomat","Saus tomat","as_eaten",29,1.3,6.6,0.2)
ing("kaldu","Kaldu","as_eaten",6,0.6,0.4,0.2)
# sayur & buah
ing("sayur","Sayur campur matang","cooked",35,2.0,6.5,0.3,source="derived",note="rata-rata sayur hijau/wortel/kol rebus")
ing("jamur","Jamur champignon rebus","cooked",28,2.2,5.3,0.1)
ing("nangka_muda","Nangka","raw",95,1.7,23.3,0.6,note="nilai nangka masak; nangka muda lebih rendah kalori")
ing("pisang","Pisang","raw",89,1.1,22.8,0.3)
ing("apel","Apel","raw",52,0.3,13.8,0.2)
ing("alpukat","Alpukat","raw",160,2.0,8.5,14.7)
ing("pepaya","Pepaya","raw",43,0.5,10.8,0.3)
ing("semangka","Semangka","raw",30,0.6,7.6,0.2)
ing("beri","Beri campur (stroberi/blueberry)","raw",45,0.7,11,0.3)

# ---- resep (gram per porsi) ----
R = {}
def rec(name, cat, serving, budget, indo, parts): menu(name, cat, serving, budget, indo); R[name] = parts
# Breakfast
rec("Nasi uduk telur","Breakfast","1 piring",B,True,[("nasi",160),("santan",25),("telur",55),("tempe",30),("bihun",30),("minyak",10)])
rec("Bubur ayam","Breakfast","1 mangkuk",B,True,[("nasi",100),("kaldu",150),("ayam_dada",50),("kecap_manis",5),("minyak",3)])
rec("Lontong sayur","Breakfast","1 porsi",B,True,[("lontong",150),("santan",60),("sayur",60),("tahu",40),("minyak",5)])
rec("Telur dadar & roti gandum","Breakfast","2 telur + 2 slice",B,False,[("telur",110),("minyak",6),("roti_gandum",60)])
rec("Greek yogurt parfait","Breakfast","1 cup",M,False,[("yogurt_greek",170),("granola",40),("beri",60)])
rec("Oatmeal pisang","Breakfast","1 mangkuk",B,False,[("oat",50),("pisang",100),("susu_kedelai",120)])
rec("Banana peanut toast","Breakfast","2 slice",B,False,[("roti_tawar",60),("selai_kacang",20),("pisang",80)])
rec("Scramble tahu telur","Breakfast","1 porsi",B,False,[("telur",110),("tahu",100),("minyak",8),("sayur",30)])
rec("Smoothie bowl pisang beri","Breakfast","1 mangkuk",M,False,[("pisang",120),("beri",80),("susu_kedelai",120),("oat",25)])
rec("Telur rebus & ubi","Breakfast","2 telur + 150 g ubi",B,False,[("telur",110),("ubi",150)])
rec("Nasi kuning & telur suwir","Breakfast","1 piring",B,True,[("nasi",150),("santan",30),("telur",55),("minyak",8),("bihun",20)])
rec("Bubur kacang hijau","Breakfast","1 mangkuk",B,True,[("kacang_hijau",120),("santan",60),("gula",20)])
rec("Roti bakar cokelat keju","Breakfast","2 slice",B,False,[("roti_tawar",60),("selai_cokelat",15),("keju",15),("mentega",5)])
rec("Pancake pisang oat","Breakfast","3 buah",M,False,[("oat",40),("pisang",100),("telur",55),("susu",60),("minyak",5)])
rec("Cheese omelette","Breakfast","2 telur",M,False,[("telur",110),("keju",25),("mentega",8)])
rec("Overnight oats buah","Breakfast","1 jar",M,False,[("oat",45),("susu",130),("yogurt_greek",60),("beri",70)])
rec("Dadar jagung","Breakfast","2 potong",B,True,[("telur",55),("jagung",80),("terigu",25),("minyak",14)])
rec("Bubur sumsum","Breakfast","1 mangkuk",B,True,[("tepung_beras",35),("santan",50),("gula",20)])
# Lunch
rec("Chicken rice bowl","Lunch","1 mangkuk",B,False,[("nasi",180),("ayam_dada",120),("sayur",60),("minyak",6)])
rec("Nasi pecel tempe","Lunch","1 piring",B,True,[("nasi",180),("tempe",50),("minyak",8),("selai_kacang",20),("gula",5),("sayur",100)])
rec("Gado-gado","Lunch","1 piring",B,True,[("sayur",150),("tahu",50),("tempe",30),("telur",55),("lontong",80),("selai_kacang",30),("gula",8),("minyak",8)])
rec("Soto ayam & nasi","Lunch","1 mangkuk + nasi",B,True,[("nasi",160),("ayam_paha",70),("kaldu",250),("bihun",30),("minyak",5),("telur",55)])
rec("Nasi ayam bakar & lalapan","Lunch","1 piring",B,True,[("nasi",180),("ayam_paha",120),("sayur",50),("kecap_manis",10),("minyak",5)])
rec("Nasi tempe orek & sayur","Lunch","1 piring",B,True,[("nasi",180),("tempe",70),("kecap_manis",10),("gula",4),("minyak",10),("sayur",80)])
rec("Pepes ikan & nasi","Lunch","1 piring",M,True,[("nasi",180),("nila",120),("sayur",30),("minyak",3)])
rec("Mie ayam","Lunch","1 mangkuk",B,True,[("mie_telur",200),("ayam_paha",60),("minyak",10),("kecap_manis",10),("sayur",40),("kaldu",100)])
rec("Nasi rawon","Lunch","1 piring",M,True,[("nasi",170),("sapi",80),("kaldu",250),("sayur",40),("minyak",10)])
rec("Tuna sandwich gandum","Lunch","1 sandwich",M,False,[("roti_gandum",70),("tuna",80),("mayo",15),("sayur",30)])
rec("Nasi goreng sayur telur","Lunch","1 piring",B,True,[("nasi",200),("telur",55),("minyak",12),("sayur",50),("kecap_manis",12)])
rec("Char siu rice","Lunch","1 piring",M,False,[("nasi",180),("babi",100),("kecap_manis",15),("gula",10),("sayur",50),("minyak",5)])
rec("Nasi padang ayam pop","Lunch","1 piring",M,True,[("nasi",220),("ayam_paha",130),("minyak",12),("santan",50),("sayur",100)])
rec("Nasi padang rendang","Lunch","1 piring",M,True,[("nasi",220),("sapi",90),("santan",50),("minyak",10),("sayur",100)])
rec("Soto betawi","Lunch","1 mangkuk",M,True,[("sapi",80),("santan",120),("kaldu",250),("kentang",60),("minyak",5)])
rec("Bakso & mie","Lunch","1 mangkuk",B,True,[("sapi_giling",80),("tepung_beras",40),("mie_telur",120),("kaldu",250),("sayur",40),("minyak",3)])
rec("Nasi liwet ayam","Lunch","1 piring",B,True,[("nasi",220),("santan",40),("ayam_paha",100),("tempe",30),("minyak",10),("sayur",30)])
rec("Gudeg nasi telur","Lunch","1 piring",B,True,[("nasi",170),("nangka_muda",120),("santan",50),("gula",15),("telur",55),("minyak",8)])
rec("Karedok","Lunch","1 porsi",B,True,[("sayur",200),("selai_kacang",30),("gula",8),("tempe",30)])
rec("Ketoprak","Lunch","1 piring",B,True,[("lontong",150),("bihun",50),("tahu",70),("minyak",10),("selai_kacang",30),("gula",8),("sayur",40)])
rec("Mie goreng telur","Lunch","1 piring",B,True,[("mie_telur",250),("telur",55),("minyak",14),("kecap_manis",15),("sayur",50)])
rec("Chicken caesar wrap","Lunch","1 wrap",M,False,[("tortilla",70),("ayam_dada",100),("sayur",50),("dressing_caesar",25),("keju",10)])
rec("Tuna poke bowl","Lunch","1 mangkuk",P,False,[("nasi",180),("tuna",100),("alpukat",50),("sayur",50),("teriyaki",15),("minyak",5)])
rec("Kari sayur & nasi","Lunch","1 piring",B,False,[("nasi",200),("sayur",150),("santan",70),("kentang",80),("minyak",10)])
rec("Nasi ayam geprek","Lunch","1 piring",B,True,[("nasi",200),("ayam_paha",120),("terigu",25),("minyak",18),("sayur",30)])
# Dinner
rec("Tofu soba stir fry","Dinner","1 porsi",M,False,[("soba",180),("tahu",120),("sayur",100),("minyak",8),("teriyaki",15)])
rec("Salmon quinoa bowl","Dinner","1 mangkuk",P,False,[("salmon",130),("quinoa",150),("sayur",60),("minyak",5)])
rec("Chicken vegetable soup","Dinner","1 mangkuk",B,False,[("ayam_dada",100),("sayur",120),("kentang",60),("kaldu",250)])
rec("Chicken noodle soup","Dinner","1 mangkuk",B,False,[("ayam_dada",80),("mie_telur",150),("sayur",40),("kaldu",250),("minyak",3)])
rec("Grilled chicken potato plate","Dinner","1 piring",M,False,[("ayam_dada",150),("kentang",200),("sayur",60),("minyak",8)])
rec("Tempeh rice bowl","Dinner","1 mangkuk",B,False,[("nasi",180),("tempe",100),("sayur",80),("kecap_manis",8),("minyak",8)])
rec("Pecel lele & nasi","Dinner","1 piring",B,True,[("nasi",180),("lele",120),("minyak",15),("selai_kacang",20),("gula",3),("sayur",50)])
rec("Ikan bakar & sayur asem","Dinner","1 ikan + nasi",M,True,[("nasi",150),("nila",150),("sayur",120),("kecap_manis",10),("minyak",4)])
rec("Capcay ayam","Dinner","1 porsi",B,False,[("ayam_dada",90),("sayur",200),("minyak",10),("saus_tiram",15)])
rec("Sayur lodeh & tempe","Dinner","1 mangkuk + nasi",B,True,[("nasi",150),("sayur",150),("santan",80),("tempe",50),("minyak",6)])
rec("Sate ayam & lontong","Dinner","10 tusuk",M,True,[("ayam_paha",130),("selai_kacang",35),("kecap_manis",15),("lontong",150),("minyak",5),("gula",5)])
rec("Udang saus tiram & nasi","Dinner","1 piring",M,False,[("nasi",180),("udang",130),("saus_tiram",20),("minyak",8),("sayur",60)])
rec("Beef stir fry & brown rice","Dinner","1 piring",P,False,[("nasi_merah",180),("sapi",110),("saus_tiram",15),("minyak",10),("sayur",100)])
rec("Bak kut teh & nasi","Dinner","1 mangkuk",M,False,[("nasi",170),("babi",100),("kaldu",300),("kecap_manis",10),("minyak",3)])
rec("Pasta ayam jamur","Dinner","1 piring",M,False,[("spaghetti",220),("ayam_dada",90),("jamur",60),("krim",40),("minyak",5)])
rec("Spaghetti bolognese","Dinner","1 piring",M,False,[("spaghetti",220),("sapi_giling",80),("saus_tomat",100),("minyak",5)])
rec("Steak ayam & sayur panggang","Dinner","1 piring",M,False,[("ayam_dada",170),("sayur",120),("minyak",10)])
rec("Gurame asam manis & nasi","Dinner","1 piring",M,True,[("nasi",180),("nila",140),("minyak",12),("saus_tomat",40),("gula",12),("sayur",30)])
rec("Tumis kangkung tahu & nasi","Dinner","1 piring",B,True,[("nasi",180),("tahu",120),("sayur",100),("minyak",8),("saus_tiram",10)])
rec("Semur daging & nasi","Dinner","1 piring",M,True,[("nasi",180),("sapi",90),("kecap_manis",20),("kentang",50),("minyak",6)])
rec("Opor ayam & lontong","Dinner","1 porsi",B,True,[("lontong",150),("ayam_paha",110),("santan",100),("minyak",5)])
rec("Ikan nila goreng & nasi","Dinner","1 piring",B,True,[("nasi",180),("nila",150),("minyak",12),("sayur",40)])
rec("Telur balado & nasi","Dinner","1 piring",B,True,[("nasi",180),("telur",110),("minyak",12),("saus_tomat",20),("sayur",20)])
rec("Tahu tempe penyet & nasi","Dinner","1 piring",B,True,[("nasi",180),("tahu",80),("tempe",60),("minyak",14),("sayur",40)])
rec("Sup ayam jamur","Dinner","1 mangkuk",B,False,[("ayam_dada",90),("jamur",60),("sayur",80),("kaldu",250)])
rec("Cumi saus padang & nasi","Dinner","1 piring",M,True,[("nasi",180),("cumi",150),("saus_tomat",60),("minyak",10),("gula",5)])
rec("Chicken teriyaki & rice","Dinner","1 piring",M,False,[("nasi",180),("ayam_paha",120),("teriyaki",40),("sayur",60),("minyak",5)])
rec("Tofu veggie curry","Dinner","1 mangkuk",M,False,[("tahu",200),("sayur",120),("santan",80),("minyak",5)])
rec("Grilled salmon & sweet potato","Dinner","1 piring",P,False,[("salmon",130),("ubi",180),("sayur",60),("minyak",5)])
rec("Sup lentil","Dinner","1 mangkuk",M,False,[("lentil",200),("sayur",80),("kaldu",200),("minyak",5)])
# Snack
rec("Pisang rebus","Snack","2 buah",B,False,[("pisang",150)])
rec("Kacang tanah rebus","Snack","1 genggam",B,False,[("kacang_rebus",50)])
rec("Telur rebus","Snack","2 butir",B,False,[("telur",110)])
rec("Greek yogurt & buah","Snack","1 cup",M,False,[("yogurt_greek",150),("beri",70)])
rec("Edamame","Snack","1 mangkuk",M,False,[("edamame",150)])
rec("Tahu goreng","Snack","3 potong",B,False,[("tahu",120),("minyak",12)])
rec("Protein shake susu","Snack","1 gelas",M,False,[("susu",250),("whey",25)])
rec("Kacang almond","Snack","1 genggam",M,False,[("almond",28)])
rec("Roti gandum alpukat","Snack","1 slice",M,False,[("roti_gandum",40),("alpukat",70)])
rec("Singkong rebus","Snack","150 g",B,False,[("singkong",150)])
rec("Apel & selai kacang","Snack","1 apel + 1 sdm",B,False,[("apel",180),("selai_kacang",16)])
rec("Yogurt & granola","Snack","1 cup",M,False,[("yogurt_greek",170),("granola",40)])
rec("Pisang goreng","Snack","2 buah",B,True,[("pisang",120),("terigu",25),("minyak",14)])
rec("Tempe goreng","Snack","3 potong",B,True,[("tempe",75),("terigu",12),("minyak",10)])
rec("Alpukat","Snack","1/2 buah",M,False,[("alpukat",100)])
rec("Buah potong campur","Snack","1 mangkuk",B,False,[("pepaya",120),("semangka",100),("apel",60)])
rec("Susu kedelai","Snack","1 gelas",B,False,[("susu_kedelai",250)])
rec("Keju & crackers","Snack","30 g keju + 5 crackers",M,False,[("keju",30),("crackers",20)])
rec("Cottage cheese & buah","Snack","1 cup",M,False,[("cottage",150),("beri",70)])
rec("Jagung rebus","Snack","1 tongkol",B,False,[("jagung",150)])
rec("Cokelat hitam","Snack","30 g",M,False,[("cokelat_hitam",30)])
rec("Smoothie pisang susu","Snack","1 gelas",B,False,[("pisang",120),("susu",200)])

def slug(n): return re.sub(r'[^a-z0-9]+','-',n.lower()).strip('-')
MEAT = {"meat","pork","fish","shellfish"}

# ---- validasi bahan: konsistensi Atwater per bahan ----
warn=[]
for k,v in ING.items():
    calc=4*v['p']+4*v['c']+9*v['f']
    if v['kcal']>0 and abs(calc-v['kcal'])>0.20*v['kcal']+5: warn.append((k,v['kcal'],round(calc)))
print("Bahan dengan kkal vs makro menyimpang >20%:",warn)

foods=[]
for name,parts in R.items():
    cat,serving,budget,indo=MENUS[name]
    kcal=p=c=f=g=0.0; allergens=set(); animals=set(); tags=set()
    for key,grams in parts:
        i=ING[key]; k=grams/100
        kcal+=i['kcal']*k; p+=i['p']*k; c+=i['c']*k; f+=i['f']*k; g+=grams
        allergens|=set(i['allergens']); tags|=set(i['tags'])
        if i['animal']: animals.add(i['animal'])
    t=set(tags)|allergens
    if indo: t.add("indonesian")
    if not (animals & MEAT):
        t.add("vegetarian")
        if not animals: t.add("vegan")
    if "pork" not in animals: t.add("halal")
    if p>=30: t.add("high protein")
    if kcal<=420: t.add("light")
    if budget==B: t.add("budget friendly")
    t.discard("pork") if "pork" in animals else None
    if "pork" in animals: t.add("pork")
    foods.append(dict(name=name,category=cat,serving=serving,serving_g=round(g),calories=round(kcal),
        protein_g=round(p,1),carbs_g=round(c,1),fat_g=round(f,1),tags=sorted(t),budget=budget,
        is_halal="pork" not in animals,ingredients=[dict(key=k,grams=gr) for k,gr in parts]))

# ---- bandingkan dengan estimasi lama ----
old={}
src=open('/tmp/gen_old.py').read().split("out=[]")[0]
ns={}; exec(src,ns)
for r in ns['rows']: old[r[0]]=r
print("\nMenu | lama kkal -> baru kkal | P C F baru | Atwater-check")
big=[]
for x in foods:
    o=old.get(x['name']); ok=o[3] if o else None
    atw=round(4*x['protein_g']+4*x['carbs_g']+9*x['fat_g'])
    flag="" if abs(atw-x['calories'])<=0.12*x['calories'] else " !atwater"
    if ok and abs(x['calories']-ok)>0.25*ok: big.append(x['name'])
    print(f"{x['name'][:32]:32} {ok if ok else '-':>5} -> {x['calories']:>4} | {x['protein_g']:>5} {x['carbs_g']:>5} {x['fat_g']:>5}{flag}")
print("\nSelisih >25% dari estimasi lama:",big)
missing=set(old)-set(R); extra=set(R)-set(old)
print("Nama hilang dari resep:",missing,"| nama baru:",extra)
assert len({slug(n) for n in R})==len(R)

json.dump([dict(key=k,name=v['name'],state=v['state'],kcal_100g=v['kcal'],protein_100g=v['p'],carbs_100g=v['c'],fat_100g=v['f'],source=v['source'],note=v['note']) for k,v in ING.items()],
          open("internal/seed/ingredients.json","w"),indent=1,ensure_ascii=False)
json.dump(foods,open("internal/seed/foods.json","w"),indent=1,ensure_ascii=False)
with open("docs/ingredients-review.csv","w",newline="") as fh:
    w=csv.writer(fh); w.writerow(["key","name","state","kcal_100g","protein_100g","carbs_100g","fat_100g","source","verified","note"])
    for k,v in ING.items(): w.writerow([k,v['name'],v['state'],v['kcal'],v['p'],v['c'],v['f'],v['source'],"FALSE",v['note']])
with open("docs/food-recipes.csv","w",newline="") as fh:
    w=csv.writer(fh); w.writerow(["menu","category","serving","serving_g","ingredient_key","ingredient","grams"])
    for x in foods:
        for ig in x['ingredients']: w.writerow([x['name'],x['category'],x['serving'],x['serving_g'],ig['key'],ING[ig['key']]['name'],ig['grams']])
with open("docs/food-images-manifest.csv","w",newline="") as fh:
    w=csv.writer(fh); w.writerow(['slug','filename','name','category','serving','suggested_image_prompt'])
    for x in foods:
        s=slug(x['name']); w.writerow([s,s+'.jpg',x['name'],x['category'],x['serving'],f"Overhead food photo of {x['name']} ({x['serving']}) served on a plate, natural daylight, clean background, no text, no people"])
print("\nOK:",len(foods),"menu,",len(ING),"bahan")
