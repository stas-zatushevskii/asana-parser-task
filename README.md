# Asana Extractor

Small Go service that periodically exports Asana users and projects to JSON files.

## What It Does

- Calls `GET https://app.asana.com/api/1.0/users`
- Calls `GET https://app.asana.com/api/1.0/projects`
- Follows Asana pagination with `limit` and `next_page.offset`
- Uses Bearer token authorization with an Asana personal access token
- Retries `429` and `5xx` responses
- Respects `Retry-After` on `429`
- Writes one JSON file per exported object:

```text
output/
├── users/
│   └── user_<gid>.json
└── projects/
    └── project_<gid>.json
```

## Configure

Copy the example config and put your personal access token into `http.access_token`:

```sh
make config
```

Allowed extraction intervals are `5m` and `30s`.

```yaml
http:
  access_token: "put_personal_access_token_here"
  request_per_minute: 150
  retry_after: 30
  retry_attempts: 3

extractor:
  interval: "5m"
  output_dir: "output"
  page_limit: 100
```

If numeric values are empty or zero, the app uses defaults. `page_limit` is capped at `100`. The access token is required.

## Run

```sh
make run
```

Build:

```sh
make build
```

Test:

```sh
make test
```

Docker:

```sh
make docker-run
```

Docker mounts the project `./output` folder into `/app/output`, so exported files are written under `./output/users` and `./output/projects` on the host.
`make run` and `make docker-run` require `config/config.yaml` to exist and to contain a real token instead of the placeholder.

Direct Docker run:

```sh
mkdir -p output
docker run --rm \
  -v "$PWD/config/config.yaml:/app/config/config.yaml:ro" \
  -v "$PWD/output:/app/output" \
  asana-extractor:latest
```

If the config is not mounted, the container prints a setup hint and exits before the application starts.

Useful make targets:

```sh
make config
make run
make docker-build
make docker-run-foreground
make docker-stop
make docker-logs
```

## Improvements

- Add a shared cache for extracted Asana data.
  When multiple service instances run at the same time, they should first check a shared Redis cache with TTL before calling Asana. If another instance has already fetched fresh users or projects, the current instance should reuse those cached results instead of duplicating API calls.

- Add cache freshness and invalidation rules.
  Define separate cache keys for users and projects, include the request scope in the key, and make TTL configurable so data freshness can be tuned without changing code.

- Improve visibility of extraction runs.
  Log page counts, cache hits/misses, extracted object counts, retry attempts, and final output path so operational issues are easier to diagnose.

- Protect concurrent output writes.
  If multiple instances write to the same output file or shared volume, introduce a locking or leader-writer strategy so one instance cannot overwrite another instance's fresh export unexpectedly.

References:

- [Asana pagination](https://developers.asana.com/jd/docs/pagination)
- [Get multiple users](https://developers.asana.com/pt/reference/getusers)
- [Get multiple projects](https://developers.asana.com/reference/getprojects)
