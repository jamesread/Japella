CREATE TABLE `campaign_social_accounts` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `social_account` int(10) unsigned NOT NULL,
  `campaign` int(10) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_csa_social_account` (`social_account`),
  KEY `fk_csa_campaign` (`campaign`),
  CONSTRAINT `fk_csa_social_account` FOREIGN KEY (`social_account`) REFERENCES `social_accounts` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_csa_campaign` FOREIGN KEY (`campaign`) REFERENCES `campaigns` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
