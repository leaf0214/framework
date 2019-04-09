package random

import "crypto/rand"

// MakeRandom is a helper that makes a new buffer full of random data.
func MakeRandom(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	return bytes, err
}
