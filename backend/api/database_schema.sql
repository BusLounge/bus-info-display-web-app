CREATE TABLE IF NOT EXISTS advertisements (
    id SERIAL PRIMARY KEY,
    add_no VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    duration VARCHAR(50) NOT NULL,
    schedule_type VARCHAR(50) NOT NULL,
    lounge_groups TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL,
    version VARCHAR(20) DEFAULT 'v1.0',
    status VARCHAR(20) DEFAULT 'Active',
    description TEXT,
    file_name VARCHAR(255),
    file_path VARCHAR(500),
    file_type VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS advertisement_schedules (
    id SERIAL PRIMARY KEY,
    advertisement_id INTEGER NOT NULL REFERENCES advertisements(id) ON DELETE CASCADE,
    schedule_type VARCHAR(50) NOT NULL,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP,
    no_end_date BOOLEAN DEFAULT FALSE,
    occurs_type VARCHAR(20),
    occurs_once_time VARCHAR(10),
    starting_time VARCHAR(10),
    ending_time VARCHAR(10),
    weekdays TEXT,
    monthly_day INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS advertisement_groups (
    id SERIAL PRIMARY KEY,
    group_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    lounges TEXT NOT NULL,
    number_of_adds INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS lounges (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    location VARCHAR(255),
    status VARCHAR(20) DEFAULT 'Active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_advertisements_category ON advertisements(category);
CREATE INDEX idx_advertisements_status ON advertisements(status);
CREATE INDEX idx_advertisements_priority ON advertisements(priority);
CREATE INDEX idx_advertisement_schedules_ad_id ON advertisement_schedules(advertisement_id);
CREATE INDEX idx_advertisement_groups_group_id ON advertisement_groups(group_id);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_advertisements_updated_at BEFORE UPDATE ON advertisements
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_advertisement_schedules_updated_at BEFORE UPDATE ON advertisement_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_advertisement_groups_updated_at BEFORE UPDATE ON advertisement_groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_lounges_updated_at BEFORE UPDATE ON lounges
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
