# Manual Migration Snippets

This folder contains helper SQL snippets to align existing databases with the new schema expectations used by `pkg/simplecontent`.

- `000_create_schema.sql` — provision the target schema prior to running goose migrations.
- `001_rename_tables_and_columns.sql` — align legacy tables/columns with the current layout.

Guidance:
- Use these only for existing deployments. New greenfield DBs should use `migrations/postgres` after the schema exists.
- Always back up your database before applying.
- Adjust schema qualifiers if you are not using the `content` schema.
- Configure your SQL session or connection string to set `search_path` to the target schema before running the snippets or goose migrations.
