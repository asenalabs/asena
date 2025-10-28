package cli

import (
	"flag"
	"fmt"
	"os"
)

type Options struct {
	PortHTTP         *string
	PortHTTPS        *string
	SSLTLSPublicKey  *string
	SSLTLSPrivateKey *string
}

func Parse() *Options {
	opts := &Options{
		PortHTTP:         flag.String("http-port", "", "HTTP port for Asena"),
		PortHTTPS:        flag.String("https-port", "", "HTTPS port for Asena"),
		SSLTLSPublicKey:  flag.String("cert-file", "", "Path to SSL/TLS certificate file"),
		SSLTLSPrivateKey: flag.String("key-file", "", "Path to SSL/TLS private key file"),
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\nUsage:\n    asena [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error:\t%v\n\n", err)
		flag.Usage()
		os.Exit(2)
	}

	return opts
}
