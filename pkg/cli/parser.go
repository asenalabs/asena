package cli

import (
	"flag"
	"fmt"
	"os"
)

type Options struct {
	PortHTTP         string
	PortHTTPS        string
	SSLTLSPublicKey  string
	SSLTLSPrivateKey string
}

func Parse() *Options {
	var opts Options

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&opts.PortHTTP, "http-port", "", "HTTP port for Asena")
	fs.StringVar(&opts.PortHTTPS, "https-port", "", "HTTPS port for Asena")
	fs.StringVar(&opts.SSLTLSPublicKey, "cert-file", "", "Path to SSL/TLS certificate file")
	fs.StringVar(&opts.SSLTLSPrivateKey, "key-file", "", "Path to SSL/TLS private key file")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "\nUsage:\n    asena [flags]\n\nFlags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error:\t%v\n\n", err)
		fs.Usage()
		os.Exit(2)
	}

	return &opts
}
