package main

import (
	"log"
	"os"
)

const path string = "./.vit"

func main() {

	arguments := os.Args[1:]

	switch arguments[1] {
	case "init":
		err := os.MkdirAll(path, 0755)
		if err != nil {
			log.Fatalf("Failed to initialize vit project: %v", err)
		}
		err = os.MkdirAll(path+"/objects", 0755)
		if err != nil {
			log.Fatalf("Failed to initialize vit project: %v", err)
		}
		err = os.MkdirAll(path+"/refs", 0755)
		if err != nil {
			log.Fatalf("Failed to initialize vit project: %v", err)
		}

		arquivo, err := os.Create(path + "/HEAD")
		if err != nil {
			log.Fatalf("Failed to create HEAD file: %v", err)
		}
		defer arquivo.Close()
		arquivo.WriteString("ref: refs/heads/main\n")
	case "add":
		log.Println("Adding file to vit project...")
		
	case "commit":
		log.Println("Committing changes to vit project...")
	
	case "log":
		log.Println("Displaying vit project log...")

	default:
		log.Fatalf("Invalid subcommand: %s", arguments[1])
		return
	}
}