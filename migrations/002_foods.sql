-- Dataset makanan. Data awal di-seed otomatis oleh aplikasi saat start (internal/seed/foods.json).
CREATE TABLE IF NOT EXISTS foods (
  id        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name      VARCHAR(150) NOT NULL,
  category  VARCHAR(20)  NOT NULL COMMENT 'Breakfast/Lunch/Dinner/Snack',
  serving   VARCHAR(80)  NULL,
  serving_g INT          NULL COMMENT 'gram per porsi',
  calories  INT          NOT NULL,
  protein   DECIMAL(6,1) NOT NULL,
  carbs     DECIMAL(6,1) NOT NULL,
  fat       DECIMAL(6,1) NOT NULL,
  tags_json JSON         NULL,
  budget    VARCHAR(20)  NOT NULL DEFAULT 'moderate' COMMENT 'budget_friendly/moderate/premium',
  is_halal  TINYINT(1)   NOT NULL DEFAULT 1,
  source    VARCHAR(30)  NOT NULL DEFAULT 'recipe' COMMENT 'recipe/estimate/tkpi/custom/computed:*',
  slug         VARCHAR(160) NULL COMMENT 'nama file gambar: <slug>.jpg',
  image_url    VARCHAR(500) NULL COMMENT 'override URL gambar',
  image_credit VARCHAR(200) NULL,
  UNIQUE KEY uq_foods_name (name),
  KEY idx_foods_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
