package com

import "crypto/sha1"
import "fmt"

func GetHashString(data []byte) string {
	h := sha1.New()
	h.Write([]byte(data))
	sha1Hash := h.Sum(nil)
	return fmt.Sprintf("%x", sha1Hash)
}
