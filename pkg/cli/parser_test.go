package cli

import (
	"flag"
	"os"
	"testing"
)

func resetFlags() {
	// Reset the default CommandLine flag set for each test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestParse_AllFlagsProvided(t *testing.T) {
	resetFlags()
	os.Args = []string{
		"asena",
		"--http-port=:8080",
		"--https-port=:8443",
		"--cert-file=/path/cert.pem",
		"--key-file=/path/key.pem",
	}

	opts := Parse()

	if *opts.PortHTTP != ":8080" {
		t.Errorf("expected :8080, got %s", *opts.PortHTTP)
	}
	if *opts.PortHTTPS != ":8443" {
		t.Errorf("expected :8443, got %s", *opts.PortHTTPS)
	}
	if *opts.SSLTLSPublicKey != "/path/cert.pem" {
		t.Errorf("expected /path/cert.pem, got %s", *opts.SSLTLSPublicKey)
	}
	if *opts.SSLTLSPrivateKey != "/path/key.pem" {
		t.Errorf("expected /path/key.pem, got %s", *opts.SSLTLSPrivateKey)
	}
}

func TestParse_NoFlagsProvided(t *testing.T) {
	resetFlags()
	os.Args = []string{"asena"}

	opts := Parse()

	if *opts.PortHTTP != "" {
		t.Errorf("expected empty PortHTTP, got %s", *opts.PortHTTP)
	}
	if *opts.PortHTTPS != "" {
		t.Errorf("expected empty PortHTTPS, got %s", *opts.PortHTTPS)
	}
	if *opts.SSLTLSPublicKey != "" {
		t.Errorf("expected empty SSLTLSPublicKey, got %s", *opts.SSLTLSPublicKey)
	}
	if *opts.SSLTLSPrivateKey != "" {
		t.Errorf("expected empty SSLTLSPrivateKey, got %s", *opts.SSLTLSPrivateKey)
	}
}
