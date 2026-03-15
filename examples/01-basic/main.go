// Example 01-basic demonstrates the simplest gonfig usage: loading defaults only.
package main

import (
	"fmt"
	"log"

	"github.com/nniel-ape/gonfig"
)

type Config struct {
	AppName  string `default:"my-app"  description:"application name"`
	LogLevel string `default:"info"    description:"logging level"`
	Port     int    `default:"8080"    description:"HTTP listen port"`
	Debug    bool   `default:"false"   description:"enable debug mode"`
}

func main() {
	var cfg Config
	if err := gonfig.Load(&cfg); err != nil {
		log.Fatal(err)
	}

	fmt.Println("AppName: ", cfg.AppName)
	fmt.Println("LogLevel:", cfg.LogLevel)
	fmt.Println("Port:    ", cfg.Port)
	fmt.Println("Debug:   ", cfg.Debug)
}
