# TaskFlow

TaskFlow is a modern project and task management application built with a high-performance Go backend and a responsive React frontend. It provides a robust platform for teams to organize workspaces, manage projects, and track tasks efficiently.

## Technology Stack

### Backend
- **Language:** Go 1.25+
- **Router:** Chi v5
- **Database:** PostgreSQL with pgx/v5
- **SQL Generator:** sqlc (type-safe SQL)
- **Authentication:** JWT (JSON Web Tokens)
- **Validation:** go-playground/validator
- **Email:** Resend API integration

### Frontend
- **Framework:** React 19 (TypeScript)
- **Build Tool:** Vite 8
- **Package Manager:** Bun
- **Linting:** ESLint & TypeScript-ESLint

### Infrastructure & Tools
- **Task Runner:** Taskfile (go-task)
- **Monorepo Management:** Bun Workspaces / Turbo
- **Database Migrations:** golang-migrate

## Features

- **User Authentication:** Secure registration, login, and session management using JWT and refresh tokens. Includes password reset functionality via email.
- **Workspace Management:** Create and manage multiple workspaces for different teams or organizations.
- **Project & Task Tracking:** Organize work into projects and detailed tasks with support for labels, comments, and attachments (database schema implemented).
- **Activity Logging:** Comprehensive audit trails for actions within the system.
- **Role-Based Access:** Infrastructure for managing workspace and project memberships.

## Project Structure

```text
.
├── apps/
│   ├── api/            # Go backend application
│   │   ├── cmd/        # Application entry points
│   │   ├── db/         # SQL queries, migrations, and generated code
│   │   ├── internal/   # Core logic (auth, workspace, middleware, etc.)
│   │   └── bin/        # Compiled binaries
│   └── web/            # React frontend application
│       ├── src/        # Frontend source code
│       └── public/     # Static assets
├── Taskfile.yml        # Project-wide automation tasks
└── package.json        # Root workspace configuration
```

## Getting Started

### Prerequisites

- Go 1.25 or higher
- Bun (JavaScript runtime and package manager)
- PostgreSQL
- Taskfile (`go-task`)
- `migrate` CLI (for database migrations)
- `sqlc` (optional, for code generation)

### Installation

1. Clone the repository.
2. Run the setup task to install all dependencies:
   ```bash
   task setup
   ```

### Configuration

Create a `.env` file in the `apps/api/` directory with the following variables:

```env
APP_ENV=development
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/taskflow?sslmode=disable
JWT_SECRET=your_minimum_32_character_secret_here
RESEND_API_KEY=re_your_api_key
FROM_EMAIL=onboarding@resend.dev
```

### Database Setup

Run the migrations to create the database schema:
```bash
task db:migrate
```

### Running the Application

Start the backend API and frontend development server:

**Backend:**
```bash
task api:run
```

**Frontend:**
```bash
task web:dev
```

## Available Tasks

The project uses `Taskfile.yml` for common operations. You can list all tasks using `task --list`.

### General
- `task setup`: Install all dependencies and tidy Go modules.
- `task build`: Build both the web and API applications.
- `task ci`: Run all checks (linting, type-checking, tests, and builds).

### API
- `task api:run`: Run the API server in development mode.
- `task api:test`: Run Go tests.
- `task api:lint`: Run golangci-lint.
- `task api:fmt`: Format Go code.

### Web
- `task web:dev`: Run the frontend development server.
- `task web:lint`: Lint the React codebase.
- `task web:typecheck`: Run TypeScript type-checking.

### Database
- `task db:migrate`: Apply all up migrations.
- `task db:rollback`: Roll back the last migration.
- `task db:generate`: Generate Go code from SQL queries using sqlc.
- `task db:new <name>`: Create a new SQL migration file.
