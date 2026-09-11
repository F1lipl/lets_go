CREATE DATABASE IF NOT EXISTS `content_server`
  CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
USE `content_server`;

CREATE TABLE IF NOT EXISTS `post` (
  `post_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `author_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `create_request_id` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `lifecycle_status` TINYINT UNSIGNED NOT NULL COMMENT '1 draft, 2 publishing, 3 published, 4 deleted',
  `visibility` TINYINT UNSIGNED NOT NULL COMMENT '1 public, 2 followers, 3 private',
  `availability_status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '1 normal, 2 pending, 3 hidden',
  `published_revision_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NULL,
  `post_version` BIGINT UNSIGNED NOT NULL DEFAULT 1,
  `first_published_at` DATETIME(3) NULL,
  `last_published_at` DATETIME(3) NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` DATETIME(3) NULL,
  PRIMARY KEY (`post_id`),
  UNIQUE KEY `uk_post_author_create_request` (`author_id`, `create_request_id`),
  KEY `idx_post_author_updated` (`author_id`, `lifecycle_status`, `updated_at`, `post_id`),
  KEY `idx_post_feed_status` (`lifecycle_status`, `visibility`, `availability_status`, `last_published_at`, `post_id`),
  CONSTRAINT `chk_post_lifecycle_status` CHECK (`lifecycle_status` BETWEEN 1 AND 4),
  CONSTRAINT `chk_post_visibility` CHECK (`visibility` BETWEEN 1 AND 3),
  CONSTRAINT `chk_post_availability_status` CHECK (`availability_status` BETWEEN 1 AND 3)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `media_asset` (
  `asset_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `owner_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `upload_request_id` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `storage_provider` VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `bucket_name` VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `object_key` VARCHAR(512) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `mime_type` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `file_size` BIGINT UNSIGNED NOT NULL,
  `width` INT UNSIGNED NULL,
  `height` INT UNSIGNED NULL,
  `content_hash` CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  `status` TINYINT UNSIGNED NOT NULL COMMENT '1 uploading, 2 processing, 3 ready, 4 failed, 5 deleted',
  `failure_code` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` DATETIME(3) NULL,
  PRIMARY KEY (`asset_id`),
  UNIQUE KEY `uk_media_owner_upload_request` (`owner_id`, `upload_request_id`),
  UNIQUE KEY `uk_media_storage_object` (`storage_provider`, `bucket_name`, `object_key`),
  KEY `idx_media_owner_status` (`owner_id`, `status`, `created_at`),
  KEY `idx_media_content_hash` (`content_hash`),
  CONSTRAINT `chk_media_asset_status` CHECK (`status` BETWEEN 1 AND 5)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `media_variant` (
  `variant_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `asset_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `variant_type` TINYINT UNSIGNED NOT NULL COMMENT '1 thumbnail, 2 large',
  `object_key` VARCHAR(512) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `mime_type` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `file_size` BIGINT UNSIGNED NOT NULL,
  `width` INT UNSIGNED NOT NULL,
  `height` INT UNSIGNED NOT NULL,
  `status` TINYINT UNSIGNED NOT NULL COMMENT '1 processing, 2 ready, 3 failed, 4 deleted',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`variant_id`),
  UNIQUE KEY `uk_media_variant_type` (`asset_id`, `variant_type`),
  KEY `idx_media_variant_status` (`status`, `updated_at`),
  CONSTRAINT `fk_media_variant_asset` FOREIGN KEY (`asset_id`) REFERENCES `media_asset` (`asset_id`),
  CONSTRAINT `chk_media_variant_type` CHECK (`variant_type` BETWEEN 1 AND 2),
  CONSTRAINT `chk_media_variant_status` CHECK (`status` BETWEEN 1 AND 4)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `tag` (
  `tag_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `normalized_name` VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `display_name` VARCHAR(64) NOT NULL,
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '1 active, 2 hidden, 3 disabled',
  `created_by` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`tag_id`),
  UNIQUE KEY `uk_tag_normalized_name` (`normalized_name`),
  KEY `idx_tag_status_name` (`status`, `normalized_name`),
  CONSTRAINT `chk_tag_status` CHECK (`status` BETWEEN 1 AND 3)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `post_draft` (
  `post_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `draft_version` BIGINT UNSIGNED NOT NULL DEFAULT 1,
  `last_save_request_id` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `title` VARCHAR(120) NOT NULL DEFAULT '',
  `summary` VARCHAR(500) NOT NULL DEFAULT '',
  `cover_asset_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NULL,
  `cover_focus_x` DECIMAL(6,5) NULL,
  `cover_focus_y` DECIMAL(6,5) NULL,
  `cover_crop_style` VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
  `presentation_mode` TINYINT UNSIGNED NOT NULL COMMENT '1 traditional, 2 route',
  `document_schema_version` INT UNSIGNED NOT NULL,
  `document_json` JSON NOT NULL,
  `tag_names_json` JSON NOT NULL,
  `route_draft_id` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  `route_draft_version` BIGINT UNSIGNED NULL,
  `plain_text` MEDIUMTEXT NOT NULL,
  `block_count` INT UNSIGNED NOT NULL DEFAULT 0,
  `image_count` INT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`post_id`),
  CONSTRAINT `fk_post_draft_post` FOREIGN KEY (`post_id`) REFERENCES `post` (`post_id`),
  CONSTRAINT `fk_post_draft_cover` FOREIGN KEY (`cover_asset_id`) REFERENCES `media_asset` (`asset_id`),
  CONSTRAINT `chk_post_draft_presentation` CHECK (`presentation_mode` BETWEEN 1 AND 2),
  CONSTRAINT `chk_post_draft_focus_x` CHECK (`cover_focus_x` IS NULL OR (`cover_focus_x` BETWEEN 0 AND 1)),
  CONSTRAINT `chk_post_draft_focus_y` CHECK (`cover_focus_y` IS NULL OR (`cover_focus_y` BETWEEN 0 AND 1))
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `post_revision` (
  `revision_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `post_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `revision_number` BIGINT UNSIGNED NOT NULL,
  `source_draft_version` BIGINT UNSIGNED NOT NULL,
  `publish_request_id` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `title` VARCHAR(120) NOT NULL,
  `summary` VARCHAR(500) NOT NULL DEFAULT '',
  `cover_asset_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NULL,
  `cover_focus_x` DECIMAL(6,5) NULL,
  `cover_focus_y` DECIMAL(6,5) NULL,
  `cover_crop_style` VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
  `presentation_mode` TINYINT UNSIGNED NOT NULL,
  `document_schema_version` INT UNSIGNED NOT NULL,
  `document_json` JSON NOT NULL,
  `plain_text` MEDIUMTEXT NOT NULL,
  `route_snapshot_id` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  `route_snapshot_version` BIGINT UNSIGNED NULL,
  `block_count` INT UNSIGNED NOT NULL DEFAULT 0,
  `image_count` INT UNSIGNED NOT NULL DEFAULT 0,
  `published_at` DATETIME(3) NOT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`revision_id`),
  UNIQUE KEY `uk_post_revision_number` (`post_id`, `revision_number`),
  UNIQUE KEY `uk_post_publish_request` (`post_id`, `publish_request_id`),
  KEY `idx_post_revision_published` (`post_id`, `published_at`),
  CONSTRAINT `fk_post_revision_post` FOREIGN KEY (`post_id`) REFERENCES `post` (`post_id`),
  CONSTRAINT `fk_post_revision_cover` FOREIGN KEY (`cover_asset_id`) REFERENCES `media_asset` (`asset_id`),
  CONSTRAINT `chk_post_revision_presentation` CHECK (`presentation_mode` BETWEEN 1 AND 2),
  CONSTRAINT `chk_post_revision_focus_x` CHECK (`cover_focus_x` IS NULL OR (`cover_focus_x` BETWEEN 0 AND 1)),
  CONSTRAINT `chk_post_revision_focus_y` CHECK (`cover_focus_y` IS NULL OR (`cover_focus_y` BETWEEN 0 AND 1))
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `post_revision_tag` (
  `revision_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `tag_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `tag_name_snapshot` VARCHAR(64) NOT NULL,
  `sort_order` SMALLINT UNSIGNED NOT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`revision_id`, `tag_id`),
  UNIQUE KEY `uk_revision_tag_order` (`revision_id`, `sort_order`),
  KEY `idx_revision_tag_lookup` (`tag_id`, `revision_id`),
  CONSTRAINT `fk_revision_tag_revision` FOREIGN KEY (`revision_id`) REFERENCES `post_revision` (`revision_id`),
  CONSTRAINT `fk_revision_tag_tag` FOREIGN KEY (`tag_id`) REFERENCES `tag` (`tag_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `post_revision_place` (
  `revision_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `place_id` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `block_id` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `sort_order` INT UNSIGNED NOT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`revision_id`, `place_id`, `block_id`),
  KEY `idx_revision_place_lookup` (`place_id`, `revision_id`),
  CONSTRAINT `fk_revision_place_revision` FOREIGN KEY (`revision_id`) REFERENCES `post_revision` (`revision_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `post_card_projection` (
  `post_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `revision_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `author_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `title` VARCHAR(120) NOT NULL,
  `summary` VARCHAR(500) NOT NULL DEFAULT '',
  `cover_asset_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NULL,
  `presentation_mode` TINYINT UNSIGNED NOT NULL,
  `route_snapshot_id` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  `route_day_count` SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  `route_node_count` INT UNSIGNED NOT NULL DEFAULT 0,
  `visibility` TINYINT UNSIGNED NOT NULL,
  `availability_status` TINYINT UNSIGNED NOT NULL,
  `published_at` DATETIME(3) NOT NULL,
  `source_post_version` BIGINT UNSIGNED NOT NULL,
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`post_id`),
  KEY `idx_card_feed` (`visibility`, `availability_status`, `published_at`, `post_id`),
  KEY `idx_card_author` (`author_id`, `published_at`, `post_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `post_stats` (
  `post_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `like_count` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `favorite_count` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `comment_count` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `view_count` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `source_version` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`post_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `author_state_projection` (
  `user_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `state` TINYINT UNSIGNED NOT NULL COMMENT '1 active, 2 restricted, 3 disabled',
  `source_version` BIGINT UNSIGNED NOT NULL,
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`user_id`),
  CONSTRAINT `chk_author_state` CHECK (`state` BETWEEN 1 AND 3)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `content_asset_ref` (
  `owner_type` TINYINT UNSIGNED NOT NULL COMMENT '1 draft, 2 revision',
  `owner_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `usage_type` TINYINT UNSIGNED NOT NULL COMMENT '1 cover, 2 block',
  `block_id` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
  `asset_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `sort_order` INT UNSIGNED NOT NULL DEFAULT 0,
  `source_version` BIGINT UNSIGNED NOT NULL,
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`owner_type`, `owner_id`, `usage_type`, `block_id`, `sort_order`),
  KEY `idx_content_asset_ref_asset` (`asset_id`, `owner_type`, `owner_id`),
  CONSTRAINT `chk_content_asset_owner_type` CHECK (`owner_type` BETWEEN 1 AND 2),
  CONSTRAINT `chk_content_asset_usage_type` CHECK (`usage_type` BETWEEN 1 AND 2)
) ENGINE=InnoDB COMMENT='Rebuildable projection extracted from document_json';

CREATE TABLE IF NOT EXISTS `outbox_event` (
  `event_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `aggregate_type` VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `aggregate_id` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `event_type` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `payload_json` JSON NOT NULL,
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '1 pending, 2 publishing, 3 published, 4 failed',
  `attempt_count` INT UNSIGNED NOT NULL DEFAULT 0,
  `next_attempt_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `occurred_at` DATETIME(3) NOT NULL,
  `published_at` DATETIME(3) NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`event_id`),
  KEY `idx_outbox_dispatch` (`status`, `next_attempt_at`, `event_id`),
  KEY `idx_outbox_aggregate` (`aggregate_type`, `aggregate_id`, `occurred_at`),
  CONSTRAINT `chk_outbox_status` CHECK (`status` BETWEEN 1 AND 4)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `inbox_event` (
  `consumer_name` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `event_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `event_type` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '1 processing, 2 processed, 3 failed',
  `attempt_count` INT UNSIGNED NOT NULL DEFAULT 0,
  `payload_json` JSON NOT NULL,
  `received_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `processed_at` DATETIME(3) NULL,
  `last_error` VARCHAR(500) NOT NULL DEFAULT '',
  PRIMARY KEY (`consumer_name`, `event_id`),
  KEY `idx_inbox_recovery` (`consumer_name`, `status`, `received_at`),
  CONSTRAINT `chk_inbox_status` CHECK (`status` BETWEEN 1 AND 3)
) ENGINE=InnoDB;
