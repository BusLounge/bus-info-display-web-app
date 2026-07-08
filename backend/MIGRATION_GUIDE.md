# Advertisement Groups Migration Guide

## Problem
The `advertisement_groups` table doesn't exist in your Supabase database yet.

## Solution
Run the migration SQL to create the table.

### Option 1: Using Supabase Dashboard (Recommended)

1. Go to your Supabase Dashboard: https://supabase.com/dashboard
2. Select your project
3. Click on **SQL Editor** in the left sidebar
4. Click **New Query**
5. Copy the contents of `run_migration.sql` and paste it into the editor
6. Click **Run** or press `Ctrl+Enter`

### Option 2: Using psql Command Line

```bash
# Navigate to the backend directory
cd backend

# Run the migration (replace with your actual database URL)
psql "postgresql://postgres.pttatcukzpceljcrwehk:KQ95tJUYdFX251VR@aws-1-us-east-1.pooler.supabase.com:6543/postgres" -f run_migration.sql
```

### Option 3: Quick SQL to run in Supabase SQL Editor

```sql
CREATE TABLE IF NOT EXISTS public.advertisement_groups (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_name           VARCHAR(255) NOT NULL UNIQUE,
    lounges              TEXT,
    no_of_advertisements INT DEFAULT 0,
    created_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Verification

After running the migration, test the API:

1. Make sure the backend server is running: `go run .\cmd\server\main.go`
2. Try creating a new advertisement group from the frontend
3. Check the browser console and backend logs for any errors

## Expected Result

You should see:
- No errors in the backend logs
- Group created successfully
- Group appears in the advertisement groups table
