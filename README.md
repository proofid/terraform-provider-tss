# Thycotic Secret Server - Terraform Provider

The [Thycotic](https://thycotic.com/) [Secret Server](https://thycotic.com/products/secret-server/) [Terraform](https://www.terraform.io/) 
Provider allows you to access and reference Secrets in your vault for use in Terraform configurations. It also allows 
you to create new Secrets, generate passwords for those secrets that are compliant with the secret's password policy, 
and use the generated passwords in the Terraform configuration. 

## Install via Registry

> Preferred way to install

The latest release can be [downloaded from the terraform registry](https://registry.terraform.io/providers/thycotic/tss/latest). The documentation can be found [here](https://registry.terraform.io/providers/thycotic/tss/latest/docs).

If wish to install straight from source, follow the steps below.

## Install form Source

### Terraform 0.12 and earlier

Extract the specific file for your OS and Architecture to the plugins directory
of the user's profile. You may have to create the directory.

| OS      | Default Path                    |
| ------- | ------------------------------- |
| Linux   | `~/.terraform.d/plugins`        |
| Windows | `%APPDATA%\terraform.d\plugins` |

### Terraform 0.13 and later

Terraform 0.13 uses a different file system layout for 3rd party providers. More information on this can be found [here](https://www.terraform.io/upgrade-guides/0-13.html#new-filesystem-layout-for-local-copies-of-providers). The following folder path will need to be created in the plugins directory of the user's profile.

#### Windows

```text
%APPDATA%\TERRAFORM.D\PLUGINS
└───terraform.thycotic.com
    └───thycotic
        └───tss
            └───1.0.2
                └───windows_amd64
```

#### Linux

```text
~/.terraform.d/plugins
└───terraform.thycotic.com
    └───thycotic
        └───tss
            └───1.0.2
                ├───linux_amd64
```

## Usage

For Terraform 0.13+, include the `terraform` block in your configuration, or plan, that specifies the provider:

```terraform
terraform {
  required_providers {
    tss = {
      source = "thycotic/tss"
      version = "1.0.2"
    }
  }
}
```

To run the example, create a `terraform.tfvars`:

```json
tss_username   = "my_app_user"
tss_password   = "Passw0rd."
tss_server_url = "https://example/SecretServer"
tss_secret_id  = "1"
```

## Secret Data Source

The Secret Data Source provides a read-only reference to one of the fields on 
an _existing_ secret in the Thycotic server. The following is an example of 
how to declare a Secret Data Source in your configuration:

```terraform
data "tss_secret" "my_password" {
  id    = var.tss_secret_id
  field = "password"
}
```

Below are the attributes for the Secret Data Source:

| Attribute Name | Attribute Type     | Usage    | Description                                                                                                                                                                                    |
|----------------|--------------------|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| id             | Integer            | Required | The numerical identifier of the secret to reference.                                                                                                                                           |
| field          | String             | Required | The name of the secret field to reference. The field name is also known as the field 'slug' in the Thycotic API.                                                                              |
| value          | String (Sensitive) | Computed | The value of the named field. **WARNING**: Although this is a sensitive field and is masked in the Terraform CLI output, know that this value is available in plaintext in the Terraform state file. |

## Secret Resource

The Secret Resource is a Secret in the Terraform server that is fully managed by 
the Terraform configuration. The following is an example of how to declare a 
Secret Resource in your configuration:

```terraform
resource "tss_secret" "new_secret" {
  name = "Secret Managed by Terraform"
  secret_template_id = 6040
  folder_id = 6
  item {
    field = "password"
    value = "Shhhhhhhhhhhhhh!-123"
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
```

Below are the attributes you'll most likely use for the Secret Resource. Other 
attributes are described in the schema at the bottom of `resource_secret.go`:

| Attribute Name     | Attribute Type | Usage    | Description                                                                                                                                                                                                                                             |
|--------------------|----------------|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| name               | String         | Required | A display name for the secret in the Thycotic web interface.                                                                                                                                                                                            |
| secret_template_id | Integer        | Required | The numerical ID of the template that defines what fields are available on the secret.                                                                                                                                                                                 |
| folder_id          | Integer        | Optional | The numerical ID of the folder that contains the secret. If a folder ID is not provided, the secret will be kept in the root folder. The user in the provider configuration must have permissions to write to this folder for the operation to succeed. |
| item               | Block          | Required | One or more items that populate the fields defined in the secret's template. The item structure is described in the following table.                                                                                                                    |

Below are the attributes on each `item` block that you're most likely to use for
the Secret Resource. Other attributes are described in the schema at the bottom 
of `resource_secret.go`:

| Attribute Name | Attribute Type     | Usage    | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
|----------------|--------------------|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| field          | String             | Required | The name of the secret field that corresponds to this item. The field name is also known as the field 'slug' in the Thycotic API. The template for the secret defines what field names are available for the secret, as well as the type for each field (eg: text, note, password, file, list, etc.)                                                                                                                                                          |
| value          | String (Sensitive) | Required | The value for the field. If this item is a file item, you may provide the contents of the file here directly as plain text, or you may use one of the Terraform functions to read in the contents of a file on disk, such as `file("/some/file/path.txt")` or `filebase64("/some/file/path.txt")`. **WARNING**: Although this is a sensitive field and is masked in the Terraform CLI output, know that this value is available in plaintext in the Terraform state file.                                                                                                                                                           |
| filename       | String             | Optional | The name to give a file when it is uploaded to the Thycotic server. Default value is `File.txt` if a name is not provided. This attribute is ignored if the field is not a file field.                                                                                                                                                                                                                                                                        |
| file_encoded   | Boolean            | Optional | Whether the contents of the file attachment have been Base64 encoded and should therefore be decoded by this plugin before posting the file contents to the Thycotic Secret Server. You should set this to `true` if you're using the Terraform `filebase64()` function to read in the contents of a file that is _not_ UTF-8 encoded, which is a requirement for the Terraform `file()` function. This attribute is ignored if this item is not a file item. |

## Generated Password Resource

The Generated Password Resource is provided as a means to generate a password
that conforms a secret's password policy. The output of this resource may then
be used to populate the sensitive fields on a Secret Resource. The following 
is an example of how to declare a Generated Password Resource in your 
configuration:

```terraform
resource "tss_generated_password" "database_user_password" {
  template_id = 6040
  field = "password"
}
```

Below are the attributes for the Generated Password Resource:

| Attribute Name | Attribute Type     | Usage    | Description                                                                                                                                                                                        |
|----------------|--------------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| template_id    | Integer            | Required | The numerical identifier of the secret template which contains the password field, and which declares minimum password requirements for that field.                                                |
| field          | String             | Required | The name (aka: slug) of the field for which the password is generated. This field must be declared on the secret template as a password field.                                                     |
| value          | String (Sensitive) | Computed | The generated password. **WARNING**: Although this is a sensitive field and is masked in the Terraform CLI output, know that this value is available in plaintext in the Terraform state file. |

**NOTE**: The Generated Password should ideally have been modelled as a Terraform 
data source since it simply provides data. However, because the password is 
ephemeral and does not persist anywhere on the Thycotic server, this plugin has no
way to know if a read on its value is the first read, second read, and so on. This
would force such a data source to generate a new value with each execution of the 
Terraform plan, which is counterproductive.