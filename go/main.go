package main

import (
	"fmt"
	"chaos-agent/library"
)

func main() {
	// Call into the library; returns a JSON string.
	audit_scope := library.GetAuditScope()
	fmt.Println(audit_scope)
}
