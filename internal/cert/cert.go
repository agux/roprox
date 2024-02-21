package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Generate(commonName string, dnsNames []string, certFolder string) (publicKeyFile, privateKeyFile string, err error) {
	// Step 1: Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}

	// Step 2: Create a certificate template
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: commonName,
		},
		DNSNames:              dnsNames,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10 year validity
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Step 3: Create a self-signed certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		return
	}

	// Step 4: Encode the private key and certificate to PEM
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	// construct privateKeyFile and publicKeyFile as absolute file path based off certFolder (as base path),
	// and commonName (convert special characters to valid path if needed).
	// take care of canonnical path separators.
	privateKeyFile = certFolder + "/" + strings.ReplaceAll(commonName, " ", "_") + "_private.pem"
	publicKeyFile = certFolder + "/" + strings.ReplaceAll(commonName, " ", "_") + "_public.pem"

	privateKeyFile = filepath.ToSlash(privateKeyFile)
	publicKeyFile = filepath.ToSlash(publicKeyFile)

	// Step 5: Write the private key and certificate files to privateKeyFile and publicKeyFile respectively
	if err = os.WriteFile(privateKeyFile, privateKeyPEM, 0600); err != nil {
		return
	}
	if err = os.WriteFile(publicKeyFile, certPEM, 0644); err != nil {
		return
	}

	return
}
