package server

import (
	"crypto/tls"
	"fmt"
	"os"
	"sync"
)

type CertManager struct {
	mu   sync.RWMutex
	cert *tls.Certificate
}

func NewCertManager(certFile, keyFile string) (*CertManager, error) {
	cm := &CertManager{}
	if certFile != "" && keyFile != "" {
		if err := cm.Load(certFile, keyFile); err != nil {
			return nil, err
		}
	}

	return cm, nil
}

func (m *CertManager) Load(certFile, keyFile string) error {
	newCert, err := validateTLS(certFile, keyFile)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	tmp := newCert
	m.cert = &tmp

	return nil
}

func (m *CertManager) GetCertificate(*tls.Certificate) (*tls.Certificate, error) {
	return m.Get()
}

func (m *CertManager) Get() (*tls.Certificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.cert == nil {
		return nil, fmt.Errorf("[TLS] no TLS certificate loaded")
	}

	return m.cert, nil
}

func validateTLS(certFile, keyFile string) (tls.Certificate, error) {
	var cert tls.Certificate
	if _, err := os.Stat(certFile); err != nil {
		return tls.Certificate{}, fmt.Errorf("[TLS] certificate file %s not accessible: %w", certFile, err)
	}
	if _, err := os.Stat(keyFile); err != nil {
		return tls.Certificate{}, fmt.Errorf("[TLS] key file %s not accessible: %w", keyFile, err)
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("[TLS] failed to load certificate/key (%s, %s): %w", certFile, keyFile, err)
	}

	return cert, nil
}
