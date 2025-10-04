package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCertManager_GetWithoutLoad(t *testing.T) {
	cm := &CertManager{} // empty
	if _, err := cm.Get(); err == nil {
		t.Fatal("expected error when no cert loaded, got nil")
	}
}

func TestCertManager_LoadAndGet(t *testing.T) {
	certFile, keyFile := generateCertKey(t)

	cm := &CertManager{}
	if err := cm.Load(certFile, keyFile); err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if _, err := cm.Get(); err != nil {
		t.Fatalf("expected cert, got error: %v", err)
	}
}

func TestValidateTLS_MissingFiles(t *testing.T) {
	_, err := validateTLS("no-cert.pem", "no-key.pem")
	if err == nil {
		t.Errorf("expected error for missing files, got nil")
	}
}

// helper: generate temporary cert/key files
func generateCertKey(t *testing.T) (string, string) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// create a template cert
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	// write cert
	certOut, _ := os.Create(certPath)
	_ = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	_ = certOut.Close()

	// write key
	keyBytes, _ := x509.MarshalECPrivateKey(priv)
	keyOut, _ := os.Create(keyPath)
	_ = pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	_ = keyOut.Close()

	return certPath, keyPath
}
