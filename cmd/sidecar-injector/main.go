package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/cyberark/sidecar-injector/pkg/inject"
)

func main() {
	var parameters inject.WebhookServerParameters

	// retrieve command line parameters
	flag.IntVar(&parameters.Port, "port", 443, "Webhook server port.")
	flag.StringVar(&parameters.CertFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&parameters.KeyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	flag.BoolVar(&parameters.NoTLS, "noTLS", false, "Disable SSL and ignore any certs.")
	flag.StringVar(&parameters.SecretlessContainerImage, "secretless-image", "cyberark/secretless-broker:latest", "Container image for the Secretless sidecar")
	flag.StringVar(&parameters.AuthenticatorContainerImage, "authenticator-image", "cyberark/conjur-kubernetes-authenticator:latest", "Container image for the Kubernetes Authenticator sidecar")
	flag.Parse()

	whsvr := &inject.WebhookServer{
		Params: parameters,
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", parameters.Port),
			TLSConfig: nil,
		},
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", whsvr.Serve)
	whsvr.Server.Handler = mux

	// start webhook server in goroutine
	go func() {
		log.Printf("Serving mutating admission webhook on %s", whsvr.Server.Addr)

		var startServer func() error
		if parameters.NoTLS {
			startServer = func() error {
				return whsvr.Server.ListenAndServe()
			}
		} else {
			startServer = func() error {
				return whsvr.Server.ListenAndServeTLS(
					parameters.CertFile,
					parameters.KeyFile,
				)
			}
		}

		if err := startServer(); err != nil {
			log.Printf("Failed to listen and serve: %v", err)
			os.Exit(1)
		}
	}()

	// listen for OS shutdown signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Printf("Received OS shutdown signal, shutting down webhook server gracefully...")
	whsvr.Server.Shutdown(context.Background())
}
