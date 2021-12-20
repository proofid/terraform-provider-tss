terraform {
  required_version = ">= 0.12.20"
  required_providers {
    tss = {
      source  = "thycotic/tss"
      version = "1.0.2"
    }
  }
}

variable "tss_username" {
  type = string
}

variable "tss_password" {
  type = string
}

variable "tss_server_url" {
  type = string
}

variable "tss_secret_id" {
  type = string
}

provider "tss" {
  username   = var.tss_username
  password   = var.tss_password
  server_url = var.tss_server_url
}

data "tss_secret" "my_username" {
  id    = var.tss_secret_id
  field = "username"
}

data "tss_secret" "my_password" {
  id    = var.tss_secret_id
  field = "password"
}

resource "tss_secret" "new_secret" {
  name = "Secret Managed by Terraform"
  secret_template_id = 6040
  folder_id = 6
  item {
    field = "password"
    value = tss_generated_password.generated_password_for_new_secret.value
  }
  item {
    field = "certificate"
    value = file("/path/my-certificate.cer")
    filename = "my-certificate.cer"
  }
  item {
    field = "key-pair"
    value = filebase64("/path/my-keys.pfx")
    filename = "my-keys.pfx"
    file_encoded = true
  }
}

resource "tss_generated_password" "generated_password_for_new_secret" {
  template_id = 6040
  field = "password"
}

output "username" {
  value     = data.tss_secret.my_username.value
}

output "password" {
  value     = data.tss_secret.my_password.value
}
