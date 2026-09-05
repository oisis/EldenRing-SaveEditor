# SetSaveLifecycleSettings

Atomically updates the host-local automatic-backup policy: the retention count
and the backup name pattern.

| | |
|---|---|
| EndpointID | `set_save_lifecycle_settings` |
| Kind | Mutation of application settings; no save revision commit |
| Status | implemented; Wails `SetSaveLifecycleSettings` |
| Source | [set_save_lifecycle_settings.go](../../../backend/endpoints/savesession/set_save_lifecycle_settings.go) |

Input is an integer `backupRetention` in the supported range `1..1000` and a
string `backupNamePattern`. The private settings file is written atomically with
owner-only permissions. A failure restores the previous in-memory value.

## The backup name pattern

The pattern is validated here and nowhere else: the backend is the single source
of the grammar, and the frontend only carries the value.

- exactly two tokens exist, `{filename}` and `{timestamp}`, and each must appear
  exactly once;
- `{filename}` expands to the name of the file being replaced, with its
  extension and without any directory part;
- `{timestamp}` expands to the UTC creation time as `YYYYMMDDHHmmSS`, the format
  2.0 has always written;
- safe literal text is allowed around them;
- an empty pattern means the default, `{filename}.{timestamp}`;
- the backend appends `_bak`, and a collision in the same second takes a counter
  before that suffix, as in `…_2_bak`.

Rejected: an unknown token, either token missing or repeated, a path separator,
a control character, an attempt to leave the directory, a leading dot, a name
that ends in a space or a dot, a name reserved on Windows, and any pattern whose
expansion would break one of those rules.

Changing the pattern renames no existing backup and does not make one
unreadable. Retention keeps acting on backups this application recorded as its
own and on names the fixed 2.0 grammar produced; a file that is neither is never
removed, whatever pattern is configured.

The same setting and the same grammar name the deployment target backups, which
read it from here rather than keeping a second copy of it.
