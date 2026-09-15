USE `content_server`;

-- Run with content writers stopped. MySQL DDL commits independently; this
-- migration can resume after interruption by inspecting remaining columns.
-- This MVP migration only supports empty affected tables. Populated content
-- needs an explicit export/conversion plan, including document_json bindings.
DROP PROCEDURE IF EXISTS `migrate_003_detach_trip`;
DELIMITER //
CREATE PROCEDURE `migrate_003_detach_trip`()
BEGIN
  DECLARE remaining INT DEFAULT 0;
  DECLARE target_table VARCHAR(64);
  DECLARE target_column VARCHAR(64);
  DECLARE finished BOOLEAN DEFAULT FALSE;
  DECLARE obsolete_columns CURSOR FOR
    SELECT TABLE_NAME, COLUMN_NAME
      FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND (
         (TABLE_NAME = 'post_draft' AND COLUMN_NAME IN
           ('presentation_mode', 'route_draft_id', 'route_draft_version'))
         OR (TABLE_NAME = 'post_revision' AND COLUMN_NAME IN
           ('presentation_mode', 'route_snapshot_id', 'route_snapshot_version'))
         OR (TABLE_NAME = 'post_card_projection' AND COLUMN_NAME IN
           ('presentation_mode', 'route_snapshot_id', 'route_day_count', 'route_node_count'))
       )
     ORDER BY TABLE_NAME, ORDINAL_POSITION;
  DECLARE CONTINUE HANDLER FOR NOT FOUND SET finished = TRUE;

  SELECT COUNT(*) INTO remaining FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME IN ('post_draft', 'post_revision', 'post_card_projection')
     AND COLUMN_NAME IN ('presentation_mode', 'route_draft_id', 'route_draft_version',
       'route_snapshot_id', 'route_snapshot_version', 'route_day_count', 'route_node_count');
  IF remaining > 0 AND (
    EXISTS (SELECT 1 FROM post_draft LIMIT 1)
    OR EXISTS (SELECT 1 FROM post_revision LIMIT 1)
    OR EXISTS (SELECT 1 FROM post_card_projection LIMIT 1)
  ) THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '003 requires empty affected tables; export and convert existing content first';
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.TABLES
      WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'post_revision_place') THEN
    SET @migration_003_sql = 'SELECT COUNT(*) INTO @migration_003_place_count FROM post_revision_place';
    PREPARE migration_003_stmt FROM @migration_003_sql;
    EXECUTE migration_003_stmt;
    DEALLOCATE PREPARE migration_003_stmt;
    IF @migration_003_place_count > 0 THEN
      SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = '003 requires empty post_revision_place; export and convert bindings first';
    END IF;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
      WHERE CONSTRAINT_SCHEMA = DATABASE() AND TABLE_NAME = 'post_draft'
        AND CONSTRAINT_NAME = 'chk_post_draft_presentation') THEN
    ALTER TABLE post_draft DROP CHECK chk_post_draft_presentation;
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
      WHERE CONSTRAINT_SCHEMA = DATABASE() AND TABLE_NAME = 'post_revision'
        AND CONSTRAINT_NAME = 'chk_post_revision_presentation') THEN
    ALTER TABLE post_revision DROP CHECK chk_post_revision_presentation;
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
      WHERE CONSTRAINT_SCHEMA = DATABASE() AND TABLE_NAME = 'post_card_projection'
        AND CONSTRAINT_NAME = 'chk_post_card_presentation') THEN
    ALTER TABLE post_card_projection DROP CHECK chk_post_card_presentation;
  END IF;

  OPEN obsolete_columns;
  drop_columns: LOOP
    FETCH obsolete_columns INTO target_table, target_column;
    IF finished THEN LEAVE drop_columns; END IF;
    SET @migration_003_sql = CONCAT('ALTER TABLE `', target_table,
      '` DROP COLUMN `', target_column, '`');
    PREPARE migration_003_stmt FROM @migration_003_sql;
    EXECUTE migration_003_stmt;
    DEALLOCATE PREPARE migration_003_stmt;
  END LOOP;
  CLOSE obsolete_columns;

  DROP TABLE IF EXISTS post_revision_place;
END//
DELIMITER ;
CALL `migrate_003_detach_trip`();
DROP PROCEDURE `migrate_003_detach_trip`;
