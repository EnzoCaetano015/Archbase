package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "version" {
		os.Exit(2)
	}
	value := os.Getenv("ARC_UPDATE_TEST_VERSION")
	if value == "oversized" {
		fmt.Print(strings.Repeat("x", 2048))
		return
	}
	fmt.Println("arc " + value)
}
