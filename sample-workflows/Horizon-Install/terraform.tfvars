# terraform.tfvars

server_url          = "https://<Connection_Server_Fqdn>"
username            = "<horizon_administrator>"
password            = "<password>"
domain              = "<domain>"

install_target_servers_fqdn = {
  Connection_Server = ["<Server_Fqdn>"]
  Replica_Servers = ["<ReplicaServer1_Fqdn>","<ReplicaServer2_Fqdn>"]
  Enrollment_Servers = ["<EnrollmentServer_Fqdn>"]
}

install_parameters = {
  install_admin_user           = "<horizon_administrator>"
  install_admin_password       = "<Password>"
  install_domain               =  "<domain>"
  admin_sid                    = "<admin_sid>"
  deployment_type              = "<Deployment_Type>"
  fips_enabled                 = false
  fw_choice                    = true
  html_access                  = true
  install_directory            = "%ProgramFiles%\\VMware\\VMware View\\Server"
  server_recovery_pwd          = "<password>"
  server_recovery_pwd_reminder = "<password_reminder>"
  vdm_ipprotocol_usage         = "IPv4"
}

horizonview_package = {
  fileurl = "https://<Web_Server>VMware-Horizon-Connection-Server-x86_64-8.13.0-xxxxxxxx.exe"
  file_size_in_bytes = "xxxxxxxxxxxxxxxxxx"
  version = "8.13.0"
  checksum = "xxxxxxxxxxxxxxxxxxxxx"
  build_number = "xxxxxxxxx"
  filename = "VMware-Horizon-Connection-Server-x86_64-8.13.0-xxxxxxxxxx.exe"
}

pre_check_parameters = {
  active_directory_fqdn = "AD_FQDN"
  target_cs_version = "2406"
  vcenter_fqdn = "vCenter_Fqdn"
  vcenter_version = "8.0.2"

}

permission_parameters = {
  ad_user_or_group_id        = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  local_access_group_id      = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

}
