# Manual Migration Snippets

This folder contains helper SQL snippets to align existing databases with the new schema expectations used by `pkg/simplecontent`.

- Use these only for existing deployments. New greenfield DBs should use `migrations/postgres`.
- Always back up your database before applying.
- Adjust schema qualifiers if you are not using the `content` schema.
