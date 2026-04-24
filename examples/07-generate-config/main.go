// Example 07-generate-config demonstrates generating config file skeletons
// from struct definitions using gonfig.Example().
//
// Run with:
//
//	go run .                          # load defaults
//	go run . --generate-config yaml   # print YAML skeleton
//	go run . --generate-config json   # print JSON skeleton
//	go run . --generate-config toml   # print TOML skeleton
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/nniel-ape/gonfig"
)

type Config struct {
	Server struct {
		Host    string        `default:"0.0.0.0" description:"server bind address" validate:"required"`
		Port    int           `default:"8080"    description:"server port" validate:"min=1,max=65535"`
		Timeout time.Duration `default:"30s"     description:"request timeout"`
	}
	DB struct {
		Host     string `default:"localhost" description:"database host" validate:"required"`
		Port     int    `default:"5432"      description:"database port" validate:"min=1,max=65535"`
		Name     string `default:"myapp"     description:"database name" validate:"required"`
		Password string `default:""          description:"database password"`
	}
	LogLevel string   `default:"info"  description:"log level" validate:"oneof=debug info warn error"`
	Debug    bool     `default:"false" description:"debug mode"`
	Tags     []string `default:"web,api" description:"application tags"`
}

func main() {
	var cfg Config

	err := gonfig.Load(&cfg,
		gonfig.WithEnvPrefix("APP"),
		gonfig.WithFlags(os.Args[1:]),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Printf("Server: %s:%d (timeout: %s)\n", cfg.Server.Host, cfg.Server.Port, cfg.Server.Timeout)
	fmt.Printf("DB: %s:%d/%s\n", cfg.DB.Host, cfg.DB.Port, cfg.DB.Name)
}
