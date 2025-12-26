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
	"sort"
	"strings"
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
		hashObject(os.Args[2])
	case "cat-file":
		if len(os.Args) < 4 {
			log.Fatal("Usage: vit cat-file <-p|-t|-s> <object_hash>")
		}
		flag := os.Args[2]
		hash := os.Args[3]
		catFile(hash, flag)
	case "write-tree":
		writeTree()
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

func hashObject(filePath string) ([]byte, error){
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", filePath, err)
	}

	return saveObject("blob", content), nil
}

func catFile(hash string, flag string) {
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

	headerStr := string(parts[0])
	headerParts := strings.Split(headerStr, " ")

	if len(headerParts) < 2 {
		log.Fatalf("Invalid object header")
	}

	objType := headerParts[0]
	objSize := headerParts[1]

	switch flag {
	case "-t":
		fmt.Println(objType)
	case "-s":
		fmt.Println(objSize)
	case "-p":
		fmt.Print(string(parts[1]))
	}

}

func saveObject(objType string, data []byte) []byte {

	header := fmt.Sprintf("%s %d\x00", objType, len(data))
	store := append([]byte(header), data...)

	hash := sha1.Sum(store)
	hashString := fmt.Sprintf("%x", hash)

	dirName := hashString[:2]
	fileName := hashString[2:]
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

	return hash[:]

}

func writeTree() {
	files, err := os.ReadDir(".")
	if err != nil {
		log.Fatalf("Failed to read current directory: %v", err)
	}

	type TreeEntry struct {
		Name string
		Hash []byte
	}
	var entries []TreeEntry

	for _, file := range files {
		if file.Name() == vitDir || file.Name() == ".git" || file.IsDir() {
			continue
		}
		hash, err := hashObject(file.Name())
		if err != nil {
			log.Fatalf("Failed to hash object %s: %v", file.Name(), err)
		}
		entries = append(entries, TreeEntry{Name: file.Name(), Hash: hash})
	}

	sort.Slice(entries, func(i, j int) bool{
		return entries[i].Name < entries[j].Name
	})


	var buf bytes.Buffer
	for _, e := range entries {
		fmt.Fprintf(&buf, "100644 %s\x00", e.Name)
		buf.Write(e.Hash)
	}
	treeHash := saveObject("tree", buf.Bytes())
	fmt.Printf("%x\n", treeHash)
}