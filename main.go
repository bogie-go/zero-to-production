// Command ecommerce is the Go service next to the Rails app.
//
// ONE BINARY, several roles, selected by the first argument: serving,
// migrating the schema, creating the database. They share a module and
// therefore a schema, a config loader and an artifact, so a deploy cannot
// ship a web container built from one commit and a migration from another.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"

	"ecommerce/app"
)

const usage = `Usage: ecommerce [command]

Commands:
  serve               the HTTP server (default)
  worker              work the job queue
  migrate <command>   inspect or move the embedded schema
  db <command>        create, drop or prepare the database itself
  routes              every mounted route, method and path (rails routes)
  mail test ADDRESS   send a test email through the configured SMTP server

With no command this serves, which is what the container does. Every command
reads the same ECOMMERCE_ENV, so they all act on the same database.
`

func main() {
	// main is the only place that exits. Everything below returns errors.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	command, args := "serve", []string{}
	if len(os.Args) > 1 {
		command, args = os.Args[1], os.Args[2:]
	}

	// `ecommerce serve --help` asks a question; answer it instead of booting.
	for _, a := range args {
		if a == "-h" || a == "--help" {
			fmt.Print(usage)
			return
		}
	}

	// One entry per role or task. A generator that adds a command (bogie g
	// authentication api_key) registers it above the marker.
	commands := map[string]func(context.Context, []string) error{
		"serve":   runServe,
		"migrate": runMigrate,
		"db":      runDB,
		"routes":  runRoutes,
		"mail":    runMail,
		"worker":  runWorker,
		// bogie:commands
	}
	if command == "help" || command == "-h" || command == "--help" {
		fmt.Print(usage)
		return
	}
	run, ok := commands[command]
	if !ok {
		// NOT an error. The container may be started with flags this binary does
		// not know, and refusing to boot over an unrecognised argument is a worse
		// failure than ignoring it. But say so, so a typo is visible in the log.
		fmt.Fprintf(os.Stderr, "ecommerce: unknown command %q; serving instead\n\n%s\n", command, usage)
		run = runServe
	}

	if err := run(ctx, args); err != nil {
		fmt.Fprintln(os.Stderr, "ecommerce:", err)
		os.Exit(1)
	}
}

// runServe is the HTTP server.
func runServe(ctx context.Context, _ []string) error {
	application, err := app.NewApplication(ctx)
	if err != nil {
		return err
	}
	return application.Start(ctx)
}

// runRoutes prints every mounted route, as `rails routes` does, by pointing
// gin's own debug route print at the format Rails uses rather than fetching
// Router.Routes() and writing a second lister: this fires from inside the
// same SetupRoutes serve runs, so the list is what is actually wired, in the
// order it was wired, not a guess read off the templates. Like gin's own
// print, it only fires in development: ecommerce routes in test or
// production prints nothing, the same trade `NewServer` already makes.
func runRoutes(ctx context.Context, _ []string) error {
	gin.DebugPrintRouteFunc = func(method, path, handler string, _ int) {
		handler = strings.TrimSuffix(strings.TrimPrefix(handler, "ecommerce/"), "-fm")
		fmt.Printf("%-7s  %-30s  %s\n", method, path, handler)
	}
	_, err := app.NewApplication(ctx)
	return err
}

// runWorker works the job queue.
//
// Its own process rather than a goroutine in the server: a backlog of jobs
// must never slow a request, and the two scale on different signals. Same
// binary, because they share a schema and must never disagree about it.
func runWorker(ctx context.Context, _ []string) error {
	application, err := app.NewApplication(ctx)
	if err != nil {
		return err
	}
	return application.StartWorker(ctx)
}
