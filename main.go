package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
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
	case "hash-object":
		if len(os.Args) < 3 {
			log.Fatal("Usage: vit hash-object <file>")
		}
		hashObject(os.Args[2], true)
	case "cat-file":
		if len(os.Args) < 4 || os.Args[2] != "-p" {
			log.Fatal("Usage: vit cat-file -p <object_hash>")
		}
		catFile(os.Args[3])
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

func hashObject(filePath string, write bool) {
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", filePath, err)
	}

	header := fmt.Sprintf("blob %d\x00", len(content))

	store := append([]byte(header), content...)

	hash := sha1.Sum(store)
	hasString := fmt.Sprintf("%x", hash)

	if !write {
		fmt.Println(hasString)
		return
	}

	dirName := hasString[:2]
	fileName := hasString[2:]
	
	objectDir := filepath.Join(vitDir, "objects", dirName)
	objectPath := filepath.Join(objectDir, fileName)

	if err := os.MkdirAll(objectDir, 0755); err != nil {
		log.Fatalf("Failed to create object directory %s: %v", objectDir, err)
	}

	file, err := os.Create(objectPath)
	if err != nil {
		log.Fatalf("Failed to create object file: %v", err)
	}
	defer file.Close()

	zw := zlib.NewWriter(file)
	if _, err := zw.Write(store); err != nil {
		log.Fatalf("Failed to write compressed object data: %v", err)
	}
	if err := zw.Close(); err != nil {
		log.Fatalf("Failed to close zlib writer: %v", err)
	}

	fmt.Println(hasString)
}

func catFile(hash string) {
	if len(hash) < 2 {
		log.Fatalf("Invalid hash: %s", hash)
	}

	dirName := hash[:2]
	fileName := hash[2:]
	path := filepath.Join(vitDir, "objects", dirName, fileName)

	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to open object file: %v", err)
	}
	defer file.Close()

	zr, err := zlib.NewReader(file)
	if err != nil {
		log.Fatalf("Failed to create zlib reader: %v", err)
	}
	defer zr.Close()

	content, err := io.ReadAll(zr)
	if err != nil {
		log.Fatalf("Failed to read compressed object data: %v", err)
	}

	parts := bytes.SplitN(content, []byte{0}, 2)
	
	if len(parts) < 2 {
		log.Fatalf("Invalid object format")
	}

	fmt.Println(string(parts[1]))

}