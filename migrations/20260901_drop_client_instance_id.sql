ALTER TABLE user_devices
    ADD KEY idx_user_devices_user_id (user_id),
    DROP INDEX uk_user_client_instance,
    DROP COLUMN client_instance_id,
    MODIFY COLUMN id CHAR(36) NOT NULL
        COMMENT '后端生成并返回给客户端的设备实例标识';
