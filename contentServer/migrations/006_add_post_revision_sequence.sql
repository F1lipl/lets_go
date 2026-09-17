USE `content_server`;

-- Apply with application writes stopped, as required by the migration runner.
DROP PROCEDURE IF EXISTS `migrate_006_add_post_revision_sequence`;
DELIMITER //
CREATE PROCEDURE `migrate_006_add_post_revision_sequence`()
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'post'
       AND COLUMN_NAME = 'revision_sequence'
  ) THEN
    ALTER TABLE `post`
      ADD COLUMN `revision_sequence` BIGINT UNSIGNED NOT NULL DEFAULT 0
      COMMENT 'Latest allocated publication number; 0 before first publication'
      AFTER `published_revision_id`;
  END IF;

  -- Preserve the highest existing number, including on a migration retry.
  UPDATE `post` p
  JOIN (
    SELECT `post_id`, MAX(`revision_number`) AS max_revision_number
      FROM `post_revision`
     GROUP BY `post_id`
  ) r ON r.post_id = p.post_id
     SET p.revision_sequence = r.max_revision_number,
         p.updated_at = p.updated_at
   WHERE p.revision_sequence < r.max_revision_number;
END//
DELIMITER ;
CALL `migrate_006_add_post_revision_sequence`();
DROP PROCEDURE `migrate_006_add_post_revision_sequence`;
