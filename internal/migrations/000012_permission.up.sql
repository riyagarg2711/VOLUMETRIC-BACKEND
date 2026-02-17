CREATE TABLE IF NOT EXISTS permission (
    id SMALLSERIAL PRIMARY KEY,
    permission VARCHAR(255) NOT NULL UNIQUE,  
    description TEXT
);


INSERT INTO permission (permission, description) VALUES
('create:scans', 'Create new scans'),
('read:scans', 'View scans'),
('update:scans', 'Update scans'),
('delete:scans', 'Delete scans'),
('create:coordinates', 'Add coordinates to scans'),
('read:results', 'View volume results'),
('admin:users', 'Manage users/roles');