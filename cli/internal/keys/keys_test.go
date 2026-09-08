package keys

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateProducesUsableEd25519Pair(t *testing.T) {
	pair, err := Generate(filepath.Join(t.TempDir(), "id_ed25519"))
	if err != nil {
		t.Fatal(err)
	}
	privateData, err := os.ReadFile(pair.PrivatePath)
	if err != nil {
		t.Fatal(err)
	}
	publicData, err := os.ReadFile(pair.PublicPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Signer(privateData); err != nil {
		t.Fatal(err)
	}
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey(publicData)
	if err != nil {
		t.Fatal(err)
	}
	if publicKey.Type() != "ssh-ed25519" {
		t.Fatalf("key type = %q", publicKey.Type())
	}
	if pair.Fingerprint != ssh.FingerprintSHA256(publicKey) {
		t.Fatalf("fingerprint does not match public key")
	}
}
