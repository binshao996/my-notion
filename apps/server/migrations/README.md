# Database Migrations

## Development

For local development, GORM's `AutoMigrate` is used automatically in `pkg/db/db.go` when the server starts. This creates and updates tables based on the Go struct definitions without needing to run manual SQL migrations.

## Production

In production, use the SQL migration files in this directory for controlled, versioned schema changes:

- `001_init.sql` -- Initial schema: users, workspaces, workspace_members
- `002_pages_blocks.sql` -- Pages and blocks tables

### Running migrations

```sh
psql $DATABASE_URL -f migrations/001_init.sql
psql $DATABASE_URL -f migrations/002_pages_blocks.sql
```

## Adding new migrations

1. Create a new file named `NNN_description.sql` (increment the number)
2. Write idempotent SQL (use `CREATE TABLE IF NOT EXISTS`, etc.)
3. Test against a staging database before applying to production
