package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"

	"os"
)

func parseHashFunction(f string) (hash.Hash, error) {
	switch f {
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("Unknown hash function: %s", f)
	}
}

// fileHash computes the hash of a file
func fileHash(filePath string, function string, salt []byte) (string, error) {
	file, err := os.Open(filePath)
	// ct, _ := os.ReadFile(filePath)
	// log.Tracef("file contents: %s", ct)
	if err != nil {
		return "", err
	}

	hash, err := parseHashFunction(function)
	if err != nil {
		return "", nil
	}

	if _, err := hash.Write(salt); err != nil {
		return "", err
	}

	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	sum := hash.Sum(nil)
	return hex.EncodeToString(sum), nil
}
