package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"

	"wled-simulator/internal/api"
	"wled-simulator/internal/ddp"
	"wled-simulator/internal/gui"
	"wled-simulator/internal/state"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"gopkg.in/yaml.v3"
)

// Config holds application configuration
type Config struct {
	Rows        int    `yaml:"rows" flag:"rows"`
	Cols        int    `yaml:"cols" flag:"cols"`
	Wiring      string `yaml:"wiring" flag:"wiring"`
	HTTPAddress string `yaml:"http_address" flag:"http"`
	DDPPort     int    `yaml:"ddp_port" flag:"ddp-port"`
	InitColor   string `yaml:"init_color" flag:"init"`
	Controls    bool   `yaml:"controls" flag:"controls"`
	Headless    bool   `yaml:"headless" flag:"headless"`
	Verbose     bool   `yaml:"verbose" flag:"v"`
}

func main() {
	// Command line flags
	var cfg Config
	flag.IntVar(&cfg.Rows, "rows", 10, "Number of LED rows")
	flag.IntVar(&cfg.Cols, "cols", 2, "Number of LED columns")
	flag.StringVar(&cfg.Wiring, "wiring", "row", "LED wiring pattern: 'row' (row-major) or 'col' (column-major)")
	flag.StringVar(&cfg.HTTPAddress, "http", ":8080", "HTTP listen address")
	flag.IntVar(&cfg.DDPPort, "ddp-port", 4048, "UDP port for DDP")
	flag.StringVar(&cfg.InitColor, "init", "#000000", "Initial color hex")
	flag.BoolVar(&cfg.Controls, "controls", false, "Show power/brightness controls in GUI")
	flag.BoolVar(&cfg.Headless, "headless", false, "Run without GUI")
	flag.BoolVar(&cfg.Verbose, "v", false, "Verbose logging")

	configFile := flag.String("config", "config.yaml", "Configuration file path")
	flag.Parse()

	// Save CLI values before loading config file
	cliValues := cfg

	// Load config file if it exists (this will overwrite cfg with file values)
	if data, err := os.ReadFile(*configFile); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			log.Printf("Error parsing config file: %v", err)
		}
	}

	// Restore CLI values that were explicitly set using reflection
	cfgValue := reflect.ValueOf(&cfg).Elem()
	cliValue := reflect.ValueOf(&cliValues).Elem()
	cfgType := reflect.TypeOf(cfg)

	flag.Visit(func(f *flag.Flag) {
		// Find the struct field that matches this flag
		for i := 0; i < cfgType.NumField(); i++ {
			field := cfgType.Field(i)
			if flagName := field.Tag.Get("flag"); flagName == f.Name {
				// Set the config value to the CLI value
				cfgValue.Field(i).Set(cliValue.Field(i))
				break
			}
		}
	})

	// Validate wiring pattern
	if cfg.Wiring != "row" && cfg.Wiring != "col" {
		log.Fatalf("Invalid wiring pattern '%s'. Must be 'row' or 'col'", cfg.Wiring)
	}

	// Calculate total LEDs
	totalLEDs := cfg.Rows * cfg.Cols

	// Initialize shared state
	ledState := state.NewLEDState(totalLEDs, cfg.InitColor)

	// Setup logging
	if cfg.Verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	fmt.Printf("WLED Simulator starting with %dx%d LED matrix (%d total LEDs, %s-major wiring)\n", cfg.Rows, cfg.Cols, totalLEDs, cfg.Wiring)
	fmt.Printf("HTTP API on %s\n", cfg.HTTPAddress)
	fmt.Printf("DDP listening on port %d\n", cfg.DDPPort)

	// Channel for server startup errors
	startupErrors := make(chan error, 2)
	var wg sync.WaitGroup

	// Start DDP server
	ddpServer := ddp.NewServer(cfg.DDPPort, ledState)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ddpServer.Start(); err != nil {
			if errors.Is(err, syscall.EADDRINUSE) {
				startupErrors <- fmt.Errorf("DDP port %d is already in use. Please choose a different port or stop the other process", cfg.DDPPort)
			} else {
				startupErrors <- fmt.Errorf("DDP server error: %v", err)
			}
			return
		}
		startupErrors <- nil
	}()

	// Start HTTP API
	apiServer := api.NewServer(cfg.HTTPAddress, ledState)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			if errors.Is(err, syscall.EADDRINUSE) {
				startupErrors <- fmt.Errorf("HTTP port %s is already in use. Please choose a different port or stop the other process", cfg.HTTPAddress)
			} else {
				startupErrors <- fmt.Errorf("API server error: %v", err)
			}
			return
		}
		startupErrors <- nil
	}()

	// Wait for both servers to start and check for errors
	fmt.Println("Starting servers...")
	for i := 0; i < 2; i++ {
		if err := <-startupErrors; err != nil {
			// Stop any successfully started servers
			ddpServer.Stop()
			apiServer.Stop()
			// Wait for goroutines to finish
			wg.Wait()
			log.Fatalf("Failed to start servers: %v", err)
		}
	}

	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start GUI if not headless
	if !cfg.Headless {
		fmt.Println("Starting GUI...")
		myApp := app.NewWithID("com.example.wled-simulator")
		guiApp := gui.NewApp(myApp, ledState, cfg.Rows, cfg.Cols, cfg.Wiring, cfg.Controls)

		// Create shutdown function for servers
		shutdownServers := func() {
			// Stop servers first
			if err := ddpServer.Stop(); err != nil {
				log.Printf("Error stopping DDP server: %v", err)
			}
			if err := apiServer.Stop(); err != nil {
				log.Printf("Error stopping API server: %v", err)
			}
		}

		// Set window close handler - this runs on the main UI thread
		guiApp.SetOnClose(func() {
			fmt.Println("\nReceived shutdown signal...")
			shutdownServers()
			myApp.Quit()
		})

		// Handle Ctrl+C in a separate goroutine
		go func() {
			<-c
			fmt.Println("\nReceived shutdown signal...")
			shutdownServers()

			// Use fyne.DoAndWait since we're in a goroutine
			fyne.DoAndWait(func() {
				myApp.Quit()
			})
		}()

		// Run GUI in main thread
		guiApp.Run()
	} else {
		// In headless mode, wait for interrupt
		<-c
		fmt.Println("\nReceived shutdown signal...")

		// Stop servers
		if err := ddpServer.Stop(); err != nil {
			log.Printf("Error stopping DDP server: %v", err)
		}
		if err := apiServer.Stop(); err != nil {
			log.Printf("Error stopping API server: %v", err)
		}
	}

	fmt.Println("Shutting down...")
	wg.Wait()
}
