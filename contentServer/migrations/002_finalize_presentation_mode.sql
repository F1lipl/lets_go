USE `content_server`;

-- presentation_mode is part of the editable draft, the immutable revision,
-- and the current published PostCard projection. It is deliberately absent
-- from the post aggregate root because draft and published modes can differ.
DROP PROCEDURE IF EXISTS `migrate_finalize_presentation_mode`;

DELIMITER //
CREATE PROCEDURE `migrate_finalize_presentation_mode`()
BEGIN
  ALTER TABLE `post_draft`
    MODIFY COLUMN `presentation_mode` TINYINT UNSIGNED NOT NULL DEFAULT 1
      COMMENT '1 standard, 2 route_graph';

  ALTER TABLE `post_revision`
    MODIFY COLUMN `presentation_mode` TINYINT UNSIGNED NOT NULL
      COMMENT '1 standard, 2 route_graph';

  ALTER TABLE `post_card_projection`
    MODIFY COLUMN `presentation_mode` TINYINT UNSIGNED NOT NULL
      COMMENT '1 standard, 2 route_graph';

  IF NOT EXISTS (
    SELECT 1
      FROM information_schema.TABLE_CONSTRAINTS
     WHERE CONSTRAINT_SCHEMA = DATABASE()
       AND TABLE_NAME = 'post_card_projection'
       AND CONSTRAINT_NAME = 'chk_post_card_presentation'
       AND CONSTRAINT_TYPE = 'CHECK'
  ) THEN
    ALTER TABLE `post_card_projection`
      ADD CONSTRAINT `chk_post_card_presentation`
        CHECK (`presentation_mode` BETWEEN 1 AND 2);
  END IF;
END//
DELIMITER ;

CALL `migrate_finalize_presentation_mode`();
DROP PROCEDURE `migrate_finalize_presentation_mode`;
