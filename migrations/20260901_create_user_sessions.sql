ALTER TABLE user_devices
    ADD UNIQUE KEY uk_user_devices_id_user (id, user_id);

CREATE TABLE IF NOT EXISTS user_sessions (
    id CHAR(36) NOT NULL COMMENT '后端生成的Session UUID',
    user_id CHAR(36) NOT NULL,
    device_id CHAR(36) NOT NULL,

    status TINYINT UNSIGNED NOT NULL DEFAULT 1
        COMMENT '0=已失效，1=正常使用',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    refreshed_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    not_after DATETIME(3) NOT NULL COMMENT 'Session最晚有效时间',

    refresh_token_key VARBINARY(64) NOT NULL,
    refresh_token_counter BIGINT UNSIGNED NOT NULL DEFAULT 0,

    revoked_at DATETIME(3) DEFAULT NULL,
    revoke_reason VARCHAR(32) DEFAULT NULL,
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    KEY idx_sessions_user_device_status (user_id, device_id, status),
    KEY idx_sessions_device_user (device_id, user_id),
    KEY idx_sessions_status_not_after (status, not_after),

    CONSTRAINT fk_user_sessions_device_user
        FOREIGN KEY (device_id, user_id)
        REFERENCES user_devices (id, user_id)
        ON DELETE CASCADE
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;
