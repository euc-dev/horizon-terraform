# variables.tf

variable "server_url" {
  description = "The URL of the REST server."
  type        = string
}

variable "username" {
  description = "The username for API authentication."
  type        = string
}

variable "password" {
  description = "The password for API authentication."
  type        = string
  sensitive   = true
}

variable "domain" {
  description = "The domain for API authentication."
  type        = string
  sensitive   = true
}

variable "install_target_servers_fqdn" {
  description = "List of target servers"
  type = map(list(string))
}

variable "install_parameters" {
  description = "Horizon server installation API parameters"
  type = map(any)
}

variable "horizonview_package" {
  description = "Horizon Server installer package register API parameters"
  type = map(any)
}

variable "pre_check_parameters" {
  description = "Pre check API parameters"
  type = map(any)
}

variable "permission_parameters" {
  description = "Permission API parameters"
  type = map(any)
}

variable "install_connection_servers" {
  description = "List of servers for connection servers"
  type        = list(string)
  default     = []
}
