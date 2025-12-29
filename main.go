package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const vitDir = ".vit"
const indexFile = "index"

type Commit struct {
	Hash    string
	Author  string
	Date    string
	Message string
	Parent  string
}

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
		hash, err := hashObject(os.Args[2]) 
   		if err != nil {
        	log.Fatal(err)
    	}
    	fmt.Printf("%x\n", hash)
	case "cat-file":
		if len(os.Args) < 4 {
			log.Fatal("Usage: vit cat-file <-p|-t|-s> <object_hash>")
		}
		flag := os.Args[2]
		hash := os.Args[3]
		catFile(hash, flag)
	case "add":
		if len(os.Args) < 3 {
			log.Fatal("Usage: vit add <file_path>")
		}
		addFile(os.Args[2])
	case "write-tree":
		writeTree()
	case "commit-tree":
		if len(os.Args) < 5 {
			log.Fatal("Usage: vit commit-tree <tree_hash> -m <message> [-p <parent_hash>]")
		}

		treeHash := os.Args[2]
		flag := os.Args[3]
		message := os.Args[4]
		parentHash := ""

		if flag != "-m" {
			log.Fatal("Expected -m flag for commit message")
		}

		if len(os.Args) >= 7 && os.Args[5] == "-p" {
			parentHash = os.Args[6]
		}
		commitTree(treeHash, message, parentHash)
	case "update-ref":
		if len(os.Args) < 4 {
			log.Fatal("Usage: vit update-ref <ref_name> <commit_hash>")
		}
		updateRef(os.Args[2], os.Args[3])
	case "log":
		ref := "HEAD"
		if len(os.Args) >= 3 {
			ref = os.Args[2]
		}
		logGraph(ref)
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
		return nil, err
	}

	return saveObject("blob", content), nil
}

func loadIndex() map[string]string {
	indexPath := filepath.Join(vitDir, indexFile)
	indexMap := make(map[string]string)

	file, err := os.Open(indexPath)
	if os.IsNotExist(err) {
		return indexMap
	}
	if err != nil {
		log.Fatalf("Failed to open index file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		if len(parts) >= 2 {
			indexMap[parts[1]] = parts[0]
		}
	}
	return indexMap
}

func saveIndex(indexMap map[string]string) {
	indexPath := filepath.Join(vitDir, indexFile)
	file, err := os.Create(indexPath)
	if err != nil {
		log.Fatalf("Failed to write index: %v", err)
	}
	defer file.Close()

	var names []string
	for name := range indexMap {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		line := fmt.Sprintf("%s %s\n", indexMap[name], name)
		file.WriteString(line)
	}
}

func addFile(filePath string) {
	hash, err := hashObject(filePath)
	if err != nil {
		log.Fatalf("Failed to add file: %v", err)
	}
	hashHex := fmt.Sprintf("%x", hash)

	indexMap := loadIndex()
	indexMap[filePath] = hashHex
	saveIndex(indexMap)

	fmt.Printf("Added %s to staging area\n", filePath)
}

func catFile(hash string, flag string) {

	objType, content, err := readObject(hash)
	if err != nil {
		log.Fatal(err)
	}

	switch flag {
	case "-t":
		fmt.Println(objType)
	case "-s":
		fmt.Println(len(content))
	case "-p":
		if objType == "tree" {
			fmt.Println("Tree object content (binary)")
		} else {
			fmt.Print(string(content))
		}
	}
}

func readObject(hash string) (string, []byte, error) {
	if len(hash) < 2 {
		return "", nil, fmt.Errorf("invalid hash")
	}
	dirName := hash[:2]
	fileName := hash[2:]
	path := filepath.Join(vitDir, "objects", dirName, fileName)

	file, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("object not found %s", hash)
	}
	defer file.Close()

	zr, err := zlib.NewReader(file)
	if err != nil {
		return "", nil, err
	}
	defer zr.Close()

	raw, err := io.ReadAll(zr)
	if err != nil {
		return "", nil, err
	}

	parts := bytes.SplitN(raw, []byte{0}, 2)
	if len(parts) < 2 {
		return "", nil, fmt.Errorf("corrupted object")
	}

	header := string(parts[0])
	headerParts := strings.Split(header, " ")
	objType := headerParts[0]

	return objType, parts[1], nil
}

func resolveRef(ref string) (string, error) {
	path := filepath.Join(vitDir, ref)
	data, err := os.ReadFile(path)
	
	if os.IsNotExist(err) {
		if len(ref) == 40 {
			return ref, nil
		}
		return "", fmt.Errorf("ref not found: %s", ref)
	}
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(data))
	
	if strings.HasPrefix(content, "ref: ") {
		return resolveRef(strings.TrimPrefix(content, "ref: "))
	}

	return content, nil
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
	indexMap := loadIndex()

	if len(indexMap) == 0 {
		log.Fatal("Nothing to write-tree (staging area is empty)")
	}

	type TreeEntry struct {
		Name string
		HashHex string
	}
	var entries []TreeEntry

	for name, hashHex := range indexMap {
		entries = append(entries, TreeEntry{Name: name, HashHex: hashHex})
	}

	sort.Slice(entries, func(i, j int) bool{
		return entries[i].Name < entries[j].Name
	})


	var buf bytes.Buffer
	for _, e := range entries {
		var hashBytes []byte
		fmt.Sscanf(e.HashHex, "%x", &hashBytes)

		fmt.Fprintf(&buf, "100644 %s\x00", e.Name)
		buf.Write(hashBytes)
	}
	treeHash := saveObject("tree", buf.Bytes())
	fmt.Printf("%x\n", treeHash)
}

func commitTree(treeHash, message, parentHash string) {
	timestamp := time.Now().Unix()
	timezone := "-0300"
	author := fmt.Sprintf("Victor <victor@vit.com> %d %s", timestamp, timezone)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "tree %s\n", treeHash)

	if parentHash != "" {
		fmt.Fprintf(&buf, "parent %s\n", parentHash)
	}
	
	fmt.Fprintf(&buf, "author %s\n", author)
	fmt.Fprintf(&buf, "committer %s\n", author)
	fmt.Fprintf(&buf, "\n%s\n", message)

	commitHash := saveObject("commit", buf.Bytes())
	fmt.Printf("%x\n", commitHash)
}

func updateRef(refName, commitHash string) {
	if len(commitHash) != 40 {
		log.Fatalf("Invalid commit hash length: %s", commitHash)
	}
	path := filepath.Join(vitDir, refName)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create ref directory: %v", err)
	}
	content := []byte(commitHash + "\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		log.Fatalf("Failed to update ref %s: %v", refName, err)
	}
	fmt.Printf("Updated %s to %s\n", refName, commitHash)
}

func parseCommit(hash string) (*Commit, error) {
	objType, content, err := readObject(hash)
	if err != nil {
		return nil, err
	}
	if objType != "commit" {
		return nil, fmt.Errorf("object %s is not a commit", hash)
	}

	lines := strings.Split(string(content), "\n")
	commit := &Commit{Hash: hash}
	
	for i, line := range lines {
		// Linha em branco separa header do body (mensagem)
		if line == "" {
			commit.Message = strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
			break
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		
		key, value := parts[0], parts[1]
		switch key {
		case "parent":
			commit.Parent = value
		case "author":
			fields := strings.Fields(value)
			if len(fields) > 2 {
				ts, _ := strconv.ParseInt(fields[len(fields)-2], 10, 64)
				commit.Date = time.Unix(ts, 0).Format(time.RFC1123)
				commit.Author = strings.Join(fields[:len(fields)-2], " ")
			}
		}
	}
	return commit, nil
}

func logGraph(ref string) {
	currentHash, err := resolveRef(ref)
	if err != nil {
		log.Fatalf("Failed to resolve ref %s: %v", ref, err)
	}

	for currentHash != "" {
		commit, err := parseCommit(currentHash)
		if err != nil {
			log.Fatalf("Error parsing commit %s: %v", currentHash, err)
		}

	
		fmt.Printf("\033[33mcommit %s\033[0m\n", commit.Hash) // Amarelo
		fmt.Printf("Author: %s\n", commit.Author)
		fmt.Printf("Date:   %s\n", commit.Date)
		fmt.Printf("\n    %s\n\n", commit.Message)

		currentHash = commit.Parent
	}
}