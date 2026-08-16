CREATE TABLE `post_media` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `post_id` int(10) unsigned NOT NULL,
  `media_url` varchar(500) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_post_media_post_id` (`post_id`),
  KEY `idx_post_media_media_url` (`media_url`),
  CONSTRAINT `fk_post_media_post` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
