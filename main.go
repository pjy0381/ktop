package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/vladimirvivien/ktop/application"
	"github.com/vladimirvivien/ktop/k8s"
	"github.com/vladimirvivien/ktop/views/overview"
)

func main() {
	var ns, kubeCfg, kubeCtx, pg string
	flag.StringVar(&ns, "namespace", "default", "namespace")
	flag.StringVar(&kubeCtx, "context", "", "kubeconfig context")
	flag.StringVar(&pg, "page", "overview", "the default UI page to show")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k8sC, err := k8s.New(ctx, kubeCfg, kubeCtx, ns)
	if err != nil {
		fmt.Printf("failed to create Kubernetes client: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Connected to: %s\n", k8sC.Config().Host)

	app := application.New(k8sC)
	app.WelcomeBanner()
	app.AddPage(overview.New(app, "Overview"))

	// launch application
	appErr := make(chan error)
	go func() {
		appErr <- app.Run(ctx)
	}()

	select {
	case err := <-appErr:
		if err != nil {
			fmt.Printf("app error: %s\n", err)
			os.Exit(1)
		}
	case <-ctx.Done():
	}
	os.Exit(0)
}
