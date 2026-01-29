package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"strings"
)

// calculateHash calculates hash based on the specified hash type
func calculateHash(data []byte, hashType string) string {
	hashTypeLower := strings.ToLower(hashType)
	
	switch hashTypeLower {
	case "md5":
		hash := md5.Sum(data)
		return hex.EncodeToString(hash[:])
	case "sha1":
		hash := sha1.Sum(data)
		return hex.EncodeToString(hash[:])
	case "sha256":
		hash := sha256.Sum256(data)
		return hex.EncodeToString(hash[:])
	case "sha512":
		hash := sha512.Sum512(data)
		return hex.EncodeToString(hash[:])
	default:
		// Default to SHA256 if hash type not specified or unknown
		hash := sha256.Sum256(data)
		return hex.EncodeToString(hash[:])
	}
}
