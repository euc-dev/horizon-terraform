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

func dataSourceLdapValidation() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceLdapRead,
		Schema: map[string]*schema.Schema{
			"consolidated_status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"response": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"component": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"response_status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"errors": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"error_key": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"error_message": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
func dataSourceLdapRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	url := fmt.Sprintf("%s/rest/config/v1/connection-servers/action/validate-ldap-requirements", config.ServerURL)

	tflog.Info(ctx, "Making API request", map[string]interface{}{
		"url": url,
	})

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := config.httpClient.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return diag.FromErr(err)
		}

		consolidatedStatus := "PASS"
		for _, component := range result {
			if status, ok := component["response_status"].(string); ok {
				if status == "FAILED" {
					consolidatedStatus = "FAILED"
					break
				} else if status == "WARNING" && consolidatedStatus != "FAILED" {
					consolidatedStatus = "WARNING"
				}
			}
		}

		if err := d.Set("response", flattenResponse(result)); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("consolidated_status", consolidatedStatus); err != nil {
			return diag.FromErr(err)
		}

		d.SetId(fmt.Sprintf("ldap_PreCheck"))

		return nil
	case http.StatusBadRequest:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Bad request: %s", resp.Status),
		})
	case http.StatusUnauthorized:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unauthorized: %s", resp.Status),
		})
	case http.StatusForbidden:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Access forbidden: %s", resp.Status),
		})
	case http.StatusTooManyRequests:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Too many requests: %s", resp.Status),
		})
	default:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unexpected response: %s", resp.Status),
		})
	}
	return diags
}
