package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"server_url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("HORIZONVIEW_SERVER_URL", nil),
			},
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("HORIZONVIEW_USERNAME", nil),
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("HORIZONVIEW_PASSWORD", nil),
			},
			"domain": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("HORIZONVIEW_DOMAIN", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"horizonview_role":           resourceRole(),
			"horizonview_package":        resourcePackage(),
			"horizonview_install_server": resourceInstallServer(),
			"horizonview_upgrade_server": resourceUpgradeServer(),
			"horizonview_permissions":    resourcePermissions(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"horizonview_privileges":   dataSourcePrivileges(),
			"horizonview_roles":        dataSourceRoles(),
			"horizonview_packages":     dataSourcePackages(),
			"horizonview_ad_precheck":  dataSourceActiveDirectoryValidation(),
			"horizonview_sys_precheck": dataSourceSystemValidation(),
			"horizonview_vc_precheck":  dataSourceVCenterValidation(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

type Config struct {
	ServerURL   string
	Username    string
	Password    string
	Domain      string
	AccessToken string
	httpClient  *http.Client
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	config := &Config{
		ServerURL: d.Get("server_url").(string),
		Username:  d.Get("username").(string),
		Password:  d.Get("password").(string),
		Domain:    d.Get("domain").(string),
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}

	tflog.Info(ctx, "Authenticating with the server", map[string]interface{}{
		"server_url": config.ServerURL,
		"username":   config.Username,
	})

	// Authenticate and set AuthToken
	accessToken, err := authenticate(config)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	config.AccessToken = accessToken

	tflog.Info(ctx, "Authentication successful", map[string]interface{}{
		"access_token": accessToken,
	})

	return config, diags
}

func authenticate(config *Config) (string, error) {
	url := fmt.Sprintf("%s/rest/login", config.ServerURL)
	body := map[string]string{
		"username": config.Username,
		"password": config.Password,
		"domain":   config.Domain,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := config.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to authenticate: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("no access token found in response")
	}

	return accessToken, nil
}
