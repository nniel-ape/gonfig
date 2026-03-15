// Example 03-all-sources demonstrates the full gonfig pipeline:
// defaults → config file → environment variables → command-line flags.
//
// --help is handled automatically by gonfig (prints usage and exits).
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
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Configuration loaded successfully!")
	fmt.Printf("Server:   %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Database: %s:%d/%s\n", cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	fmt.Printf("LogLevel: %s\n", cfg.LogLevel)
}
