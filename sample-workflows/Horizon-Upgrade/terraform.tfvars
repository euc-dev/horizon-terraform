# terraform.tfvars

server_url          = "https://<Connection_Server_Fqdn>"
username            = "<horizon_administrator>"
password            = "<password>"
domain              = "<domain>"

upgrade_target_servers_fqdn = {
  Connection_Server = ["<Server_Fqdn>"]
  Replica_Servers = ["<ReplicaServer1_Fqdn>","<ReplicaServer2_Fqdn>"]
  Enrollment_Servers = ["<EnrollmentServer_Fqdn>"]
}

upgrade_parameters = {
  upgrade_admin_user           = "<horizon_administrator>"
  upgrade_admin_password       = "<Password>"
  upgrade_domain               =  "<domain>"
}

horizonview_package = {
  fileurl = "https://<Web_Server>/VMware-Horizon-Connection-Server-x86_64-8.13.0-xxxxxxxx.exe"
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
