CREATE TABLE IF NOT EXISTS role_permission (
    role_id SMALLINT NOT NULL REFERENCES role(id) ON DELETE CASCADE,
    permission_id SMALLINT NOT NULL REFERENCES permission(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

