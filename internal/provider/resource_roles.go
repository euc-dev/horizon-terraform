package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRoleCreate,
		DeleteContext: resourceRoleDelete,
		UpdateContext: resourceRoleUpdate,
		ReadContext:   resourceRoleRead,
		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"privileges": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				DiffSuppressFunc: suppressSubPrivilegesDiff,
			},
			"read_privileges": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"sub_privileges": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"applies_to_local_access_group": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"applies_to_federation_access_group": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"built_in_role_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func dataSourceRoles() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRolesRead,
		Schema: map[string]*schema.Schema{
			"roles": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"privileges": {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Computed: true,
						},
						"applies_to_local_access_group": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"applies_to_federation_access_group": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"built_in_role_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceRoleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	config := m.(*Config)
	url := fmt.Sprintf("%s/rest/config/v1/roles", config.ServerURL)

	body := map[string]interface{}{
		"name":        d.Get("name").(string),
		"description": d.Get("description").(string),
		"privileges":  d.Get("privileges").([]interface{}),
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return diag.FromErr(err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
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

	if resp.StatusCode == http.StatusCreated {
		location := resp.Header.Get("Location")
		if location == "" {
			return diag.Errorf("No location header in response")
		}
		// Extract the ID from the location header
		id := location[strings.LastIndex(location, "/")+1:]
		d.SetId(id)
		tflog.Info(ctx, "Role created successfully", map[string]interface{}{
			"id": id,
		})
		return resourceRoleRead(ctx, d, m)

	} else {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return diag.FromErr(err)
		}
		tflog.Debug(ctx, "Response body", map[string]interface{}{
			"body": string(respBody),
		})

		return append(diags, handleResponseErrors(resp, respBody)...)
	}

}

func handleResponseErrors(resp *http.Response, respBody []byte) diag.Diagnostics {
	var diags diag.Diagnostics
	switch resp.StatusCode {
	case http.StatusBadRequest:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Bad request: %s", resp.Status),
			Detail:   string(respBody),
		})
	case http.StatusUnauthorized:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unauthorized: %s", resp.Status),
			Detail:   string(respBody),
		})
	case http.StatusForbidden:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Access forbidden: %s", resp.Status),
			Detail:   string(respBody),
		})
	case http.StatusConflict:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Conflict: %s", resp.Status),
			Detail:   string(respBody),
		})
	case http.StatusTooManyRequests:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Too many requests: %s", resp.Status),
			Detail:   string(respBody),
		})
	default:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unexpected response: %s", resp.Status),
			Detail:   string(respBody),
		})
	}
	return diags
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	id := d.Id()
	if id == "" {
		return diag.Errorf("id is required for role get operation")
	}
	url := fmt.Sprintf("%s/rest/config/v1/roles/%s", config.ServerURL, id)

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

	if resp.StatusCode != http.StatusOK {
		return diag.Errorf("Failed to read role: %s", resp.Status)
	}

	var role map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&role); err != nil {
		return diag.FromErr(err)
	}

	if name, ok := role["name"].(string); ok {
		d.Set("name", name)
	}
	if description, ok := role["description"].(string); ok {
		d.Set("description", description)
	}
	if privileges, ok := role["privileges"].([]interface{}); ok {
		// Filter out sub_privileges and only set names
		filteredPrivileges := make([]interface{}, 0)
		for _, p := range privileges {
			if privilege, ok := p.(map[string]interface{}); ok {
				filteredPrivilege := make(map[string]interface{})
				if name, ok := privilege["name"].(string); ok {
					filteredPrivilege["name"] = name
				}
				filteredPrivileges = append(filteredPrivileges, filteredPrivilege)
			}
		}
		if err := d.Set("read_privileges", filteredPrivileges); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func resourceRoleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	id := d.Id()
	if id == "" {
		return diag.Errorf("id is required for role delete operation")
	}

	url := fmt.Sprintf("%s/rest/config/v1/roles/%s", config.ServerURL, id)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.AccessToken))

	resp, err := config.httpClient.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		d.SetId("")
	case http.StatusNotFound:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Role with ID %s not found", id),
		})
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

func resourceRoleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	id := d.Id()
	if id == "" {
		return diag.Errorf("id is required for role update operation")
	}
	url := fmt.Sprintf("%s/rest/config/v1/roles/%s", config.ServerURL, id)

	body := map[string]interface{}{
		"description": d.Get("description").(string),
		"privileges":  d.Get("privileges").([]interface{}),
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return diag.FromErr(err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
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
	case http.StatusNoContent:
		// Resource updated successfully
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
	case http.StatusNotFound:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Not found: %s", resp.Status),
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

func dataSourceRolesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	url := fmt.Sprintf("%s/rest/config/v1/roles", config.ServerURL)

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

	if resp.StatusCode != http.StatusOK {
		return diag.Errorf("Failed to fetch roles: %s", resp.Status)
	}

	var roles []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return diag.FromErr(err)
	}

	var roleData []interface{}
	for _, role := range roles {
		roleEntry := map[string]interface{}{
			"id":                                 getString(role["id"]),
			"name":                               getString(role["name"]),
			"description":                        getString(role["description"]),
			"privileges":                         getList(role["privileges"]),
			"applies_to_local_access_group":      getBool(role["applies_to_local_access_group"]),
			"applies_to_federation_access_group": getBool(role["applies_to_federation_access_group"]),
			"built_in_role_type":                 getString(role["built_in_role_type"]),
		}
		roleData = append(roleData, roleEntry)
	}

	if err := d.Set("roles", roleData); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("horizonview_roles")

	return diags
}

func getString(value interface{}) string {
	if value == nil {
		return ""
	}
	return value.(string)
}

func getBool(value interface{}) bool {
	if value == nil {
		return false
	}
	return value.(bool)
}

func getList(value interface{}) []interface{} {
	if value == nil {
		return []interface{}{}
	}
	return value.([]interface{})
}

func suppressSubPrivilegesDiff(k, old, new string, d *schema.ResourceData) bool {
	return strings.Contains(k, "sub_privileges")
}
