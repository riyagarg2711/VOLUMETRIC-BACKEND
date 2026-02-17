CREATE TABLE IF NOT EXISTS role (
    id SMALLSERIAL PRIMARY KEY,
    user_role VARCHAR(255) NOT NULL UNIQUE,  
    description TEXT
);


INSERT INTO role (user_role, description) VALUES
('super_admin', 'Full access, including user management'),
('user', 'Basic access for scan creation/viewing');