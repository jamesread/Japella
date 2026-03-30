-- No-op: can_post and can_manage may have been created by migration 16; dropping them here
-- would break schema when rolled back past 19 but not past 16.
SELECT 1;
