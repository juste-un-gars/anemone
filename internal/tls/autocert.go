// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

// Package tls handles TLS certificate generation and management for HTTPS.
package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/juste-un-gars/anemone/internal/logger"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// CertConfig holds certificate configuration
type CertConfig struct {
	CertPath string
	KeyPath  string
	DataDir  string
}

// GenerateOrLoadCertificate generates a self-signed certificate or loads existing one
func GenerateOrLoadCertificate(cfg *CertConfig) error {
	// Check if certificate already exists
	if _, err := os.Stat(cfg.CertPath); err == nil {
		if _, err := os.Stat(cfg.KeyPath); err == nil {
			logger.Info("🔒 Using existing TLS certificate", "path", cfg.CertPath)
			return nil
		}
	}

	logger.Info("🔐 Generating self-signed TLS certificate...")

	// Create certs directory if it doesn't exist
	certDir := filepath.Dir(cfg.CertPath)
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return fmt.Errorf("failed to create certs directory: %w", err)
	}

	// Generate private key using ECDSA P-256
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour) // 10 years

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Anemone NAS"},
			CommonName:   "Anemone Self-Signed Certificate",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "anemone.local", "nas.local"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// Add local network IP addresses
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil || ipnet.IP.To16() != nil {
					template.IPAddresses = append(template.IPAddresses, ipnet.IP)
				}
			}
		}
	}

	// Create self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Write certificate to file
	certOut, err := os.OpenFile(cfg.CertPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open cert file for writing: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Write private key to file
	keyOut, err := os.OpenFile(cfg.KeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open key file for writing: %w", err)
	}
	defer keyOut.Close()

	privBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	logger.Info("✅ Self-signed certificate generated successfully")
	logger.Info("Certificate", "cert_path", cfg.CertPath)
	logger.Info("Private Key", "key_path", cfg.KeyPath)
	logger.Info("Valid for: 10 years (until )", "format", notAfter.Format("2006-01-02"))
	logger.Info("   ⚠️  Your browser will show a security warning - this is normal for self-signed certificates")

	return nil
}
