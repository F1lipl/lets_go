USE `content_server`;

CREATE TABLE IF NOT EXISTS `event_delivery` (
  `event_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `consumer_name` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '1 pending, 2 processing, 3 succeeded, 4 superseded, 5 failed',
  `attempt_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Increment on every claim; an expired task increments again only when re-claimed',
  `max_attempts` INT UNSIGNED NOT NULL DEFAULT 10,
  `next_attempt_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `claim_token` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NULL,
  `locked_until` DATETIME(3) NULL,
  `last_error` VARCHAR(1000) NOT NULL DEFAULT '',
  `completed_at` DATETIME(3) NULL COMMENT 'Terminal time for succeeded, superseded or failed',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`event_id`, `consumer_name`),
  KEY `idx_delivery_ready` (`status`, `next_attempt_at`, `event_id`, `consumer_name`),
  KEY `idx_delivery_consumer_ready` (`consumer_name`, `status`, `next_attempt_at`, `event_id`),
  KEY `idx_delivery_expired` (`status`, `locked_until`, `event_id`, `consumer_name`),
  KEY `idx_delivery_completed` (`status`, `completed_at`),
  CONSTRAINT `fk_delivery_event` FOREIGN KEY (`event_id`) REFERENCES `outbox_event` (`event_id`) ON DELETE RESTRICT,
  CONSTRAINT `chk_delivery_status` CHECK (`status` BETWEEN 1 AND 5),
  CONSTRAINT `chk_delivery_attempts` CHECK (`max_attempts` > 0),
  CONSTRAINT `chk_delivery_claim` CHECK (
    (`status` = 2 AND `claim_token` IS NOT NULL AND `locked_until` IS NOT NULL)
    OR (`status` <> 2 AND `claim_token` IS NULL AND `locked_until` IS NULL)
  ),
  CONSTRAINT `chk_delivery_completion` CHECK (
    (`status` IN (3,4,5) AND `completed_at` IS NOT NULL)
    OR (`status` IN (1,2) AND `completed_at` IS NULL)
  )
) ENGINE=InnoDB COMMENT='One durable task per event and consumer; payload is stored in outbox_event';

-- Fan-out runs in one short transaction. State 2 is reserved for legacy use;
-- do not commit it in the local fan-out implementation.
ALTER TABLE `outbox_event`
  MODIFY COLUMN `status` TINYINT UNSIGNED NOT NULL DEFAULT 1
    COMMENT '1 pending dispatch, 2 reserved, 3 dispatched, 4 dispatch failed',
  MODIFY COLUMN `published_at` DATETIME(3) NULL
    COMMENT 'Fan-out committed at; does not mean all consumers completed';

-- Preserve existing values while adding the terminal no-op result.
ALTER TABLE `inbox_event`
  MODIFY COLUMN `status` TINYINT UNSIGNED NOT NULL DEFAULT 1
    COMMENT '1 processing, 2 processed, 3 failed, 4 superseded',
  DROP CHECK `chk_inbox_status`,
  ADD CONSTRAINT `chk_inbox_status` CHECK (`status` BETWEEN 1 AND 4);
