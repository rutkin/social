-- Set client encoding to UTF-8
SET client_encoding = 'UTF8';

-- Create temporary table for import
CREATE TEMP TABLE temp_users (
    full_name TEXT,
    birthdate DATE,
    city TEXT
);

-- Copy data from CSV file
COPY temp_users FROM '/tmp/people.v2.csv' WITH (FORMAT csv, DELIMITER ',', NULL '');

-- Insert data into users table with generated UUIDs and hashed passwords
INSERT INTO users (id, first_name, last_name, birthdate, biography, city, password)
SELECT 
    gen_random_uuid(),
    split_part(full_name, ' ', 2), -- first name is the second part
    split_part(full_name, ' ', 1), -- last name is the first part
    birthdate,
    '', -- empty biography
    city,
    -- Use a default password for all imported users
    '$2a$14$8dI5vF7ZxVvZxVvZxVvZxO' -- This is a bcrypt hash of 'password123'
FROM temp_users;

-- Drop temporary table
DROP TABLE temp_users; 