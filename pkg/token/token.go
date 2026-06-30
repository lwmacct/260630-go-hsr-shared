package token

import (
	"crypto/rand"
	"errors"
	"math/big"
)

const (
	alphabet      = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	defaultLength = 40
)

func NewWithPrefix(prefix string) (string, error) {
	value, err := NewBase62(defaultLength)
	if err != nil {
		return "", err
	}
	return prefix + "_" + value, nil
}

func NewBase62(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("token length must be positive")
	}
	alphabetSize := big.NewInt(int64(len(alphabet)))
	buf := make([]byte, 0, length)
	for len(buf) < length {
		index, err := rand.Int(rand.Reader, alphabetSize)
		if err != nil {
			return "", err
		}
		buf = append(buf, alphabet[index.Int64()])
	}
	return string(buf), nil
}
