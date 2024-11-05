terraform {
  required_providers {
    horizonview = {
      source = "custom/horizonviewprovider/horizonview"
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


resource "horizonview_package" "upgrade_package"{
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

  depends_on = [horizonview_package.upgrade_package]
}

data "horizonview_ldap_precheck" "ldapprecheck" {
  depends_on             = [null_resource.sleep]
}

output "ldapprecheck_status" {
  value = data.horizonview_ldap_precheck.ldapprecheck.consolidated_status
}

locals {
  should_continue = (data.horizonview_ldap_precheck.ldapprecheck.consolidated_status == "PASS")
}

resource "null_resource" "initial_precheck" {
  count = local.should_continue ? 1 : 0

  provisioner "local-exec" {
    command = "echo 'LDAP validation is successful. Executing next set of pre check...'"
  }

  depends_on = [data.horizonview_ldap_precheck.ldapprecheck]
}

data "horizonview_ldap_precheck" "ldapprecheck" {
  depends_on             = [null_resource.sleep]
}

output "ldapprecheck_status" {
  value = data.horizonview_ldap_precheck.ldapprecheck.consolidated_status
}

locals {
  should_continue = (data.horizonview_ldap_precheck.ldapprecheck.consolidated_status == "PASS")
}

resource "null_resource" "control_flow" {
  provisioner "local-exec" {
    command = "if [ \"${local.should_continue}\" != \"true\" ]; then echo 'Ldap Validation Status is FAILED. Aborting further execution.' >&2; exit 1; else echo 'Ldap Validation Status is PASS. Continuing with the tasks...'; fi"
    interpreter = ["sh", "-c"]
  }

  depends_on = [data.horizonview_ldap_precheck.ldapprecheck]
}

data "horizonview_ad_precheck" "adprecheck" {
  for_each = {
    for fqdn, servers in var.upgrade_target_servers_fqdn : fqdn => servers
    if length(servers) > 0
  }
  target_server_fqdn     = each.value[0]
  active_directory_fqdn  = var.pre_check_parameters["active_directory_fqdn"]
  target_cs_version      = var.pre_check_parameters["target_cs_version"]
  depends_on             = [null_resource.control_flow]
}



output "ad_precheck_consolidated_status" {
  value = {
    for fqdn, _ in var.upgrade_target_servers_fqdn : fqdn => data.horizonview_ad_precheck.adprecheck[fqdn].consolidated_status
  }
}

data "horizonview_sys_precheck" "sysprecheck" {
  for_each = {
    for fqdn, servers in var.upgrade_target_servers_fqdn : fqdn => servers
    if length(servers) > 0
  }
  target_server_fqdn     = each.value[0]
  target_cs_version      = var.pre_check_parameters["target_cs_version"]
  depends_on             = [null_resource.control_flow]
}

output "sysprecheck_consolidated_status" {
  value = {
    for fqdn, _ in var.upgrade_target_servers_fqdn : fqdn => data.horizonview_sys_precheck.sysprecheck[fqdn].consolidated_status
  }
}

data "horizonview_vc_precheck" "vcprecheck" {
  for_each = {
    for fqdn, servers in var.upgrade_target_servers_fqdn : fqdn => servers
    if length(servers) > 0
  }
  target_server_fqdn     = each.value[0]
  vcenter_fqdn           = var.pre_check_parameters["vcenter_fqdn"]
  target_cs_version      = var.pre_check_parameters["target_cs_version"]
  vcenter_version        = var.pre_check_parameters["vcenter_version"]
  depends_on             = [null_resource.control_flow]
}

output "vcprecheck_consolidated_status" {
  value = {
    for fqdn, _ in var.upgrade_target_servers_fqdn : fqdn => data.horizonview_vc_precheck.vcprecheck[fqdn].consolidated_status
  }
}

locals {
    ad_precheck_responses = {
     for fqdn, servers in var.upgrade_target_servers_fqdn : fqdn => {
       for server in servers : server => can(data.horizonview_ad_precheck.adprecheck[server]) ? data.horizonview_ad_precheck.adprecheck[server].response : null
     }
    }

    sys_precheck_responses = {
    for fqdn, servers in var.upgrade_target_servers_fqdn : fqdn => {
      for server in servers : server => can(data.horizonview_sys_precheck.sysprecheck[server]) ? data.horizonview_sys_precheck.sysprecheck[server].response : null
    }
  }

  vc_precheck_responses = {
    for fqdn, servers in var.upgrade_target_servers_fqdn : fqdn => {
      for server in servers : server => can(data.horizonview_vc_precheck.vcprecheck[server]) ? data.horizonview_vc_precheck.vcprecheck[server].response : null
    }
  }

  ad_precheck_pass = {
    for fqdn, servers in var.upgrade_target_servers_fqdn : fqdn => [
      for server in servers :
        can(data.horizonview_ad_precheck.adprecheck[fqdn]) && data.horizonview_ad_precheck.adprecheck[fqdn].consolidated_status == "PASS"
    ]
  }

  sys_precheck_pass = {
    for fqdn, servers in var.upgrade_target_servers_fqdn : fqdn => [
      for server in servers :
        can(data.horizonview_sys_precheck.sysprecheck[fqdn]) && data.horizonview_sys_precheck.sysprecheck[fqdn].consolidated_status == "PASS"
    ]
  }

  vc_precheck_pass = {
    for fqdn, servers in var.upgrade_target_servers_fqdn : fqdn => [
      for server in servers :
        can(data.horizonview_vc_precheck.vcprecheck[fqdn]) && data.horizonview_vc_precheck.vcprecheck[fqdn].consolidated_status == "PASS"
    ]
  }

  all_tasks_pass = {
      for fqdn, servers in var.upgrade_target_servers_fqdn : fqdn => [
        for idx, server in servers : local.ad_precheck_pass[fqdn][idx] &&
                                     local.sys_precheck_pass[fqdn][idx] &&
                                     local.vc_precheck_pass[fqdn][idx]
      ]
    }

    successful_precheck_servers = compact(flatten([
      for fqdn, servers in var.upgrade_target_servers_fqdn : [
        for idx, server in servers : local.all_tasks_pass[fqdn][idx] ? server : null
      ]
    ]))

    failed_precheck_servers = compact(flatten([
      for fqdn, servers in var.upgrade_target_servers_fqdn : [
        for idx, server in servers : !local.all_tasks_pass[fqdn][idx] ? server : null
      ]
    ]))
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

resource "horizonview_upgrade_server" "upgrade_servers" {
  count = length(local.successful_precheck_servers)

  domain            = var.upgrade_parameters["upgrade_domain"]
  password          = var.upgrade_parameters["upgrade_admin_password"]
  server_installer_package_id  = horizonview_package.upgrade_package.id
  target_server_fqdn            = local.successful_precheck_servers[count.index]
  user_name         = var.upgrade_parameters["upgrade_admin_user"]

  depends_on = [
  null_resource.sleep,
  null_resource.print_successful_precheck_servers]

}
