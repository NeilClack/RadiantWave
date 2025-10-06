package main

import (
	"fmt"
	"os"

	"radiantwavetech.com/radiant_wave/internal/application"
)

func main() {
	fmt.Printf("Starting application\n")
	if err := application.Run(); err != nil {
		fmt.Printf("Application exited with a fatal error: %v", err)
		os.Exit(1)
	}
	fmt.Printf("Stopping application")
}
