# Bus Schedule Lounge - Backend API

Backend server for Advertisement Management System built with Go.

## Quick Start

### 1. Install PostgreSQL
Download and install PostgreSQL from https://www.postgresql.org/download/

### 2. Create Database
```sql
CREATE DATABASE bus_schedule_lounge;
```

### 3. Run Migrations
```bash
psql -U postgres -d bus_schedule_lounge -f migrations/001_create_advertisements_tables.up.sql
```

### 4. Configure Environment
Edit `.env` file with your database credentials

### 5. Run Server
```bash
go run cmd/api/main.go
```

Server will start on http://localhost:8080

## API Endpoints

### Advertisements
- `GET /api/advertisements` - Get all
- `POST /api/advertisements` - Create
- `GET /api/advertisements/{id}` - Get by ID
- `PUT /api/advertisements/{id}` - Update
- `DELETE /api/advertisements/{id}` - Delete

### Groups
- `GET /api/advertisement-groups` - Get all
- `POST /api/advertisement-groups` - Create
- `GET /api/advertisement-groups/{id}` - Get by ID
- `PUT /api/advertisement-groups/{id}` - Update
- `DELETE /api/advertisement-groups/{id}` - Delete

### Health
- `GET /health` - Health check
