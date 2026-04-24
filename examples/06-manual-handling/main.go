// Example 06-manual-handling demonstrates manual handling of --help and
// validation errors instead of relying on gonfig's defaults.
//
// Use WithAutoHelp(false) to receive flag.ErrHelp from Load, then print
// custom help text. Use errors.As to inspect individual validation failures.
//
//	cd examples/06-manual-handling && go run .
//	cd examples/06-manual-handling && go run . --help
//	cd examples/06-manual-handling && go run . --server-port 0
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nniel-ape/gonfig"
)

type Config struct {
	Server struct {
		Host string `default:"localhost" description:"server bind address" short:"H"`
		Port int    `default:"8080"      description:"server port" validate:"min=1,max=65535" short:"p"`
	}
	LogLevel string `default:"info" description:"logging level" validate:"oneof=debug info warn error" short:"l"`
}

func main() {
	var cfg Config

	err := gonfig.Load(&cfg,
		gonfig.WithEnvPrefix("APP"),
		gonfig.WithFlags(os.Args[1:]),
		gonfig.WithAutoHelp(false), // disable auto-help to handle it ourselves
	)

	// Handle --help manually: print a custom banner + usage.
	if errors.Is(err, flag.ErrHelp) {
		fmt.Println("My Application v1.0.0")
		fmt.Println()
		fmt.Println("Usage: myapp [flags]")
		fmt.Println()
		fmt.Print(gonfig.Usage(&cfg, gonfig.WithEnvPrefix("APP")))
		os.Exit(0)
	}

	// Handle validation errors: inspect each field failure individually.
	var ve *gonfig.ValidationError
	if errors.As(err, &ve) {
		fmt.Fprintln(os.Stderr, "Configuration errors:")

		for _, fe := range ve.Errors {
			fmt.Fprintf(os.Stderr, "  - %s: %s\n", fe.Field, fe.Message)
		}

		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("LogLevel: %s\n", cfg.LogLevel)
}
