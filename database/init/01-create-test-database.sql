-- Create test database for running tests
CREATE DATABASE IF NOT EXISTS kibaship_test;

-- Grant all privileges to root user on test database
GRANT ALL PRIVILEGES ON kibaship_test.* TO 'root'@'%';

-- Flush privileges to ensure changes take effect
FLUSH PRIVILEGES;
