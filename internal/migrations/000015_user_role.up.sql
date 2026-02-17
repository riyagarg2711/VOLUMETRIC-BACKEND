CREATE TABLE IF NOT EXISTS user_role (
    user_id CHAR(36) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    role_id SMALLINT NOT NULL REFERENCES role(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);