// Example 02-config-file demonstrates loading configuration from a YAML file.
// Run from its directory: cd examples/02-config-file && go run .
package main

import (
	"fmt"
	"log"

	"github.com/nniel-ape/gonfig"
)

type Config struct {
	Server struct {
		Host string `default:"localhost" description:"server bind address"`
		Port int    `default:"8080"      description:"server port"`
	}
	Database struct {
		Host     string `default:"localhost" description:"database host"`
		Port     int    `default:"5432"      description:"database port"`
		Name     string `default:"mydb"      description:"database name"`
		User     string `default:"postgres"  description:"database user"`
		Password string `default:""          description:"database password"`
	}
	LogLevel string `default:"info" description:"logging level"`
}

func main() {
	var cfg Config
	if err := gonfig.Load(&cfg, gonfig.WithFile("config.yaml")); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Server:")
	fmt.Printf("  Host: %s\n", cfg.Server.Host)
	fmt.Printf("  Port: %d\n", cfg.Server.Port)
	fmt.Println("Database:")
	fmt.Printf("  Host: %s\n", cfg.Database.Host)
	fmt.Printf("  Port: %d\n", cfg.Database.Port)
	fmt.Printf("  Name: %s\n", cfg.Database.Name)
	fmt.Printf("  User: %s\n", cfg.Database.User)
	fmt.Println("LogLevel:", cfg.LogLevel)
}
