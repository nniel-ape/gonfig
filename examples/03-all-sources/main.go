// Example 03-all-sources demonstrates the full gonfig pipeline:
// defaults → config file → environment variables → command-line flags.
//
// Try different combinations:
//
//	cd examples/03-all-sources && go run .
//	cd examples/03-all-sources && APP_LOG_LEVEL=warn go run .
//	cd examples/03-all-sources && go run . --server-port 9090
//	cd examples/03-all-sources && go run . -p 9090
//	cd examples/03-all-sources && go run . --help
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
	Database struct {
		Host string `default:"localhost" description:"database host"`
		Port int    `default:"5432"      description:"database port" validate:"min=1,max=65535"`
		Name string `default:"mydb"      description:"database name" validate:"required"`
	}
	LogLevel string `default:"info" description:"logging level" validate:"oneof=debug info warn error" short:"l"`
}

func main() {
	var cfg Config

	err := gonfig.Load(&cfg,
		gonfig.WithFile("config.yaml"),
		gonfig.WithEnvPrefix("APP"),
		gonfig.WithFlags(os.Args[1:]),
	)

	// Handle --help: print usage and exit.
	if errors.Is(err, flag.ErrHelp) {
		fmt.Print(gonfig.Usage(&cfg, gonfig.WithEnvPrefix("APP")))
		os.Exit(0)
	}

	// Handle validation errors.
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

	fmt.Println("Configuration loaded successfully!")
	fmt.Printf("Server:   %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Database: %s:%d/%s\n", cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	fmt.Printf("LogLevel: %s\n", cfg.LogLevel)
}
