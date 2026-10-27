-- Bahan pangan per 100 g (diimpor lewat `go run ./cmd/importer ingredients ...`)
CREATE TABLE IF NOT EXISTS ingredients (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name         VARCHAR(200)  NOT NULL,
  calories_100g DECIMAL(7,2) NOT NULL,
  protein_100g  DECIMAL(7,2) NOT NULL,
  carbs_100g    DECIMAL(7,2) NOT NULL,
  fat_100g      DECIMAL(7,2) NOT NULL,
  source       VARCHAR(20)   NOT NULL COMMENT 'tkpi/usda/custom',
  source_ref   VARCHAR(100)  NULL,
  code         VARCHAR(50)   NULL COMMENT 'kunci seed (NULL untuk hasil impor)',
  state        VARCHAR(20)   NULL COMMENT 'raw/cooked/dry/as_eaten',
  verified     TINYINT(1)    NOT NULL DEFAULT 0 COMMENT '1 = direview ahli gizi; seed tidak menimpa',
  note         VARCHAR(300)  NULL,
  UNIQUE KEY uq_ing_code (code),
  KEY idx_ing_name (name),
  KEY idx_ing_source (source)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Resep: menu = sum(bahan x gram)
CREATE TABLE IF NOT EXISTS food_components (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  food_id       BIGINT UNSIGNED NOT NULL,
  ingredient_id BIGINT UNSIGNED NOT NULL,
  grams         DECIMAL(7,1) NOT NULL,
  UNIQUE KEY uq_food_ingredient (food_id, ingredient_id),
  CONSTRAINT fk_fc_food FOREIGN KEY (food_id) REFERENCES foods(id) ON DELETE CASCADE,
  CONSTRAINT fk_fc_ing  FOREIGN KEY (ingredient_id) REFERENCES ingredients(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
