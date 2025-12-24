package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const vitDir = ".vit"

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: vit <command> [<args>]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "init":
		initVit()
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}


func initVit() {
	
	dirs := []string{
		filepath.Join(vitDir, "objects"),
		filepath.Join(vitDir, "refs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	headPath := filepath.Join(vitDir, "HEAD")
	headContent := []byte("ref: refs/heads/main\n")
	if err := os.WriteFile(headPath, headContent, 0644); err != nil {
		log.Fatalf("Failed to create HEAD file: %v", err)
	}

}