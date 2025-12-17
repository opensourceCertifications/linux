// monitor/go/breaks/picker_main.go
package main

import (
	// Replace "yourmodule" with the module path from your go.mod
	"chaos-agent/library"
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
	files, err := library.PickRandomBinaries()
	if err != nil {
		library.SendMessage(MonitorIP, MonitorPort, "chaos_report", "broken", Token, EncryptionKey)
	}
	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("files to be corrupted: %s", files), Token, EncryptionKey)
	for _, file := range files {
		err := library.CorruptFile(file, 100) // Corrupt 10% of the file
		if err != nil {
			library.SendMessage(MonitorIP, MonitorPort, "chaos_report", "broken", Token, EncryptionKey)
		}
	}
	for _, file := range files {
		library.SendMessage(MonitorIP, MonitorPort, "variable", fmt.Sprintf("BrokenFiles,%s", file), Token, EncryptionKey)
	}
}
