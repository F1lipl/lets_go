USE `content_server`;

-- Revisions are created at publication; published_at is the retained timestamp.
DROP PROCEDURE IF EXISTS `migrate_007_remove_post_revision_created_at`;
DELIMITER //
CREATE PROCEDURE `migrate_007_remove_post_revision_created_at`()
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'post_revision'
       AND COLUMN_NAME = 'created_at'
  ) THEN
    ALTER TABLE `post_revision` DROP COLUMN `created_at`;
  END IF;
END//
DELIMITER ;
CALL `migrate_007_remove_post_revision_created_at`();
DROP PROCEDURE `migrate_007_remove_post_revision_created_at`;
