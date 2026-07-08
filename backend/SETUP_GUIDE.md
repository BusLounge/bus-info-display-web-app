# Backend Setup & Installation Guide

## ✅ Installation Status: COMPLETE

All dependencies have been successfully installed:
- ✅ gorilla/mux v1.8.1 (HTTP Router)
- ✅ joho/godotenv v1.5.1 (Environment Config)
- ✅ lib/pq v1.11.1 (PostgreSQL Driver)

## 📋 Next Steps

### Step 1: Install & Setup PostgreSQL

1. **Download PostgreSQL**
   - Go to: https://www.postgresql.org/download/windows/
   - Download PostgreSQL 16 or 15
   - Run installer and remember your password

2. **Create Database**
   Open Command Prompt or PowerShell and run:
   ```bash
   psql -U postgres
   ```
   Enter your PostgreSQL password, then:
   ```sql
   CREATE DATABASE bus_schedule_lounge;
   \q
   ```

### Step 2: Run Database Migrations

Navigate to backend folder and run:
```bash
cd D:\Bus_Schedule_Lounge-dashboard\backend
psql -U postgres -d bus_schedule_lounge -f migrations\001_create_advertisements_tables.up.sql
```

### Step 3: Configure Database Connection

The `.env` file is already created with default settings:
```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres   ← Change this to your PostgreSQL password
DB_NAME=bus_schedule_lounge
DB_SSL_MODE=disable
SERVER_PORT=8080
```

**IMPORTANT:** Edit `.env` and change `DB_PASSWORD` to your actual PostgreSQL password!

### Step 4: Run the Backend Server

**Option A: Run directly with Go**
```bash
cd D:\Bus_Schedule_Lounge-dashboard\backend
go run cmd/api/main.go
```

**Option B: Build and run executable**
```bash
cd D:\Bus_Schedule_Lounge-dashboard\backend
go build -o bin/server.exe cmd/api/main.go
.\bin\server.exe
```

### Step 5: Test the Server

Open browser and visit:
- Health Check: http://localhost:8080/health
- Get Advertisements: http://localhost:8080/api/advertisements
- Get Groups: http://localhost:8080/api/advertisement-groups

## 🚀 API Endpoints

### Advertisements
- `GET    /api/advertisements` - Get all advertisements
- `POST   /api/advertisements` - Create new advertisement
- `GET    /api/advertisements/{id}` - Get advertisement by ID
- `PUT    /api/advertisements/{id}` - Update advertisement
- `DELETE /api/advertisements/{id}` - Delete advertisement

### Advertisement Groups
- `GET    /api/advertisement-groups` - Get all groups
- `POST   /api/advertisement-groups` - Create new group
- `GET    /api/advertisement-groups/{id}` - Get group by ID
- `PUT    /api/advertisement-groups/{id}` - Update group
- `DELETE /api/advertisement-groups/{id}` - Delete group

## 🧪 Testing the API

### Create Advertisement Group (using PowerShell)
```powershell
$body = @{
    name = "Lounge Group A"
    lounges = @("Lounge 1", "Lounge 2", "Lounge 3")
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/advertisement-groups" -Method POST -Body $body -ContentType "application/json"
```

### Get All Groups
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/advertisement-groups" -Method GET
```

## 📁 Project Structure

```
backend/
├── cmd/api/main.go                    ← Main entry point
├── internal/
│   ├── config/config.go              ← Configuration
│   ├── database/database.go          ← Database connection
│   ├── handlers/                     ← HTTP handlers
│   ├── middleware/                   ← CORS, Logging
│   ├── models/advertisement.go       ← Data models
│   ├── repository/                   ← Database operations
│   └── services/                     ← Business logic
├── migrations/                        ← Database schemas
├── .env                              ← Environment config
├── go.mod                            ← Go dependencies
└── README.md                         ← Documentation
```

## 🔧 Troubleshooting

### "Failed to connect to database"
- Make sure PostgreSQL is running
- Check your password in `.env` file
- Verify database exists: `psql -U postgres -l`

### "Port already in use"
- Change `SERVER_PORT` in `.env` to another port (e.g., 8081)

### Build errors
- Run: `go mod tidy`
- Then: `go build cmd/api/main.go`

## ✨ Success Indicators

When server starts successfully, you'll see:
```
Successfully connected to database
Server starting on port 8080
```

Now your backend is ready to connect with the Angular frontend!
