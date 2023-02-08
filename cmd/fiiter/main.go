package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/PxyUp/fitter/pkg/config"
	"github.com/PxyUp/fitter/pkg/logger"
	"github.com/PxyUp/fitter/pkg/runtime"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"os/signal"
	"path"
	"time"
)

func main() {
	filePath := flag.String("path", "config.yaml", "Path for config file yaml|json")
	flag.Parse()

	cfg := getConfig(*filePath)
	if cfg == nil {
		log.Fatalf("empty config file %s", filePath)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cancel()
	}()
	logger := logger.NewLogger()
	runtime.New(ctx, cfg, logger.With("component", "runtime")).Start()
	logger.Info("shutdown...")
	time.Sleep(time.Second * 5)
}

func getConfig(filePath string) *config.Config {
	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("unable to read config file %s with error %s", filePath, err.Error())
		return nil
	}
	cfg := &config.Config{}
	if path.Ext(filePath) == ".json" {
		err = json.Unmarshal(file, &cfg)
		if err != nil {
			log.Fatalf("unable to json unmarshal config file %s with error %s", filePath, err.Error())
			return nil
		}

		return cfg
	}

	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		log.Fatalf("unable to yaml unmarshal config file %s with error %s", filePath, err.Error())
		return nil
	}

	return cfg
}
