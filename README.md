# GoGoGo API Server

A learning project to practice building HTTP API servers in Go.

## Tech Stack

- **Air** - Hot reload during development
- **Mux Router** - HTTP request routing
- **GORM** - Database ORM handling

## Project Structure

```
.
├── app/              # Application code and features
├── db/               # Database functions and models
├── static/           # Static resources
└── generated/        # GORM-generated files
```

## Prerequisites

- Go 1.19 or higher
- Air (for hot reload)

## Installation

```bash
# Install dependencies
go mod download

# Install Air for hot reload
go install github.com/cosmtrek/air@latest
```

## Running Locally

### Development (with hot reload)
```bash
air
```

### Production Build
```bash
go build -o server
./server
```

## API Documentation

Add your endpoints and usage examples here.

---

**Note:** This is a learning project for practicing Go development patterns.