package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

var (
	timeout           = flag.Int("timeout", 10, "Timeout in seconds for waiting for resource")
	repeatedSuccesses = flag.Int("repeated-successes", 1, "Number of repeated successes before considering the resource available")

	usageText = `awfi: A[nother] W[ait] F[or] I[t] tool

awfi is a simple tool to wait for a resource to become available. It supports
both HTTP and Postgres resources. Requests are retried every second until the
resource becomes available or the timeout is reached. The default timeout is 10
seconds.

For HTTP/HTTPS resources, the tool will wait for a 200 status code. For Postgres
resources, the tool will wait for a successful connection and success when executing
the query "SELECT 1".

Usage:
	awfi [flags] <resource>

Examples:
	# Wait for an HTTP resource
	awfi http://example.com

	# Wait for a Postgres resource
	awfi postgres://user:password@localhost:5432/dbname

	# Wait for a Postgres resource with a custom timeout
	awfi --timeout=30 postgres://user:password@localhost:5432/dbname

Flags:` // flag.Usage() will print the flags
)

func isHttpResource(resource string) bool {
	return strings.HasPrefix(resource, "http://") || strings.HasPrefix(resource, "https://")
}

func isPostgresResource(resource string) bool {
	return strings.HasPrefix(resource, "postgres://") || strings.HasPrefix(resource, "postgresql://")
}

func checkPostgresResource(ctx context.Context, resource string) error {
	cappedCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(*timeout))
	defer cancel()

	pgConn, err := pgx.Connect(cappedCtx, resource)
	if err != nil {
		return errors.Wrap(err, "failed to connect to postgres")
	}

	defer func() {
		_ = pgConn.Close(cappedCtx)
	}()

	var one int
	err = pgConn.QueryRow(cappedCtx, "SELECT 1").Scan(&one)
	if err != nil {
		return errors.Wrap(err, "failed to query postgres")
	}

	return nil
}

func checkHttpResource(ctx context.Context, resource string) error {
	cappedCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(*timeout))
	defer cancel()

	cx := &http.Client{
		Timeout: time.Second * time.Duration(*timeout),
	}

	req, err := http.NewRequestWithContext(cappedCtx, "GET", resource, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	resp, err := cx.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to perform request")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return errors.New("non-200 status code")
	}

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response body")
	}

	return nil
}

func waitForHttpResource(ctx context.Context, resource string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			if err := checkHttpResource(ctx, resource); err == nil {
				return nil
			}
		}
	}
}

type ResourceChecker interface {
	Check(ctx context.Context) error
}

type PostgresChecker struct {
	ConnString string
}

var _ ResourceChecker = (*PostgresChecker)(nil)

func (p *PostgresChecker) Check(ctx context.Context) error {
	return checkPostgresResource(ctx, p.ConnString)
}

type HttpChecker struct {
	Resource string
}

var _ ResourceChecker = (*HttpChecker)(nil)

func (h *HttpChecker) Check(ctx context.Context) error {
	return checkHttpResource(ctx, h.Resource)
}

func waitForResource(ctx context.Context, checker ResourceChecker, successThreshold int) error {
	successes := 0
	var err error
	for {
		select {
		case <-ctx.Done():
			return err
		case <-time.After(time.Second):
			if err = checker.Check(ctx); err == nil {
				successes++
				if successes >= successThreshold {
					return nil
				}
			} else {
				successes = 0
			}
		}
	}
}

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "%s\n", usageText)
		flag.PrintDefaults()
	}
	flag.Parse()

	var resource string
	if flag.NArg() > 0 {
		resource = flag.Arg(0)
	} else {
		fmt.Println("Resource is required")
		flag.Usage()
		return
	}

	timeoutDuration := time.Second * time.Duration(*timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	if isHttpResource(resource) {
		httpChecker := &HttpChecker{Resource: resource}
		err := waitForResource(ctx, httpChecker, *repeatedSuccesses)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else if isPostgresResource(resource) {
		pgChecker := &PostgresChecker{ConnString: resource}
		err := waitForResource(ctx, pgChecker, *repeatedSuccesses)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		fmt.Printf("Unsupported resource type: %s\n", resource)
		flag.Usage()
	}
}
