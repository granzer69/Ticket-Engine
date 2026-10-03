-- Ticket Engine schema (V1-compatible, evolved in later phases)
CREATE TABLE IF NOT EXISTS tickets (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NULL,
    sold_at DATETIME(3) NULL,
    INDEX idx_tickets_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
