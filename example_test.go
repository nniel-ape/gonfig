package gonfig_test

import (
	"errors"
	"fmt"

	"github.com/nniel-ape/gonfig"
)

func ExampleLoad() {
	type Config struct {
		DB struct {
			Host string `default:"localhost" description:"database host"`
			Port int    `default:"5432"      description:"database port"`
		}
		LogLevel string `default:"info" description:"logging level"`
	}

	var cfg Config
	err := gonfig.Load(&cfg)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(cfg.DB.Host)
	fmt.Println(cfg.DB.Port)
	fmt.Println(cfg.LogLevel)
	// Output:
	// localhost
	// 5432
	// info
}

func ExampleLoad_withFileContent() {
	type Config struct {
		DB struct {
			Host string `default:"localhost"`
			Port int    `default:"5432"`
		}
		LogLevel string `default:"info"`
	}

	data := []byte(`{"db":{"host":"myhost","port":3306},"log_level":"debug"}`)

	var cfg Config
	err := gonfig.Load(&cfg, gonfig.WithFileContent(data, gonfig.JSON))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(cfg.DB.Host)
	fmt.Println(cfg.DB.Port)
	fmt.Println(cfg.LogLevel)
	// Output:
	// myhost
	// 3306
	// debug
}

func ExampleLoad_validation() {
	type Config struct {
		Port int `default:"0" validate:"required,min=1,max=65535"`
	}

	var cfg Config
	err := gonfig.Load(&cfg)

	var ve *gonfig.ValidationError
	if errors.As(err, &ve) {
		for _, fe := range ve.Errors {
			fmt.Printf("%s: %s\n", fe.Field, fe.Message)
		}
	}
	// Output:
	// Port: required field is empty
	// Port: value 0 is less than minimum 1
}

func ExampleUsage() {
	type Config struct {
		DB struct {
			Host string `default:"localhost" description:"database host"`
			Port int    `default:"5432"      description:"database port"`
		}
		LogLevel string `default:"info" description:"logging level"`
	}

	var cfg Config
	fmt.Print(gonfig.Usage(&cfg, gonfig.WithEnvPrefix("APP")))
	// Output:
	// --log-level  APP_LOG_LEVEL  string  (default: info)  logging level
	//
	// DB:
	//   --db-host  APP_DB_HOST  string  (default: localhost)  database host
	//   --db-port  APP_DB_PORT  int     (default: 5432)       database port
}

func ExampleLoad_structGonfigTag() {
	type Strategy struct {
		Name   string  `default:"momentum"`
		Weight float64 `default:"0.5"`
	}
	type Config struct {
		Strategy Strategy `gonfig:"lm"`
	}

	data := []byte(`{"lm":{"name":"mean_revert","weight":0.8}}`)

	var cfg Config
	err := gonfig.Load(&cfg, gonfig.WithFileContent(data, gonfig.JSON))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(cfg.Strategy.Name)
	fmt.Println(cfg.Strategy.Weight)
	// Output:
	// mean_revert
	// 0.8
}

func ExampleWithFlags() {
	type Config struct {
		Host string `default:"localhost"`
		Port int    `default:"8080"`
	}

	var cfg Config
	err := gonfig.Load(&cfg, gonfig.WithFlags([]string{"--host", "example.com", "--port", "9090"}))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(cfg.Host)
	fmt.Println(cfg.Port)
	// Output:
	// example.com
	// 9090
}
