-- Repair / align social_account_shares with migration 16 (can_post, can_manage).
-- IF NOT EXISTS: safe when 16 already created these columns (MariaDB 10.5+).
ALTER TABLE social_account_shares
  ADD COLUMN IF NOT EXISTS can_post TINYINT(1) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS can_manage TINYINT(1) NOT NULL DEFAULT 0;
