package htpasswd

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Algorithm identifies a password encoding that can be written to an
// htpasswd file.
type Algorithm string

const (
	AlgorithmAPR1        Algorithm = "apr1"
	AlgorithmBcrypt      Algorithm = "bcrypt"
	AlgorithmSHA         Algorithm = "sha"
	AlgorithmPlain       Algorithm = "plain"
	AlgorithmCryptSHA256 Algorithm = "crypt-sha256"
	AlgorithmCryptSHA512 Algorithm = "crypt-sha512"
)

// EncodeOptions controls Encode. Cost is used only for bcrypt; zero selects
// bcrypt.DefaultCost.
type EncodeOptions struct {
	Algorithm Algorithm
	Cost      int
}

// Encode returns an htpasswd-compatible encoding of password. Cryptographic
// salts are read from crypto/rand.
func Encode(password string, options EncodeOptions) (string, error) {
	algorithm := options.Algorithm
	if algorithm == "" {
		algorithm = AlgorithmAPR1
	}

	switch algorithm {
	case AlgorithmAPR1:
		salt, err := randomSalt(8)
		if err != nil {
			return "", err
		}
		return PrefixCryptApr1 + salt + "$" + md5Crypt(password, salt, PrefixCryptApr1), nil
	case AlgorithmBcrypt:
		cost := options.Cost
		if cost == 0 {
			cost = bcrypt.DefaultCost
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
		if err != nil {
			return "", fmt.Errorf("generate bcrypt password: %w", err)
		}
		return string(hash), nil
	case AlgorithmSHA:
		hash := sha1.Sum([]byte(password))
		return "{SHA}" + base64.StdEncoding.EncodeToString(hash[:]), nil
	case AlgorithmPlain:
		return password, nil
	case AlgorithmCryptSHA256, AlgorithmCryptSHA512:
		salt, err := randomSalt(16)
		if err != nil {
			return "", err
		}
		prefix := PrefixCryptSha256
		if algorithm == AlgorithmCryptSHA512 {
			prefix = PrefixCryptSha512
		}
		return prefix + salt + "$" + shaCrypt(password, "", salt, prefix), nil
	default:
		return "", fmt.Errorf("unsupported password algorithm %q", algorithm)
	}
}

const saltAlphabet = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func randomSalt(length int) (string, error) {
	random := make([]byte, length)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	for i := range random {
		random[i] = saltAlphabet[int(random[i])%len(saltAlphabet)]
	}
	return string(random), nil
}
