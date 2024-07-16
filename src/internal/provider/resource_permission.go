package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePermissions() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePermissionsCreate,
		DeleteContext: resourcePermissionsDelete,
		UpdateContext: resourcePermissionsUpdate,
		ReadContext:   resourcePermissionsRead,
		CustomizeDiff: customizeDiffPermissions,
		Schema: map[string]*schema.Schema{
			"permissions": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ad_user_or_group_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"federation_access_group_id": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  nil,
						},
						"local_access_group_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"role_id": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"display_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"group": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePermissionsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	config := m.(*Config)
	url := fmt.Sprintf("%s/rest/config/v1/permissions", config.ServerURL)

	permissions := d.Get("permissions").([]interface{})

	var body []map[string]interface{}
	for _, perm := range permissions {
		permMap := perm.(map[string]interface{})
		permBody := map[string]interface{}{
			"ad_user_or_group_id":   permMap["ad_user_or_group_id"].(string),
			"local_access_group_id": permMap["local_access_group_id"].(string),
			"role_id":               permMap["role_id"].(string),
		}

		if v, ok := permMap["federation_access_group_id"].(string); ok && v != "" {
			permBody["federation_access_group_id"] = v
		}

		body = append(body, permBody)
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal request body: %s", err)
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] JSON Body: %s", string(jsonBody))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return diag.FromErr(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := config.httpClient.Do(req)
	if err != nil {
		log.Printf("[ERROR] Failed to execute HTTP request: %s", err)
		return diag.FromErr(err)
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] Response Status: %s", resp.Status)

	switch resp.StatusCode {
	case http.StatusOK:
		log.Printf("[INFO] Create permission successful with status 200 OK")
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return diag.FromErr(err)
		}

		var result []map[string]interface{}
		if err := json.Unmarshal(responseBody, &result); err != nil {
			return diag.FromErr(err)
		}

		if len(result) == 0 {
			return diag.FromErr(fmt.Errorf("no response received"))
		}

		if id, ok := result[0]["id"].(string); ok {
			d.SetId(id)
		} else {
			return diag.FromErr(fmt.Errorf("id not found in response"))
		}
		return resourcePermissionsRead(ctx, d, m)
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

func resourcePermissionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	id := d.Id()
	if id == "" {
		return diag.Errorf("id is required for package get operation")
	}
	url := fmt.Sprintf("%s/rest/config/v1/permissions/%s", config.ServerURL, id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.AccessToken))

	resp, err := config.httpClient.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] Response Status: %s", resp.Status)

	switch resp.StatusCode {
	case http.StatusOK:
		var permissions map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&permissions); err != nil {
			return diag.FromErr(err)
		}

		if val, ok := permissions["ad_user_or_group_id"].(string); ok {
			d.Set("ad_user_or_group_id", val)
		}
		if val, ok := permissions["federation_access_group_id"].(string); ok {
			d.Set("federation_access_group_id", val)
		}
		if val, ok := permissions["local_access_group_id"].(string); ok {
			d.Set("local_access_group_id", val)
		}
		if val, ok := permissions["role_id"].(string); ok {
			d.Set("role_id", val)
		}

		// Ignore "display_name" and "group" as requested
		return nil
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

func resourcePermissionsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	id := d.Id()
	if id == "" {
		return diag.Errorf("id is required for permission delete operation")
	}

	url := fmt.Sprintf("%s/rest/config/v1/permissions", config.ServerURL)

	body := []string{id}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return diag.FromErr(err)
	}

	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonBody))
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

	log.Printf("[DEBUG] Response Status: %s", resp.Status)

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		log.Printf("[INFO] Delete permission successful with status %s", resp.Status)
		d.SetId("")
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

func resourcePermissionsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Errorf("Updates are not supported for this resource")
}

func customizeDiffPermissions(ctx context.Context, diff *schema.ResourceDiff, v interface{}) error {
	if diff.HasChange("display_name") {
		diff.Clear("display_name")
	}
	if diff.HasChange("group") {
		diff.Clear("group")
	}
	return nil
}
