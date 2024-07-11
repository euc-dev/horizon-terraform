// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
    "github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
    "terraform-provider-horizonview/internal/provider"
)

func main() {
    plugin.Serve(&plugin.ServeOpts{
        ProviderFunc: provider.Provider,
    })
}
