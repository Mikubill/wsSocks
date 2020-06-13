package main

import (
	"crypto/tls"
	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
	"os"
)

var (
	tlsConfig = tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.CurveP256},
		MinVersion:               tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
	}
	log     = logrus.New()
	authKey []byte
)

func main() {
	mainThread, _ = ants.NewPool(1000)
	log.SetLevel(logrus.InfoLevel)
	app.EnableBashCompletion = true
	err := app.Run(os.Args)
	if err != nil {
		log.Println(err)
	}
}
