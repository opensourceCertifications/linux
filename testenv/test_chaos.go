package main

import (
	"fmt"
	"log"
	"os"
	_ "testenv/breaks/breaks"
	"testenv/internal/comm"
	"testenv/internal/registry"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <function_name> <monitor_ip>", os.Args[0])
	}

	funcName := os.Args[1]
	monitorIP := os.Args[2]

	// Initialize comm with the monitor address and default port 9000
	comm.Init(fmt.Sprintf("%s:%d", monitorIP, 9000))

	fn, ok := registry.Get(funcName)
	if !ok {
		log.Fatalf("Function '%s' not found in registry", funcName)
	}

	log.Printf("Running chaos function: %s", funcName)
	err := fn()
	if err != nil {
		log.Fatalf("Function '%s' returned error: %v", funcName, err)
	}

	log.Printf("Function '%s' executed successfully.", funcName)
}
