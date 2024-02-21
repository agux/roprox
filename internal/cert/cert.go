package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agux/roprox/internal/conf"
	"github.com/pkg/errors"
)

func LoadOrGenerate(commonName string) (cert tls.Certificate, err error) {
	// check if the certificate and key exist
	certFolder := conf.Args.Proxy.SSLCertificatePath
	// the commonName is most likely a domain address in the URL. make an idiomatic, valid baseFileName based off commonName.
	baseFileName := strings.ToLower(strings.ReplaceAll(commonName, ".", "_"))
	privateKeyFile := certFolder + "/" + baseFileName + "_private.pem"
	publicKeyFile := certFolder + "/" + baseFileName + "_public.pem"

	privateKeyFile = filepath.ToSlash(privateKeyFile)
	publicKeyFile = filepath.ToSlash(publicKeyFile)

	// check if the privateKeyFile and the publicKeyFile exists
	toGen := false
	if _, err := os.Stat(privateKeyFile); os.IsNotExist(err) {
		toGen = true
	}
	if _, err := os.Stat(publicKeyFile); os.IsNotExist(err) {
		toGen = true
	}

	if toGen {
		cert, err = genAndSave(commonName, publicKeyFile, privateKeyFile)
		return
	}

	// load certificate file first and check if expired
	var certPEMBlock []byte
	if certPEMBlock, err = os.ReadFile(publicKeyFile); err != nil {
		err = errors.Wrap(err, "failed to read certificate file")
		return
	}
	pemBlock, _ := pem.Decode(certPEMBlock)
	if pemBlock == nil {
		err = errors.New("failed to decode PEM block from certificate")
		return
	}

	var x509Cert *x509.Certificate
	x509Cert, err = x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		err = errors.Wrap(err, "failed to parse certificate")
		return
	}

	// Check expiration
	if time.Now().After(x509Cert.NotAfter) {
		// certificate is expired, auto renew
		cert, err = genAndSave(commonName, publicKeyFile, privateKeyFile)
	} else {
		// certificate is valid, load the file pair
		cert, err = tls.LoadX509KeyPair(publicKeyFile, privateKeyFile)
	}

	return
}

func Generate(commonName string) (cert tls.Certificate, certPEM, keyPEM []byte, err error) {
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
		// DNSNames:              dnsNames,
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
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	cert, err = tls.X509KeyPair(certPEM, keyPEM)

	return
}

func genAndSave(commonName, publicKeyFile, privateKeyFile string) (cert tls.Certificate, err error) {
	var certPEM, keyPEM []byte
	if cert, certPEM, keyPEM, err = Generate(commonName); err != nil {
		err = errors.Wrapf(err, "failed to generate certificate for %s", commonName)
		return
	}
	// overwrite existing file
	if err = os.WriteFile(publicKeyFile, certPEM, 0644); err != nil {
		err = errors.Wrap(err, "failed to write certificate to file")
		return
	}
	if err = os.WriteFile(privateKeyFile, keyPEM, 0644); err != nil {
		err = errors.Wrap(err, "failed to write private key to file")
		return
	}
	return
}
