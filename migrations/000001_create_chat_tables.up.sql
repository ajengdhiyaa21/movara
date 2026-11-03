-- MOVARA chat MVP schema.
-- Execute only against the approved test database after charset/collation review.
-- This migration intentionally does not set database/table collation.

CREATE TABLE chat_conversations (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    pm_user_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_chat_conversations_pm_user_id (pm_user_id),
    CONSTRAINT fk_chat_conversations_pm_user
        FOREIGN KEY (pm_user_id) REFERENCES users (id)
        ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB;

CREATE TABLE chat_messages (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    conversation_id BIGINT UNSIGNED NOT NULL,
    sender_user_id BIGINT UNSIGNED NOT NULL,
    client_message_id VARCHAR(100) NOT NULL,
    body VARCHAR(3000) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_chat_messages_conversation_client_id (conversation_id, client_message_id),
    KEY idx_chat_messages_conversation_id_id (conversation_id, id),
    KEY idx_chat_messages_sender_created_at (sender_user_id, created_at),
    CONSTRAINT fk_chat_messages_conversation
        FOREIGN KEY (conversation_id) REFERENCES chat_conversations (id)
        ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT fk_chat_messages_sender
        FOREIGN KEY (sender_user_id) REFERENCES users (id)
        ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB;

CREATE TABLE chat_read_states (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    conversation_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    last_read_message_id BIGINT UNSIGNED NULL,
    last_read_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_chat_read_states_conversation_user (conversation_id, user_id),
    KEY idx_chat_read_states_user_conversation (user_id, conversation_id),
    KEY idx_chat_read_states_last_read_message (last_read_message_id),
    CONSTRAINT fk_chat_read_states_conversation
        FOREIGN KEY (conversation_id) REFERENCES chat_conversations (id)
        ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT fk_chat_read_states_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB;
