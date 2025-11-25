# Quick Start Guide - AI-KMS

## Prerequisites

You need PostgreSQL with the `pgvector` extension.

## Option 1: Docker (Recommended)

1. **Start Docker Desktop**

2. **Start all services:**
   ```bash
   make docker-up
   ```
   This starts PostgreSQL, Jaeger, and the app.

3. **Access the app:**
   - Backend: http://localhost:8080
   - Jaeger UI: http://localhost:16686

4. **Stop services:**
   ```bash
   make docker-down
   ```

## Option 2: Local PostgreSQL

If you have PostgreSQL installed locally:

```bash
# Create database
createdb ai_kms

# Connect and enable pgvector
psql ai_kms -c "CREATE EXTENSION vector;"

# Start the app
make run
```

## Frontend

In a separate terminal:

```bash
cd frontend
npm install
npm run dev
```

Access at: http://localhost:5173

## Troubleshooting

**"OPENAI_API_KEY is required"**
- Copy `.env.example` to `.env`
- Add your actual OpenAI API key

**"database does not exist"**
- Use Docker: `make docker-up`
- Or create manually: `createdb ai_kms`

**Docker not running**
- Start Docker Desktop
- Verify: `docker ps`
