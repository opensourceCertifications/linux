// monitor/go/breaks/picker_main.go
package main

import (
	// Replace "yourmodule" with the module path from your go.mod
	"chaos-agent/shared/library"
	"fmt"
	"os"
	"strconv"
)

var (
	MonitorIP      string
	MonitorPortStr string
	MonitorPort    int
	Token          string
	EncryptionKey  string
)

func init() {
	if MonitorPortStr != "" {
		p, err := strconv.Atoi(MonitorPortStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid MonitorPortStr %q: %v\n", MonitorPortStr, err)
		} else {
			MonitorPort = p
		}
	}
}

func main() {
	// Intentionally ignore the returned list for now; no logging/output per your request.
	fmt.Println("Starting file swap chaos operation...")
	files, err := library.PickRandomBinaries()
	if err != nil {
		msg := fmt.Sprintf("failed to pick random binaries: %v", err)
		library.SendMessage(MonitorIP, MonitorPort, "error", msg, Token, EncryptionKey)
		return
	}
	if err == nil {
		library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("files to be jumbled: %s", files), Token, EncryptionKey)
	}
	err = library.CyclicJumble(files)
	if err != nil {
		msg := fmt.Sprintf("error in CyclicJumble: %v", err)
		library.SendMessage(MonitorIP, MonitorPort, "error", msg, Token, EncryptionKey)
		return
	}
}
