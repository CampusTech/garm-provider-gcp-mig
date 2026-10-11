// SPDX-License-Identifier: Apache-2.0

// garm-provider-gcp-mig: GARM external provider creating runners through
// regional managed instance groups on Google Cloud.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudbase/garm-provider-common/execution"
	"github.com/cloudbase/garm-provider-common/execution/common"
	"github.com/ut0mt8/garm-provider-gcp-mig/provider"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	env, err := execution.GetEnvironment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting environment: %v\n", err)
		os.Exit(1)
	}

	prov := provider.New(env.ProviderConfigFile, env.ControllerID)
	result, err := env.Run(ctx, prov)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to run command: %v\n", err)
		os.Exit(common.ResolveErrorToExitCode(err))
	}
	if len(result) > 0 {
		fmt.Fprint(os.Stdout, result)
	}
}
