CREATE TABLE IF NOT EXISTS users (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name        VARCHAR(100) NOT NULL,
  email       VARCHAR(191) NOT NULL,
  api_key     CHAR(64)     NOT NULL COMMENT 'SHA-256 hash dari API key (plaintext tidak disimpan)',
  created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_users_email (email),
  UNIQUE KEY uq_users_api_key (api_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS user_profiles (
  user_id           BIGINT UNSIGNED NOT NULL PRIMARY KEY,
  calorie_target    INT           NOT NULL DEFAULT 2000,
  protein_target    INT           NOT NULL DEFAULT 100,
  carbs_target      INT           NOT NULL DEFAULT 250,
  fat_target        INT           NOT NULL DEFAULT 65,
  water_target      DECIMAL(4,2)  NOT NULL DEFAULT 2.50 COMMENT 'liter',
  diet_preference   VARCHAR(30)   NOT NULL DEFAULT 'none',
  is_halal          TINYINT(1)    NOT NULL DEFAULT 0,
  allergies_json    JSON          NULL,
  budget_preference VARCHAR(20)   NOT NULL DEFAULT 'moderate',
  CONSTRAINT fk_profiles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS meal_logs (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id       BIGINT UNSIGNED NOT NULL,
  item_name     VARCHAR(150) NOT NULL,
  meal_category ENUM('Breakfast','Lunch','Dinner','Snack') NOT NULL,
  calories      INT          NOT NULL DEFAULT 0,
  protein       DECIMAL(6,1) NOT NULL DEFAULT 0,
  carbs         DECIMAL(6,1) NOT NULL DEFAULT 0,
  fat           DECIMAL(6,1) NOT NULL DEFAULT 0,
  eaten_at      DATETIME(3)  NOT NULL,
  created_at    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_meal_user_eaten (user_id, eaten_at),
  CONSTRAINT fk_meals_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS water_logs (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id    BIGINT UNSIGNED NOT NULL,
  amount_ml  INT         NOT NULL,
  logged_at  DATETIME(3) NOT NULL,
  KEY idx_water_user_logged (user_id, logged_at),
  CONSTRAINT fk_water_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS favorite_meals (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id    BIGINT UNSIGNED NOT NULL,
  item_name  VARCHAR(150) NOT NULL,
  calories   INT          NOT NULL DEFAULT 0,
  protein    DECIMAL(6,1) NOT NULL DEFAULT 0,
  carbs      DECIMAL(6,1) NOT NULL DEFAULT 0,
  fat        DECIMAL(6,1) NOT NULL DEFAULT 0,
  tags_json  JSON         NULL,
  KEY idx_fav_user (user_id),
  CONSTRAINT fk_fav_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
