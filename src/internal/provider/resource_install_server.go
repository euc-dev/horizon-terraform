package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceInstallServer() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceInstallSchudleTaskCreate,
		ReadContext:   resourceInstallTaskStatusRead,
		DeleteContext: resourceInstallTaskDelete,
		UpdateContext: resourceInstallTaskUpdate,
		Schema: map[string]*schema.Schema{
			"domain": {
				Type:     schema.TypeString,
				Required: true,
			},
			"password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"server_installer_package_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"server_msi_install_spec": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admin_sid": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"deployment_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"fips_enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"fw_choice": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"html_access": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"install_directory": {
							Type:     schema.TypeString,
							Required: true,
						},
						"primary_connection_server_fqdn": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"server_instance_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"server_recovery_pwd": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
						"server_recovery_pwd_reminder": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"vdm_ipprotocol_usage": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"target_server_fqdn": {
				Type:     schema.TypeString,
				Required: true,
			},
			"user_name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceInstallSchudleTaskCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	config := m.(*Config)

	domain := d.Get("domain").(string)
	password := d.Get("password").(string)
	serverInstallerPackageID := d.Get("server_installer_package_id").(string)
	serverMSIInstallSpec := d.Get("server_msi_install_spec").([]interface{})[0].(map[string]interface{})
	targetServerFQDN := d.Get("target_server_fqdn").(string)
	userName := d.Get("user_name").(string)

	requestBody := map[string]interface{}{
		"domain":                      domain,
		"password":                    password,
		"server_installer_package_id": serverInstallerPackageID,
		"server_msi_install_spec":     map[string]interface{}{},
		"target_server_fqdn":          targetServerFQDN,
		"user_name":                   userName,
	}
	log.Printf("[DEBUG] domain: %s", domain)
	log.Printf("[DEBUG] serverInstallerPackageID: %s", serverInstallerPackageID)
	log.Printf("[DEBUG] targetServerFQDN: %s", targetServerFQDN)
	log.Printf("[DEBUG] userName: %s", userName)
	log.Printf("[DEBUG] serverMSIInstallSpec: %+v", serverMSIInstallSpec)

	spec := requestBody["server_msi_install_spec"].(map[string]interface{})
	spec["fips_enabled"] = serverMSIInstallSpec["fips_enabled"]
	spec["fw_choice"] = serverMSIInstallSpec["fw_choice"]
	spec["install_directory"] = serverMSIInstallSpec["install_directory"]
	spec["server_instance_type"] = serverMSIInstallSpec["server_instance_type"]
	spec["vdm_ipprotocol_usage"] = serverMSIInstallSpec["vdm_ipprotocol_usage"]

	if val, ok := serverMSIInstallSpec["admin_sid"]; ok && val != "" {
		spec["admin_sid"] = val
	}
	if val, ok := d.GetOk("html_access"); ok {
		spec["html_access"] = val.(bool)
	}
	if val, ok := serverMSIInstallSpec["deployment_type"]; ok && val != "" {
		spec["deployment_type"] = val
	}
	if val, ok := serverMSIInstallSpec["primary_connection_server_fqdn"]; ok && val != "" {
		spec["primary_connection_server_fqdn"] = val
	}
	if val, ok := serverMSIInstallSpec["server_recovery_pwd"]; ok && val != "" {
		spec["server_recovery_pwd"] = val
	}
	if val, ok := serverMSIInstallSpec["server_recovery_pwd_reminder"]; ok && val != "" {
		spec["server_recovery_pwd_reminder"] = val
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal request body: %s", err)
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] JSON Body: %s", string(jsonBody))

	url := fmt.Sprintf("%s/rest/config/v1/connection-servers/action/install-connection-server", config.ServerURL)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("[ERROR] Failed to create new HTTP request: %s", err)
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
	case http.StatusNoContent:
		log.Printf("[INFO] Installation request for %s was successful with status 204 No Content", targetServerFQDN)
		return resourceInstallTaskStatusRead(ctx, d, m)
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

func resourceInstallTaskStatusRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	targetServerFQDN := d.Get("target_server_fqdn").(string)

	status, err := pollInstallStatus(ctx, targetServerFQDN, m)
	if err != nil {
		log.Printf("[ERROR] Polling install status failed for %s: %s", targetServerFQDN, err)
		d.SetId("")
		log.Printf("[DEBUG] SetId called with empty string due to error for %s", targetServerFQDN)
		return diag.FromErr(err)
	}

	if status == "POST_INSTALLATION_CHECK_SUCCESS" {
		d.SetId(targetServerFQDN)
		log.Printf("[DEBUG] SetId called with %s for successful install", targetServerFQDN)
	} else {
		log.Printf("[ERROR] Installation failed for %s: %s", targetServerFQDN, status)
		d.SetId("")
		log.Printf("[DEBUG] SetId called with empty string due to failed install for %s", targetServerFQDN)
		return diag.Errorf("Installation failed: %s", status)
	}
	return diags
}

func pollInstallStatus(ctx context.Context, targetServerFQDN string, m interface{}) (string, error) {
	timeout := time.After(60 * time.Minute)
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	status, err := fetchUpgradeStatus(targetServerFQDN, m)
	if err != nil {
		return "", err
	}
	log.Printf("[DEBUG] Initial Installation Status for %s: %s", targetServerFQDN, status)

	if status == "POST_INSTALLATION_CHECK_SUCCESS" {
		return status, nil
	}

	if status != "" && strings.Contains(status, "error_message") {
		log.Printf("[ERROR] Initial installation  status indicates failure: %s", status)
		return "", fmt.Errorf("installation failed: %s", status)
	}

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("timeout while waiting for installation to complete")
		case <-ticker.C:
			var response map[string]interface{}
			status, err := fetchInstallationStatus(targetServerFQDN, m)
			if err != nil {
				return "", err
			}

			log.Printf("[DEBUG] Installation Status for %s: %s", targetServerFQDN, status)

			if status == "POST_INSTALLATION_CHECK_SUCCESS" {
				return status, nil
			}

			if status != "" {
				if errorMsg, ok := response["error_message"].(string); ok && errorMsg != "" {
					return "", fmt.Errorf("installation failed: %s", errorMsg)
				}
				log.Printf("[DEBUG] Continuing to poll, status for %s: %s", targetServerFQDN, status)
			}
		}
	}
}

func fetchInstallationStatus(targetServerFQDN string, m interface{}) (string, error) {
	config := m.(*Config)

	url := fmt.Sprintf("%s/rest/config/v1/connection-servers/action/retrieve-installer-status?fqdn=%s", config.ServerURL, targetServerFQDN)
	log.Printf("%s", url)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create new HTTP request: %s", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := config.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute HTTP request: %s", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return "", fmt.Errorf("failed to decode JSON response: %s", err)
		}
		status, ok := response["status"].(string)

		if errorMsg, ok := response["error_message"].(string); ok && errorMsg != "" {
			log.Printf("[ERROR] Installation error for %s: %s", targetServerFQDN, errorMsg)
			return status, fmt.Errorf("installation failed: %s", errorMsg)
		}

		if !ok {
			return "", fmt.Errorf("unexpected response format: missing 'status'")
		}
		return status, nil

	case http.StatusBadRequest:
		return "", fmt.Errorf("bad request: %s", resp.Status)
	case http.StatusUnauthorized:
		return "", fmt.Errorf("unauthorized: %s", resp.Status)
	case http.StatusForbidden:
		return "", fmt.Errorf("access forbidden: %s", resp.Status)
	case http.StatusTooManyRequests:
		return "", fmt.Errorf("too many requests: %s", resp.Status)
	default:
		return "", fmt.Errorf("unexpected response: %s", resp.Status)
	}
}

func resourceInstallTaskUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceInstallSchudleTaskCreate(ctx, d, m)
}

func resourceInstallTaskDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Errorf("Updates are not supported for this resource")
}
