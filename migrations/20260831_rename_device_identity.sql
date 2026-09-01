ALTER TABLE user_devices
    MODIFY COLUMN id BIGINT UNSIGNED NOT NULL;

ALTER TABLE user_devices
    ADD COLUMN record_id CHAR(36) NULL FIRST;

UPDATE user_devices
SET record_id = UUID()
WHERE record_id IS NULL;

ALTER TABLE user_devices
    MODIFY COLUMN record_id CHAR(36) NOT NULL;

ALTER TABLE user_devices
    DROP PRIMARY KEY;

ALTER TABLE user_devices
    DROP COLUMN id;

ALTER TABLE user_devices
    CHANGE COLUMN record_id id CHAR(36) NOT NULL;

ALTER TABLE user_devices
    DROP INDEX uk_user_device,
    CHANGE COLUMN device_id client_instance_id VARCHAR(128)
        COLLATE utf8mb4_unicode_ci NOT NULL
        COMMENT '后端签发的客户端实例标识',
    ADD COLUMN created_at DATETIME(3) NOT NULL
        DEFAULT CURRENT_TIMESTAMP(3),
    ADD COLUMN updated_at DATETIME(3) NOT NULL
        DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),
    ADD PRIMARY KEY (id),
    ADD UNIQUE KEY uk_user_client_instance (user_id, client_instance_id);
