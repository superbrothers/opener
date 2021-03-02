package main

import (
	"log"
	"os"
)

func main() {
	cmd := NewOpenerCmd(os.Stderr)
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
