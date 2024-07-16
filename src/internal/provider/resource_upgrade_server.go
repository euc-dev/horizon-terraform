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

func resourceUpgradeServer() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUpgradeSchudleTaskCreate,
		ReadContext:   resourceUpgradeTaskStatusRead,
		DeleteContext: resourceUpgradeTaskDelete,
		UpdateContext: resourceUpgradeTaskUpdate,

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

func resourceUpgradeSchudleTaskCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	config := m.(*Config)

	domain := d.Get("domain").(string)
	password := d.Get("password").(string)
	serverInstallerPackageID := d.Get("server_installer_package_id").(string)
	targetServerFQDN := d.Get("target_server_fqdn").(string)
	userName := d.Get("user_name").(string)

	requestBody := map[string]interface{}{
		"domain":                      domain,
		"password":                    password,
		"server_installer_package_id": serverInstallerPackageID,
		"target_server_fqdn":          targetServerFQDN,
		"user_name":                   userName,
	}
	log.Printf("[DEBUG] domain: %s", domain)
	log.Printf("[DEBUG] serverInstallerPackageID: %s", serverInstallerPackageID)
	log.Printf("[DEBUG] targetServerFQDN: %s", targetServerFQDN)
	log.Printf("[DEBUG] userName: %s", userName)

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal request body: %s", err)
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] JSON Body: %s", string(jsonBody))

	url := fmt.Sprintf("%s/rest/config/v1/connection-servers/action/upgrade-connection-server", config.ServerURL)

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
		log.Printf("[INFO] Upgrade request for %s was successful with status 204 No Content", targetServerFQDN)
		return resourceUpgradeTaskStatusRead(ctx, d, m)
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

func resourceUpgradeTaskStatusRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	targetServerFQDN := d.Get("target_server_fqdn").(string)

	status, err := pollUpgradeStatus(ctx, targetServerFQDN, m)
	if err != nil {
		log.Printf("[ERROR] Polling upgrade status failed for %s: %s", targetServerFQDN, err)
		d.SetId("")
		log.Printf("[DEBUG] SetId called with empty string due to error for %s", targetServerFQDN)
		return diag.FromErr(err)
	}

	if strings.ToUpper(status) == "UPGRADE_SUCCESS" {
		d.SetId(targetServerFQDN)
		log.Printf("[DEBUG] SetId called with %s for successful upgrade", targetServerFQDN)
		return diags
	} else {
		log.Printf("[ERROR] Upgrade failed for %s: %s", targetServerFQDN, status)
		d.SetId("")
		log.Printf("[DEBUG] SetId called with empty string due to failed upgrade for %s", targetServerFQDN)
		return diag.Errorf("upgrade failed: %s", status)
	}

}

func pollUpgradeStatus(ctx context.Context, targetServerFQDN string, m interface{}) (string, error) {
	timeout := time.After(60 * time.Minute)
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	status, err := fetchUpgradeStatus(targetServerFQDN, m)
	if err != nil {
		return "", err
	}
	log.Printf("[DEBUG] Initial Upgrade Status for %s: %s", targetServerFQDN, status)

	if strings.ToUpper(status) == "UPGRADE_SUCCESS" {
		return status, nil
	}

	if status != "" && strings.Contains(status, "error_message") {
		log.Printf("[ERROR] Initial upgrade  status indicates failure: %s", status)
		return "", fmt.Errorf("upgrade failed: %s", status)
	}
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("timeout while waiting for upgrade to complete")
		case <-ticker.C:
			var response map[string]interface{}
			status, err := fetchUpgradeStatus(targetServerFQDN, m)
			if err != nil {
				return "", err
			}

			log.Printf("[DEBUG] Upgrade Status for %s: %s", targetServerFQDN, status)

			if strings.ToUpper(status) == "UPGRADE_SUCCESS" {
				return status, nil
			}

			if status != "" {
				if errorMsg, ok := response["error_message"].(string); ok && errorMsg != "" {
					return "", fmt.Errorf("upgrade failed: %s", errorMsg)
				}
				log.Printf("[DEBUG] Continuing to poll, status for %s: %s", targetServerFQDN, status)
			}
		}
	}
}

func fetchUpgradeStatus(targetServerFQDN string, m interface{}) (string, error) {
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
		return "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return "", fmt.Errorf("failed to decode JSON response: %s", err)
		}
		status, ok := response["status"].(string)
		if !ok {
			return "", fmt.Errorf("unexpected response format: missing 'status'")
		}

		if errorMsg, ok := response["error_message"].(string); ok && errorMsg != "" {
			log.Printf("[ERROR] Upgrade error for %s: %s", targetServerFQDN, errorMsg)
			return status, fmt.Errorf("upgrade failed: %s", errorMsg)
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

func resourceUpgradeTaskUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceUpgradeSchudleTaskCreate(ctx, d, m)
}

func resourceUpgradeTaskDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Errorf("Updates are not supported for this resource")
}
