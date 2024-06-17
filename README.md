# awfi - Another Wait-For-It Tool

`awfi` is a simple tool to wait for a resource to become available. It supports
both HTTP and Postgres resources. Requests are retried every second until the
resource becomes available or the timeout is reached. The default timeout is 10
seconds.

For HTTP/HTTPS resources, the tool will wait for a 200 status code. For Postgres
resources, the tool will wait for a successful connection and success when executing
the query "SELECT 1".

## Installation

Currently, the only supported installation method is using `go install`:
```bash
go install github.com/parrotmac/awfi@latest
```

## Usage

```bash
awfi [flags] <resource>
```

### Flags

- `-t, --timeout`: The timeout in seconds. Default is 10 seconds.

### Examples

Wait for a local Postgres database to become available:
```bash
awfi postgres://user:password@localhost:5432/dbname
```

Wait for a remote HTTP resource to become available:
```bash
awfi https://example.com
```

## Why Another Wait-For-It Tool?

While building out CI/CD pipelines, I found myself needing a simple tool to wait
for resources to become available. In particular, when waiting for a Postgres
database to become available, I would do copy-paste something like the following
snippet:

```shell
did_start=false
# shellcheck disable=SC2034
for i in {1..30}; do
  if pg_isready -U "$username" -h "$host" -p "$port" -d "$dbname" -t 1; then
    did_start=true
    break
  fi
  printf "."
  sleep 1
done
printf "\n"
if ! $did_start; then
  echo "Failed to start database"
  exit 1
fi
```

However, in actual usage I'd find myself tweaking the script to add more features
or cover the seemingly common edge case of the database being 'ready' while still
being unable to accept connections:
```shell
 Network cool_app_default  Creating
 Network cool_app_default  Created
 Container cool-app-db-1  Creating
 Container cool-app-db-1  Created
 Container cool-app-db-1  Starting
 Container cool-app-db-1  Started
/var/run/postgresql:5432 - no response
/var/run/postgresql:5432 - accepting connections
psql: error: connection to server at "localhost" (::1), port 5432 failed: server closed the connection unexpectedly
	This probably means the server terminated abnormally
	before or while processing the request.
```

Since I found myself tweaking the script and still running into issues, I decided
to write a simple tool that would handle the most common cases and edge cases
for me.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
