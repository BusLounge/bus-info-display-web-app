-- Migration: Create advertisement_groups table
-- Date: 2026-02-13
-- Purpose: Create the advertisement_groups table for storing lounge groups

-- Drop table if exists (optional - remove this line if you want to keep existing data)
-- DROP TABLE IF EXISTS public.advertisement_groups;

-- Create advertisement_groups table
CREATE TABLE IF NOT EXISTS public.advertisement_groups (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_name           VARCHAR(255) NOT NULL UNIQUE,
    lounges              TEXT,
    no_of_advertisements INT DEFAULT 0,
    created_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Verify table was created
SELECT 'Table created successfully!' as status;
SELECT * FROM public.advertisement_groups LIMIT 5;
