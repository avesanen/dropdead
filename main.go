package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var defaultConfig = &config{
	Addr:        ":5001",
	DbPath:      "./data",
	UploadsPath: "./data",
}

func main() {
	var confPath string
	flag.StringVar(&confPath, "c", "", "config file")
	flag.Parse()

	var conf *config

	if confPath == "" {
		log.Printf("Using default config.")
		conf = defaultConfig
	} else {
		log.Printf("Loading config from %s.", confPath)
		c, err := loadConfig(confPath)
		if err != nil {
			log.Printf("Can't load config file: %s", err.Error())
			os.Exit(1)
		}
		conf = c
	}

	d, err := NewDropdead(conf)
	if err != nil {
		log.Printf("Error creating new dropdead: %s", err.Error())
		os.Exit(1)
	}

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, syscall.SIGINT)
	signal.Notify(stopChan, syscall.SIGTERM)

	for {
		errChan := d.ListenAndServe()
		for {
			select {
			case sig := <-stopChan:
				log.Printf("Received %s, shutting down.", sig.String())
				err := d.Shutdown()

				if err != nil {
					log.Printf("Shutdown failed: %s", err.Error())
					continue
				} else {
					log.Println("Shutdown complete.")
					os.Exit(0)
				}
			case err := <-errChan:
				log.Printf("Dropdead shut down unexpectedly: %s", err.Error())
				break
			}
		}
		time.Sleep(time.Second)
	}
}
