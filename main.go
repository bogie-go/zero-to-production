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
	"syscall"

	"ecommerce/app"
)

const usage = `Usage: ecommerce [command]

Commands:
  serve               the HTTP server (default)
  migrate <command>   inspect or move the embedded schema
  db <command>        create, drop or prepare the database itself

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
