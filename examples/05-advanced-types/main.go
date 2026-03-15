// Example 05-advanced-types demonstrates gonfig with slices, maps, and time.Duration.
// Slices and maps are best loaded from config files (YAML/JSON/TOML).
//
// Run from its directory: cd examples/05-advanced-types && go run .
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nniel-ape/gonfig"
)

type Config struct {
	Server struct {
		Host         string        `default:"localhost"  description:"server host"`
		Port         int           `default:"8080"       description:"server port"`
		ReadTimeout  time.Duration `default:"30s"        description:"read timeout"`
		WriteTimeout time.Duration `default:"30s"        description:"write timeout"`
	}
	AllowedOrigins []string          `description:"CORS allowed origins"`
	RateLimits     []int             `description:"rate limit tiers (req/min)"`
	Weights        []float64         `description:"load balancer weights"`
	Labels         map[string]string `description:"metadata labels"`
}

func main() {
	var cfg Config
	if err := gonfig.Load(&cfg, gonfig.WithFile("config.yaml")); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Server:")
	fmt.Printf("  Host:         %s\n", cfg.Server.Host)
	fmt.Printf("  Port:         %d\n", cfg.Server.Port)
	fmt.Printf("  ReadTimeout:  %s\n", cfg.Server.ReadTimeout)
	fmt.Printf("  WriteTimeout: %s\n", cfg.Server.WriteTimeout)

	fmt.Println("AllowedOrigins:")
	for _, o := range cfg.AllowedOrigins {
		fmt.Printf("  - %s\n", o)
	}

	fmt.Println("RateLimits:")
	for _, r := range cfg.RateLimits {
		fmt.Printf("  - %d req/min\n", r)
	}

	fmt.Println("Weights:")
	for _, w := range cfg.Weights {
		fmt.Printf("  - %.2f\n", w)
	}

	fmt.Println("Labels:")
	for k, v := range cfg.Labels {
		fmt.Printf("  %s: %s\n", k, v)
	}
}
