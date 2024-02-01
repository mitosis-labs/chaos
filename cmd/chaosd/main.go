package main

import (
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/cosmos-builders/chaos/app"

	"github.com/cosmos-builders/chaos/app/params"
	"github.com/cosmos-builders/chaos/cmd/chaosd/cmd"
)

func main() {
	params.SetAddressPrefixes()
	rootCmd, _ := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "CHAOSD", app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}

