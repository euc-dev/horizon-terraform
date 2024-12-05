terraform {
  required_providers {
    horizonview = {
      source = "local/terraform-horizonview/horizonview"
      version= "0.1.1"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }
}

provider "horizonview" {
    server_url = var.server_url
    username   = var.username
    password   = var.password
    domain     = var.domain
  }

provider "null" {
  # No configuration is needed for the null provider
}

locals {
  cs_server = var.install_target_servers_fqdn["Connection_Server"]
}

locals {
  es_server = var.install_target_servers_fqdn["Replica_Servers"]
}

locals {
  rs_server = var.install_target_servers_fqdn["Enrollment_Servers"]
}

resource "horizonview_role" "create" {
  description = "LCM administrator role."
  name = "LCM"
  privileges = ["LCM_MANAGEMENT"]
}

output "role_id" {
  value = resource.horizonview_role.create.id
}

resource "horizonview_permissions" "permission" {
  permissions{
     ad_user_or_group_id        = var.permission_parameters["ad_user_or_group_id"]
     local_access_group_id      = var.permission_parameters["local_access_group_id"]
     role_id                    = horizonview_role.create.id
   }
  depends_on = [horizonview_role.create]
}

resource "horizonview_package" "Register"{
  fileurl = var.horizonview_package["fileurl"]
  file_size_in_bytes = var.horizonview_package["file_size_in_bytes"]
  version = var.horizonview_package["version"]
  checksum = var.horizonview_package["checksum"]
  build_number = var.horizonview_package["build_number"]
  filename = var.horizonview_package["filename"]

 depends_on = [horizonview_permissions.permission]
}

resource "null_resource" "sleep" {
  provisioner "local-exec" {
    command = "echo 'Sleeping for 180  seconds...' && sleep 180 && echo 'Done sleeping.'"
  }

  depends_on = [horizonview_package.Register]
}

data "horizonview_ad_precheck" "adprecheck" {
  for_each = {
    for fqdn, servers in var.install_target_servers_fqdn : fqdn => servers
    if length(servers) > 0
  }
  target_server_fqdn     = each.value[0]
  active_directory_fqdn  = var.pre_check_parameters["active_directory_fqdn"]
  target_cs_version      = var.pre_check_parameters["target_cs_version"]
  depends_on             = [null_resource.sleep]
}

output "ad_precheck_consolidated_status" {
  value = {
    for fqdn, _ in var.install_target_servers_fqdn : fqdn => data.horizonview_ad_precheck.adprecheck[fqdn].consolidated_status
  }
}

data "horizonview_sys_precheck" "sysprecheck" {
  for_each = {
    for fqdn, servers in var.install_target_servers_fqdn : fqdn => servers
    if length(servers) > 0
  }
  target_server_fqdn     = each.value[0]
  target_cs_version      = var.pre_check_parameters["target_cs_version"]
  depends_on             = [null_resource.sleep]
}

output "sysprecheck_consolidated_status" {
  value = {
    for fqdn, _ in var.install_target_servers_fqdn : fqdn => data.horizonview_sys_precheck.sysprecheck[fqdn].consolidated_status
  }
}

data "horizonview_vc_precheck" "vcprecheck" {
  for_each = {
    for fqdn, servers in var.install_target_servers_fqdn : fqdn => servers
    if length(servers) > 0
  }
  target_server_fqdn     = each.value[0]
  vcenter_fqdn           = var.pre_check_parameters["vcenter_fqdn"]
  target_cs_version      = var.pre_check_parameters["target_cs_version"]
  vcenter_version        = var.pre_check_parameters["vcenter_version"]
  depends_on             = [null_resource.sleep]
}

output "vcprecheck_consolidated_status" {
  value = {
    for fqdn, _ in var.install_target_servers_fqdn : fqdn => data.horizonview_vc_precheck.vcprecheck[fqdn].consolidated_status
  }
}

locals {
    ad_precheck_responses = {
     for fqdn, servers in var.install_target_servers_fqdn : fqdn => {
       for server in servers : server => can(data.horizonview_ad_precheck.adprecheck[server]) ? data.horizonview_ad_precheck.adprecheck[server].response : null
     }
    }

    sys_precheck_responses = {
    for fqdn, servers in var.install_target_servers_fqdn : fqdn => {
      for server in servers : server => can(data.horizonview_sys_precheck.sysprecheck[server]) ? data.horizonview_sys_precheck.sysprecheck[server].response : null
    }
  }

  vc_precheck_responses = {
    for fqdn, servers in var.install_target_servers_fqdn : fqdn => {
      for server in servers : server => can(data.horizonview_vc_precheck.vcprecheck[server]) ? data.horizonview_vc_precheck.vcprecheck[server].response : null
    }
  }

  ad_precheck_pass = {
    for fqdn, servers in var.install_target_servers_fqdn : fqdn => [
      for server in servers :
        can(data.horizonview_ad_precheck.adprecheck[fqdn]) && data.horizonview_ad_precheck.adprecheck[fqdn].consolidated_status == "PASS"
    ]
  }

  sys_precheck_pass = {
    for fqdn, servers in var.install_target_servers_fqdn : fqdn => [
      for server in servers :
        can(data.horizonview_sys_precheck.sysprecheck[fqdn]) && data.horizonview_sys_precheck.sysprecheck[fqdn].consolidated_status == "PASS"
    ]
  }

  vc_precheck_pass = {
    for fqdn, servers in var.install_target_servers_fqdn : fqdn => [
      for server in servers :
        can(data.horizonview_vc_precheck.vcprecheck[fqdn]) && data.horizonview_vc_precheck.vcprecheck[fqdn].consolidated_status == "PASS"
    ]
  }

  all_tasks_pass = {
      for fqdn, servers in var.install_target_servers_fqdn : fqdn => [
        for idx, server in servers : local.ad_precheck_pass[fqdn][idx] &&
                                     local.sys_precheck_pass[fqdn][idx] &&
                                     local.vc_precheck_pass[fqdn][idx]
      ]
    }

    successful_precheck_servers = compact(flatten([
      for fqdn, servers in var.install_target_servers_fqdn : [
        for idx, server in servers : local.all_tasks_pass[fqdn][idx] ? server : null
      ]
    ]))

    failed_precheck_servers = compact(flatten([
      for fqdn, servers in var.install_target_servers_fqdn : [
        for idx, server in servers : !local.all_tasks_pass[fqdn][idx] ? server : null
      ]
    ]))

    install_connection_servers = compact([
      for idx, server in var.install_target_servers_fqdn["Connection_Server"] : local.all_tasks_pass["Connection_Server"][idx] ? server : null
    ])

    install_replica_servers = compact([
      for idx, server in var.install_target_servers_fqdn["Replica_Servers"] : local.all_tasks_pass["Replica_Servers"][idx] ? server : null
    ])

    install_enrollment_servers = compact([
      for idx, server in var.install_target_servers_fqdn["Enrollment_Servers"] : local.all_tasks_pass["Enrollment_Servers"][idx] ? server : null
    ])

      primary_connection_server_fqdn = length(local.install_connection_servers) > 0 ? local.install_connection_servers[0] : null
  }

output "successful_precheck_servers" {
  value = local.successful_precheck_servers
}

output "failed_precheck_servers" {
  value = local.failed_precheck_servers
}

resource "null_resource" "print_failed_precheck_servers" {
  triggers = {
    failed_precheck_servers = join(",", local.failed_precheck_servers)
  }
  provisioner "local-exec" {
    command = "echo 'Precheck failed, install / upgrade is not supported on the below list of servers:\n${join("\n", local.failed_precheck_servers)}'"
  }
}

resource "null_resource" "print_successful_precheck_servers" {
  triggers = {
    successful_precheck_servers = join(",", local.successful_precheck_servers)
  }
  provisioner "local-exec" {
    command = "echo 'Precheck successful, install / upgrade is supported on the below list of servers:\n${join("\n", local.successful_precheck_servers)}'"
  }
}

locals {
  should_continue = local.failed_precheck_servers != null ? length(local.failed_precheck_servers) == 0 : true
}

resource "null_resource" "precheck_result" {
  count = 1

  provisioner "local-exec" {
    command = <<EOT
    if [[ ${local.should_continue} == "true" ]]; then
      echo "Proceeding with Installation..."
    else
      echo "Precheck failed. Exiting..."
      exit 1
    fi
    EOT
  }
}

resource "horizonview_install_server" "install_cs" {
  domain = var.install_parameters["install_domain"]
  password = var.install_parameters["install_admin_password"]
  server_installer_package_id  = horizonview_package.Register.id
  server_msi_install_spec {
    admin_sid                    = var.install_parameters["admin_sid"]
    deployment_type              = var.install_parameters["deployment_type"]
    fips_enabled                 = var.install_parameters["fips_enabled"]
    fw_choice                    = var.install_parameters["fw_choice"]
    html_access                  = var.install_parameters["html_access"]
    install_directory            = var.install_parameters["install_directory"]
    server_recovery_pwd          = var.install_parameters["server_recovery_pwd"]
    server_recovery_pwd_reminder = var.install_parameters["server_recovery_pwd_reminder"]
    server_instance_type         = "STANDARD_SERVER"
    vdm_ipprotocol_usage         = var.install_parameters["vdm_ipprotocol_usage"]
}
  target_server_fqdn            = var.install_target_servers_fqdn["Connection_Server"][0]
  user_name         = var.install_parameters["install_admin_user"]

  depends_on = [
       null_resource.sleep,
       null_resource.print_successful_precheck_servers,
       null_resource.precheck_result]
}

resource "horizonview_install_server" "install_rs" {
  count = length(var.install_target_servers_fqdn["Replica_Servers"])
  target_server_fqdn = var.install_target_servers_fqdn["Replica_Servers"][count.index]
  domain   = var.install_parameters["install_domain"]
  password = var.install_parameters["install_admin_password"]
  server_installer_package_id  = horizonview_package.Register.id
  server_msi_install_spec {
    fips_enabled                 = var.install_parameters["fips_enabled"]
    fw_choice                    = var.install_parameters["fw_choice"]
    install_directory            = var.install_parameters["install_directory"]
    html_access                  = var.install_parameters["html_access"]
    server_instance_type         = "REPLICA_SERVER"
    primary_connection_server_fqdn = local.primary_connection_server_fqdn
    vdm_ipprotocol_usage         = var.install_parameters["vdm_ipprotocol_usage"]
  }
  user_name = var.install_parameters["install_admin_user"]

  depends_on = [
    null_resource.sleep,
    null_resource.print_successful_precheck_servers,
    null_resource.precheck_result,
    horizonview_install_server.install_cs
  ]
}

resource "horizonview_install_server" "install_es" {
  count = length(var.install_target_servers_fqdn["Enrollment_Servers"])
  target_server_fqdn = var.install_target_servers_fqdn["Enrollment_Servers"][count.index]

  domain  = var.install_parameters["install_domain"]
  password = var.install_parameters["install_admin_password"]
  server_installer_package_id  = horizonview_package.Register.id
  server_msi_install_spec {
    fips_enabled                 = var.install_parameters["fips_enabled"]
    fw_choice                    = var.install_parameters["fw_choice"]
    install_directory            = var.install_parameters["install_directory"]
    server_instance_type         = "ENROLLMENT_SERVER"
    vdm_ipprotocol_usage         = var.install_parameters["vdm_ipprotocol_usage"]
  }
  user_name = "Administrator"

  depends_on = [
    null_resource.sleep,
    null_resource.print_successful_precheck_servers,
    null_resource.precheck_result,
    horizonview_install_server.install_cs
  ]
}
