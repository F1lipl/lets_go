USE `content_server`;

-- Operation request markers are retired. Updates continue to use version
-- columns for optimistic concurrency. Conditional DDL permits safe retries.
DROP PROCEDURE IF EXISTS `migrate_005_remove_operation_request_ids`;
DELIMITER //
CREATE PROCEDURE `migrate_005_remove_operation_request_ids`()
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.STATISTICS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'post'
       AND INDEX_NAME = 'uk_post_author_create_request'
  ) THEN
    ALTER TABLE `post` DROP INDEX `uk_post_author_create_request`;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'post'
       AND COLUMN_NAME = 'create_request_id'
  ) THEN
    ALTER TABLE `post` DROP COLUMN `create_request_id`;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.STATISTICS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'post_revision'
       AND INDEX_NAME = 'uk_post_publish_request'
  ) THEN
    ALTER TABLE `post_revision` DROP INDEX `uk_post_publish_request`;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'post_revision'
       AND COLUMN_NAME = 'publish_request_id'
  ) THEN
    ALTER TABLE `post_revision` DROP COLUMN `publish_request_id`;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.STATISTICS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'media_asset'
       AND INDEX_NAME = 'uk_media_owner_upload_request'
  ) THEN
    ALTER TABLE `media_asset` DROP INDEX `uk_media_owner_upload_request`;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'media_asset'
       AND COLUMN_NAME = 'upload_request_id'
  ) THEN
    ALTER TABLE `media_asset` DROP COLUMN `upload_request_id`;
  END IF;
END//
DELIMITER ;
CALL `migrate_005_remove_operation_request_ids`();
DROP PROCEDURE `migrate_005_remove_operation_request_ids`;
