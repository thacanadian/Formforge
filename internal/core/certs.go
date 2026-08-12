package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type CertPaths struct{ Cert, Key, CA, CAKey, CADER string }

func EnsureCertificates(dataDir string) (CertPaths, error) {
	dir := filepath.Join(dataDir, "certs")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return CertPaths{}, err
	}
	p := CertPaths{
		Cert: filepath.Join(dir, "server.pem"), Key: filepath.Join(dir, "server-key.pem"),
		CA: filepath.Join(dir, "FormForge-Local-CA.pem"), CAKey: filepath.Join(dir, "FormForge-Local-CA-key.pem"),
		CADER: filepath.Join(dir, "FormForge-Local-CA.cer"),
	}

	caCert, caKey, err := loadCA(p)
	if err != nil || caCert == nil || caKey == nil || time.Until(caCert.NotAfter) < 180*24*time.Hour {
		// Version 1.0/1.1 did not retain the CA private key, which made it
		// impossible to renew a server certificate when a LAN address changed.
		// Regenerate only certificate material; customer data is untouched.
		caCert, caKey, err = createCA(p)
		if err != nil {
			return p, err
		}
		if err := createServerCertificate(p, caCert, caKey); err != nil {
			return p, err
		}
		return p, nil
	}

	if serverCertificateNeedsRenewal(p, caCert) {
		if err := createServerCertificate(p, caCert, caKey); err != nil {
			return p, err
		}
	}
	return p, nil
}

func loadCA(p CertPaths) (*x509.Certificate, *rsa.PrivateKey, error) {
	certPEM, err := os.ReadFile(p.CA)
	if err != nil {
		return nil, nil, err
	}
	keyPEM, err := os.ReadFile(p.CAKey)
	if err != nil {
		return nil, nil, err
	}
	certBlock, _ := pem.Decode(certPEM)
	keyBlock, _ := pem.Decode(keyPEM)
	if certBlock == nil || keyBlock == nil {
		return nil, nil, errors.New("invalid local CA material")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func createCA(p CertPaths) (*x509.Certificate, *rsa.PrivateKey, error) {
	caKey, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	caTpl := &x509.Certificate{
		SerialNumber: randomSerial(), Subject: pkix.Name{CommonName: "FormForge Local CA", Organization: []string{"FormForge"}},
		NotBefore: now.Add(-time.Hour), NotAfter: now.AddDate(10, 0, 0),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return nil, nil, err
	}
	if err := writePEM(p.CA, "CERTIFICATE", caDER, 0644); err != nil {
		return nil, nil, err
	}
	if err := writePEM(p.CAKey, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(caKey), 0600); err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(p.CADER, caDER, 0644); err != nil {
		return nil, nil, err
	}
	return caCert, caKey, nil
}

func createServerCertificate(p CertPaths, caCert *x509.Certificate, caKey *rsa.PrivateKey) error {
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	dns, ips := currentCertificateNames()
	now := time.Now().UTC()
	serverTpl := &x509.Certificate{
		SerialNumber: randomSerial(), Subject: pkix.Name{CommonName: "FormForge Local Server", Organization: []string{"FormForge"}},
		NotBefore: now.Add(-time.Hour), NotAfter: now.AddDate(2, 0, 0), DNSNames: dns, IPAddresses: ips,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTpl, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return err
	}
	if err := writePEM(p.Cert, "CERTIFICATE", serverDER, 0600); err != nil {
		return err
	}
	return writePEM(p.Key, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(serverKey), 0600)
}

func serverCertificateNeedsRenewal(p CertPaths, caCert *x509.Certificate) bool {
	certPEM, err := os.ReadFile(p.Cert)
	if err != nil || !fileExists(p.Key) || !fileExists(p.CADER) {
		return true
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return true
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil || time.Until(cert.NotAfter) < 45*24*time.Hour {
		return true
	}
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	if _, err := cert.Verify(x509.VerifyOptions{Roots: roots, DNSName: "localhost", KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}); err != nil {
		return true
	}
	requiredDNS, requiredIPs := currentCertificateNames()
	dnsSet := map[string]bool{}
	for _, name := range cert.DNSNames {
		dnsSet[strings.ToLower(name)] = true
	}
	for _, name := range requiredDNS {
		if !dnsSet[strings.ToLower(name)] {
			return true
		}
	}
	ipSet := map[string]bool{}
	for _, ip := range cert.IPAddresses {
		ipSet[ip.String()] = true
	}
	for _, ip := range requiredIPs {
		if !ipSet[ip.String()] {
			return true
		}
	}
	return false
}

func currentCertificateNames() ([]string, []net.IP) {
	dnsSet := map[string]bool{"localhost": true, "formforge.local": true}
	host, _ := os.Hostname()
	host = strings.TrimSpace(host)
	if host != "" {
		dnsSet[host] = true
		if !strings.Contains(host, ".") {
			dnsSet[host+".local"] = true
		}
	}
	ipSet := map[string]net.IP{
		"127.0.0.1": net.ParseIP("127.0.0.1"),
		"::1":       net.ParseIP("::1"),
	}
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || (!ip.IsPrivate() && !ip.IsLinkLocalUnicast()) {
				continue
			}
			ipSet[ip.String()] = ip
		}
	}
	dns := make([]string, 0, len(dnsSet))
	for name := range dnsSet {
		dns = append(dns, name)
	}
	sort.Strings(dns)
	ipKeys := make([]string, 0, len(ipSet))
	for key := range ipSet {
		ipKeys = append(ipKeys, key)
	}
	sort.Strings(ipKeys)
	ips := make([]net.IP, 0, len(ipKeys))
	for _, key := range ipKeys {
		ips = append(ips, ipSet[key])
	}
	return dns, ips
}

func fileExists(path string) bool { _, err := os.Stat(path); return err == nil }
func randomSerial() *big.Int {
	n, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	return n
}
func writePEM(path, typ string, b []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: typ, Bytes: b})
}
