package main

import (
	"fmt"
	"log"
	"os"
	"time"
	_ "testenv/breaks/breaks"
	"testenv/internal/comm"
	"testenv/internal/registry"
	"github.com/opensourceCertifications/linux/shared/types"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <function_name> <monitor_ip>", os.Args[0])
	}

	funcName := os.Args[1]
	monitorIP := os.Args[2]

	// Initialize comms
	comm.Init(fmt.Sprintf("%s:%d", monitorIP, 9000))

	fn, ok := registry.Get(funcName)
	if !ok {
		log.Fatalf("Function '%s' not found in registry", funcName)
	}

	log.Printf("Running chaos function: %s", funcName)

	// Send start report
	comm.SendMessage("chaos_report", types.ChaosReport{
		Timestamp: time.Now().Format(time.RFC3339),
		Agent:     "test_chaos",
		Action:    fmt.Sprintf("Starting chaos function: %s", funcName),
	})

	err := fn()
	if err != nil {
		msg := fmt.Sprintf("Function '%s' returned error: %v", funcName, err)
		log.Println(msg)

		// Send failure report
		comm.SendMessage("chaos_report", types.ChaosReport{
			Timestamp: time.Now().Format(time.RFC3339),
			Agent:     "test_chaos",
			Action:    msg,
		})

		os.Exit(1)
	}

	successMsg := fmt.Sprintf("Function '%s' executed successfully.", funcName)
	log.Println(successMsg)

	// Send success report
	comm.SendMessage("chaos_report", types.ChaosReport{
		Timestamp: time.Now().Format(time.RFC3339),
		Agent:     "test_chaos",
		Action:    successMsg,
	})
}
