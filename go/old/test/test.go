package main

import (
	_ "embed"
	"fmt"
)

//go:embed vars
var varsFile string

func main() {
	fmt.Print(varsFile)
}
