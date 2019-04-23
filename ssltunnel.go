package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	inport  int
	outport int
	hosts   string
)

func init() {
	flag.IntVar(&inport, "wrap", 80, "The local port to wrap with SSL.")
	flag.IntVar(&outport, "serve", 443, "The serve port to bind to.")
	flag.StringVar(&hosts, "hosts", "localhost", "A comma separated list of hostnames to serve.")
}

func main() {
	flag.Parse()
	_, kerr := os.Open("key")
	_, cerr := os.Open("cert")
	if os.IsNotExist(kerr) || os.IsNotExist(cerr) {
		genCert()
	} else {
		log.Println("Using existing key and cert.")
	}
	addr := "0.0.0.0:" + strconv.FormatInt(int64(outport), 10)
	log.Println("Binding to " + addr + " and wrapping localhost:" + strconv.FormatInt(int64(inport), 10))
	proxyAddr := "http://127.0.0.1:" + strconv.FormatInt(int64(inport), 10)
	proxyUrl, err := url.Parse(proxyAddr)
	if err != nil {
		log.Println(err)
	}
	http.Handle("/", httputil.NewSingleHostReverseProxy(proxyUrl))
	log.Fatal(http.ListenAndServeTLS(addr, "cert", "key", nil))
}

func genCert() {
	log.Println("Generating private key...")
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Println(err)
		log.Fatal("Unable to generate RSA Private key.")
	} else {
		log.Println("Generated new RSA Private key.")
	}
	log.Println("Self-signing certificate...")
	notBefore := time.Now()
	notAfter := time.Date(2049, 12, 31, 23, 59, 59, 0, time.UTC)
	hostNames := strings.Split(hosts, ",")
	template := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"Super Secure Widgets Co."},
		},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP("0.0.0.0")},
		DNSNames:    hostNames,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Println(err)
		log.Fatal("Unable to generate certificate.")
	} else {
		log.Println("Created self-signed certificate.")
	}
	log.Println("Writing files...")
	cert, err := os.Create("cert")
	if err != nil {
		log.Println(err)
		log.Fatal("Unable to write certificate file.")
	}
	defer cert.Close()
	err = pem.Encode(cert, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if err != nil {
		log.Println(err)
		log.Fatal("Unable to PEM encode certificate.")
	}
	key, err := os.OpenFile("key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Println(err)
		log.Fatal("Unable to write key file.")
	}
	defer key.Close()
	err = pem.Encode(key, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	if err != nil {
		log.Println(err)
		log.Fatal("Unable to PEM encode private key.")
	}
}
