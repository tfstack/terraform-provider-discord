//go:build tools
// +build tools

// This file ensures that tool dependencies are tracked in go.mod.
// See https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
