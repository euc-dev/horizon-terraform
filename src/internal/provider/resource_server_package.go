package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePackage() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePackageCreate,
		DeleteContext: resourcePackageDelete,
		UpdateContext: resourcePackageUpdate,
		ReadContext:   resourcePackageRead,
		Schema: map[string]*schema.Schema{
			"fileurl": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"build_number": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"checksum": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"file_size_in_bytes": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"filename": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"version": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"display_name": {
				Type:     schema.TypeString,
				Optional: true,
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

func dataSourcePackages() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePackagesRead,
		Schema: map[string]*schema.Schema{
			"packages": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"fileurl": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"build_number": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"checksum": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"file_size_in_bytes": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"filename": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"version": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"id": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func resourcePackageCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	config := m.(*Config)
	fileURL := d.Get("fileurl").(string)
	url := fmt.Sprintf("%s/rest/config/v1/server-installer-packages/action/register?fileUrl=%s", config.ServerURL, fileURL)

	body := map[string]interface{}{
		"build_number":       d.Get("build_number").(string),
		"checksum":           d.Get("checksum").(string),
		"display_name":       d.Get("display_name").(string),
		"file_size_in_bytes": d.Get("file_size_in_bytes").(int),
		"filename":           d.Get("filename").(string),
		"version":            d.Get("version").(string),
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

	if resp.StatusCode == http.StatusOK {
		var responseBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
			return diag.FromErr(err)
		}

		id, ok := responseBody["server_installer_package_id"].(string)
		if !ok || id == "" {
			return diag.Errorf("server_installer_package_id not found in response body")
		}

		d.SetId(id)
		tflog.Info(ctx, "Package registered successfully", map[string]interface{}{
			"id": id,
		})
		return resourcePackageRead(ctx, d, m)
	} else {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return diag.FromErr(err)
		}
		tflog.Debug(ctx, "Response body", map[string]interface{}{
			"body": string(respBody),
		})

		return append(diags, handleResponseErrors(resp, respBody)...)
	}

}

func resourcePackageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	id := d.Id()
	if id == "" {
		return diag.Errorf("id is required for package get operation")
	}
	url := fmt.Sprintf("%s/rest/config/v1/server-installer-packages/%s", config.ServerURL, id)

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
		return diag.Errorf("Failed to read package: %s", resp.Status)
	}

	var pkg map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("file_size_in_bytes", pkg["file_size_in_bytes"]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("version", pkg["version"]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("checksum", pkg["checksum"]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("build_number", pkg["build_number"]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("filename", pkg["filename"]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("display_name", pkg["display_name"]); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourcePackageDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	id := d.Id()
	if id == "" {
		return diag.Errorf("id is required for package delete operation")
	}

	url := fmt.Sprintf("%s/rest/config/v1/server-installer-packages/%s/action/unregister", config.ServerURL, id)

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
	case http.StatusNoContent:
		d.SetId("")
	case http.StatusNotFound:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("pacakge with ID %s not found", id),
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

func resourcePackageUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Errorf("Updates are not supported for this resource")
}

func dataSourcePackagesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	url := fmt.Sprintf("%s/rest/config/v1/server-installer-packages", config.ServerURL)

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
		return diag.Errorf("Failed to fetch packages: %s", resp.Status)
	}

	var pkgs []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&pkgs); err != nil {
		return diag.FromErr(err)
	}

	var pkgData []interface{}
	for _, pkg := range pkgs {
		pkgEntry := map[string]interface{}{
			"id":                 getString(pkg["id"]),
			"build_number":       getString(pkg["build_number"]),
			"checksum":           getString(pkg["checksum"]),
			"file_size_in_bytes": getString(pkg["file_size_in_bytes"]),
			"filename":           getBool(pkg["filename"]),
			"url":                getBool(pkg["url"]),
			"version":            getString(pkg["version"]),
		}
		pkgData = append(pkgData, pkgEntry)
	}

	if err := d.Set("pkgs", pkgData); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("horizonview_pkgs")

	return diags
}
