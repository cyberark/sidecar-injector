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
	"github.com/cyberark/sidecar-injector/pkg/version"
)

// Define environment variables used in Secrets Provider config

func main() {
	var parameters inject.WebhookServerParameters

	// Reset flag package to avoid pollution by glog, which is an indirect dependency
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// retrieve command line parameters
	flag.IntVar(&parameters.Port, "port", 443, "Webhook server port.")
	flag.StringVar(&parameters.CertFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "Path to file containing the x509 Certificate for HTTPS.")
	flag.StringVar(&parameters.KeyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "Path to file containing the x509 Private Key for HTTPS.")
	flag.BoolVar(&parameters.NoHTTPS, "noHTTPS", false, "Run Webhook server as HTTP (not HTTPS).")
	flag.StringVar(&parameters.SecretlessContainerImage, "secretless-image", "cyberark/secretless-broker:latest", "Container image for the Secretless sidecar")
	flag.StringVar(&parameters.AuthenticatorContainerImage, "authenticator-image", "cyberark/conjur-authn-k8s-client:latest", "Container image for the Kubernetes Authenticator sidecar")
	flag.StringVar(&parameters.SecretsProviderContainerImage, "secrets-provider-image", "cyberark/secrets-provider-for-k8s:latest", "Container image for the Secrets Provider sidecar")

	// Flag.parse only covers `-version` flag but for `version`, we need to explicitly
	// check the args
	showVersion := flag.Bool("version", false, "Show current version")

	flag.Parse()

	// Either the flag or the arg should be enough to show the version
	if *showVersion || flag.Arg(0) == "version" {
		fmt.Printf("cyberark-sidecar-injector v%s\n", version.Get())
		return
	}

	log.Printf("cyberark-sidecar-injector v%s starting up...", version.Get())

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
		if parameters.NoHTTPS {
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
