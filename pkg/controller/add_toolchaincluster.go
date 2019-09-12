package controller

import (
	"github.com/alexeykazakov/toolchain-operator/pkg/controller/toolchaincluster"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, toolchaincluster.Add)
}
