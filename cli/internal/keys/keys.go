package keys

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

type Pair struct {
	PrivatePath string
	PublicPath  string
	Fingerprint string
}

func Generate(privatePath string) (Pair, error) {
	privatePath, err := filepath.Abs(privatePath)
	if err != nil {
		return Pair{}, err
	}
	if err := os.MkdirAll(filepath.Dir(privatePath), 0o700); err != nil {
		return Pair{}, err
	}
	publicPath := privatePath + ".pub"
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Pair{}, err
	}
	privatePEM, err := ssh.MarshalPrivateKey(private, "pocketbase-pockethost")
	if err != nil {
		return Pair{}, err
	}
	publicKey, err := ssh.NewPublicKey(public)
	if err != nil {
		return Pair{}, err
	}
	if err := os.WriteFile(privatePath, pem.EncodeToMemory(privatePEM), 0o600); err != nil {
		return Pair{}, err
	}
	if err := os.WriteFile(publicPath, ssh.MarshalAuthorizedKey(publicKey), 0o644); err != nil {
		return Pair{}, err
	}
	_ = os.Chmod(privatePath, 0o600)
	return Pair{PrivatePath: privatePath, PublicPath: publicPath, Fingerprint: ssh.FingerprintSHA256(publicKey)}, nil
}

func Signer(data []byte) (ssh.Signer, error) {
	if signer, err := ssh.ParsePrivateKey(data); err == nil {
		return signer, nil
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("unsupported private key format")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return ssh.NewSignerFromKey(key)
}

func DefaultPrivatePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "pocketbase-pockethost", "keys", "pockethost_ed25519"), nil
}
