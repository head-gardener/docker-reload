# `docker-reload`

Reload docker containers on file changes (see `example/compose.yml`). Use cases
include secret/certificate rotation for applications that don't automatically
reload on file changes. **Only supports docker compose deployments.**

This works by listening to `inotify` for file changes and comparing salted
hashes. Memory footprint is about 5-10MB per watcher.

## Example configuration

```yaml
watchers:
  - paths:
      - file: /secrets/grafana/password
      - file: /public/grafana/username
    selector:
      label: watch-category=grafana
    action: restart
    hash: sha256
  - paths:
      - dir: /certs
        globs: ['postgres*']
    selector:
      label: watch-category=postgres-certs
    action: sighup
    hash: sha1
```

## Option reference

### Command line flags

- `-config`: config file path.
- `-log-level`: log level (`debug`, `info`, etc).

### Configuration options

- `watchers`: watchers configuration.
    - `paths`: defines files that need to be watched for changes. Can be `dir`
      and `glob` or `file` (which expands to `dir` and `glob`).
    - `selector`: defines containers that need to be reloaded. Can be `label`
      or `name`.
    - `action`: action to perform on file change. Can be `restart` or `sighup`.
    - `hash`: hash function to use. Can be `sha1` or `sha256`.
- `default`: fields that should be set on watchers by default. Same fields as a
  watcher

## Caveats

- **Do not listen for files directly in `/`**
- NFS and some other similar filesystems possibly don't work with `inotify`.

## Missing features

PR's are welcome!

These features are **not planned** but can be added if changes are important and
not too complicated: 

- Docker client configuration
- Rolling updates
- Support for Swarm (or any other compositor) 
- Notifications
- Optimizations

I might implement these features some time in the future:

- Polling for changes as an alternative to `inotify` for NFS, etc.
- More hash functions, actions, selectors, etc.
