package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePrivileges() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePrivilegesRead,
		Schema: map[string]*schema.Schema{
			"privileges": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"sub_privileges": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func dataSourcePrivilegesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	url := fmt.Sprintf("%s/rest/config/v1/privileges", config.ServerURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.AccessToken))

	tflog.Info(ctx, "Making request to fetch privileges", map[string]interface{}{
		"url": url,
	})

	resp, err := config.httpClient.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return diag.Errorf("Failed to fetch privileges: %s", resp.Status)
	}

	var privileges []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&privileges); err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Fetched privileges", map[string]interface{}{
		"privileges": privileges,
	})

	if err := d.Set("privileges", privileges); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("horizonview_privileges")

	return diags
}
