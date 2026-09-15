USE `content_server`;

-- Stop content writers and back up before applying this migration.
-- Draft content and draft_version are retained; only the last request marker
-- is retired. The conditional DDL supports retry after interruption.
DROP PROCEDURE IF EXISTS `migrate_004_remove_draft_save_request_id`;
DELIMITER //
CREATE PROCEDURE `migrate_004_remove_draft_save_request_id`()
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'post_draft'
       AND COLUMN_NAME = 'last_save_request_id'
  ) THEN
    ALTER TABLE `post_draft` DROP COLUMN `last_save_request_id`;
  END IF;
END//
DELIMITER ;
CALL `migrate_004_remove_draft_save_request_id`();
DROP PROCEDURE `migrate_004_remove_draft_save_request_id`;
