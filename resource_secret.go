package main

import (
	"encoding/base64"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/thycotic/tss-sdk-go/server"
	"log"
	"regexp"
	"strconv"
)

func resourceSecretCreate(resourceSecret *schema.ResourceData, meta interface{}) error {
	tss, err := server.New(meta.(server.Configuration))
	if err != nil {
		log.Printf("[ERROR] configuration error: %s", err)
		return err
	}
	log.Printf("[DEBUG] creating secret with name '%s'", resourceSecret.Get("name"))

	template, err := fetchTemplateForResourceSecret(resourceSecret, tss)
	if err != nil {
		log.Printf("[ERROR] unable to fetch template for the secret: %s", err)
		return err
	}

	secretModel, err:= resourceSecretToModel(resourceSecret, template, 0)
	if err != nil {
		log.Printf("[ERROR] error converting secret resource to model: %s", err)
		return err
	}

	secret, err := tss.CreateSecret(*secretModel)
	if err != nil {
		log.Printf("[ERROR] unable to create secret: %s", err)
		return err
	}

	err = modelToResourceSecret(secret, resourceSecret)
	if err != nil {
		log.Printf("[ERROR] error converting model to secret resource: %s", err)
		return err
	}

	log.Printf("[DEBUG] created secret with name '%s' and ID '%d'", secret.Name, secret.ID)
	return nil
}

func resourceSecretRead(resourceSecret *schema.ResourceData, meta interface{}) error {
	id, err := strconv.Atoi(resourceSecret.Id())
	if err != nil {
		log.Printf("[ERROR] configuration error, ID is not an integer: %s", resourceSecret.Id())
		return err
	}

	tss, err := server.New(meta.(server.Configuration))
	if err != nil {
		log.Printf("[ERROR] configuration error: %s", err)
		return err
	}
	log.Printf("[DEBUG] reading secret with name '%s' and ID '%d'", resourceSecret.Get("name"), id)

	secret, err := tss.Secret(id)
	if err != nil {
		log.Printf("[ERROR] unable to read secret: %s", err)
		return err
	}

	err = modelToResourceSecret(secret, resourceSecret)
	if err != nil {
		log.Printf("[ERROR] error converting model to resource secret: %s", err)
		return err
	}

	return nil
}

func resourceSecretUpdate(resourceSecret *schema.ResourceData, meta interface{}) error {
	id, err := strconv.Atoi(resourceSecret.Id())
	if err != nil {
		log.Printf("[ERROR] configuration error, ID is not an integer: %s", resourceSecret.Id())
		return err
	}

	tss, err := server.New(meta.(server.Configuration))
	if err != nil {
		log.Printf("[ERROR] configuration error: %s", err)
		return err
	}
	log.Printf("[DEBUG] updating secret with name '%s' and ID '%d'", resourceSecret.Get("name"), id)

	template, err := fetchTemplateForResourceSecret(resourceSecret, tss)
	if err != nil {
		log.Printf("[ERROR] unable to fetch template for the secret: %s", err)
		return err
	}

	secretModel, err:= resourceSecretToModel(resourceSecret, template, id)
	if err != nil {
		log.Printf("[ERROR] error converting resource secret to model: %s", err)
		return err
	}

	// Key generation is only supported when the secret is created. Passing
	// key args into the UpdateSecret method will cause an error
	secretModel.SshKeyArgs = nil

	secret, err := tss.UpdateSecret(*secretModel)
	if err != nil {
		log.Printf("[ERROR] unable to update secret: %s", err)
		return err
	}

	err = modelToResourceSecret(secret, resourceSecret)
	if err != nil {
		log.Printf("[ERROR] error converting model to resource secret: %s", err)
		return err
	}

	log.Printf("[DEBUG] updated secret with name '%s' and ID '%d'", secret.Name, secret.ID)
	return nil
}

func resourceSecretDelete(resourceSecret *schema.ResourceData, meta interface{}) error {
	id, err := strconv.Atoi(resourceSecret.Id())
	if err != nil {
		log.Printf("[ERROR] configuration error, ID is not an integer: %s", resourceSecret.Id())
		return err
	}

	tss, err := server.New(meta.(server.Configuration))
	if err != nil {
		log.Printf("[ERROR] configuration error: %s", err)
		return err
	}
	log.Printf("[DEBUG] deleting secret with name '%s' and ID '%d'", resourceSecret.Get("name"), id)

	err = tss.DeleteSecret(id)
	if err != nil {
		log.Printf("[ERROR] unable to delete secret: %s", err)
		return err
	}

	return nil
}

func fetchTemplateForResourceSecret(resourceSecret *schema.ResourceData, server *server.Server) (*server.SecretTemplate, error) {
	secretTemplateId := resourceSecret.Get("secret_template_id").(int)
	return server.SecretTemplate(secretTemplateId)
}

func resourceSecretToModel(resourceSecret *schema.ResourceData, template *server.SecretTemplate, id int) (*server.Secret, error) {
	secret := new(server.Secret)
	var err error

	secret.ID = id
	secret.Active = resourceSecret.Get("active").(bool)
	secret.AutoChangeEnabled = resourceSecret.Get("auto_change_enabled").(bool)
	secret.CheckOutChangePasswordEnabled = resourceSecret.Get("check_out_change_password_enabled").(bool)
	secret.CheckOutEnabled = resourceSecret.Get("check_out_enabled").(bool)
	secret.CheckOutIntervalMinutes = resourceSecret.Get("check_out_interval_minutes").(int)
	secret.DelayIndexing = resourceSecret.Get("delay_indexing").(bool)
	secret.EnableInheritPermissions = resourceSecret.Get("enable_inherit_permissions").(bool)
	secret.EnableInheritSecretPolicy = resourceSecret.Get("enable_inherit_secret_policy").(bool)
	secret.FolderID = resourceSecret.Get("folder_id").(int)
	secret.Name = resourceSecret.Get("name").(string)
	secret.ProxyEnabled = resourceSecret.Get("proxy_enabled").(bool)
	secret.RequiresComment = resourceSecret.Get("requires_comment").(bool)
	secret.SecretPolicyID = resourceSecret.Get("secret_policy_id").(int)
	secret.SecretTemplateID = resourceSecret.Get("secret_template_id").(int)
	secret.SessionRecordingEnabled = resourceSecret.Get("session_recording_enabled").(bool)
	secret.SiteID = resourceSecret.Get("site_id").(int)
	secret.WebLauncherRequiresIncognitoMode = resourceSecret.Get("web_launcher_requires_incognito_mode").(bool)
	generateSshKeys := resourceSecret.Get("generate_ssh_keys").(bool)
	generateSshPassphrase := resourceSecret.Get("generate_ssh_passphrase").(bool)
	if generateSshKeys || generateSshPassphrase {
		secret.SshKeyArgs = &server.SshKeyArgs{GeneratePassphrase: generateSshPassphrase, GenerateSshKeys: generateSshKeys}
	} else {
		secret.SshKeyArgs = nil
	}

	// Iterate the configuration's item values and map them
	// into the model's fields
	resourceItems := resourceSecret.Get("item").([]interface{})
	secret.Fields = []server.SecretField{}
	initializedFields := make(map[string]string)
	for _, resourceItem := range resourceItems {
		resourceItemMap := resourceItem.(map[string]interface{})
		secretField := server.SecretField{}

		// Transfer the field name (aka 'slug'), and use it to set the field ID
		// which is required by the TSS API to bind an item to a field on the
		// secret
		var templateField *server.SecretTemplateField
		if resourceField, found := resourceItemMap["field"]; found {
			secretField.Slug = resourceField.(string)
			if templateField, found = template.GetField(secretField.Slug); found {
				secretField.FieldID = templateField.SecretTemplateFieldID
			} else {
				return nil, fmt.Errorf("[ERROR] an item on the secret named '%s' has an unrecognized field name '%s'",
					secret.Name, secretField.Slug)
			}
		} else {
			return nil, fmt.Errorf("[ERROR] an item on the secret named '%s' is missing a field name", secret.Name)
		}

		// Transfer the item value
		if resourceValue, found := resourceItemMap["value"]; found {
			if resourceFileEncoded, encodedFound := resourceItemMap["file_encoded"];
					templateField.IsFile && encodedFound && resourceFileEncoded.(bool) {
				log.Printf("[DEBUG] Decoding file from base64 before posting to Thycotic server")
				valueDecoded, err := base64.StdEncoding.DecodeString(resourceValue.(string))
				if err != nil {
					return nil, err
				}
				secretField.ItemValue = string(valueDecoded)
			} else {
				secretField.ItemValue = resourceValue.(string)
			}
		} else {
			return nil, fmt.Errorf("[ERROR] an item on the secret named '%s' is missing a value. To remove " +
				"an optional field from a secret, simply remove the item from the secret's item list in the " +
				"configuration", secret.Name)
		}

		// Transfer the filename, if present
		if resourceFilename, found := resourceItemMap["filename"]; found {
			secretField.Filename = resourceFilename.(string)
		} else {
			secretField.Filename = "File.txt"
		}

		secret.Fields = append(secret.Fields, secretField)
		initializedFields[secretField.Slug] = secretField.Slug
	}

	// Iterate the template's fields, and if any of them have not been initialized
	// above, set their value to an empty string. This approach allows the terraform 
	// user to remove an item from a secret by simply removing its entry from the 
	// "item" list in the configuration's secret declaration.
	for _, templateField := range template.Fields {
		if _, found := initializedFields[templateField.FieldSlugName]; !found {
			secretField := server.SecretField{}
			secretField.Slug = templateField.FieldSlugName
			secretField.FieldID = templateField.SecretTemplateFieldID
			secretField.ItemValue = ""
			secret.Fields = append(secret.Fields, secretField)
		}
	}

	return secret, err
}

func modelToResourceSecret(model *server.Secret, resourceSecret *schema.ResourceData) error {
	resourceSecret.SetId(strconv.Itoa(model.ID))

	var err error
	if err = resourceSecret.Set("active", model.Active); err != nil { return err }
	if err = resourceSecret.Set("auto_change_enabled", model.AutoChangeEnabled); err != nil { return err }
	if err = resourceSecret.Set("check_out_change_password_enabled", model.CheckOutChangePasswordEnabled); err != nil { return err }
	if err = resourceSecret.Set("check_out_enabled", model.CheckOutEnabled); err != nil { return err }
	if err = resourceSecret.Set("check_out_interval_minutes", model.CheckOutIntervalMinutes); err != nil { return err }
	if err = resourceSecret.Set("delay_indexing", model.DelayIndexing); err != nil { return err }
	if err = resourceSecret.Set("enable_inherit_permissions", model.EnableInheritPermissions); err != nil { return err }
	if err = resourceSecret.Set("enable_inherit_secret_policy", model.EnableInheritSecretPolicy); err != nil { return err }
	if err = resourceSecret.Set("folder_id", model.FolderID); err != nil { return err }
	if err = resourceSecret.Set("name", model.Name); err != nil { return err }
	if err = resourceSecret.Set("proxy_enabled", model.ProxyEnabled); err != nil { return err }
	if err = resourceSecret.Set("requires_comment", model.RequiresComment); err != nil { return err }
	if err = resourceSecret.Set("secret_policy_id", model.SecretPolicyID); err != nil { return err }
	if err = resourceSecret.Set("secret_template_id", model.SecretTemplateID); err != nil { return err }
	if err = resourceSecret.Set("session_recording_enabled", model.SessionRecordingEnabled); err != nil { return err }
	if err = resourceSecret.Set("site_id", model.SiteID); err != nil { return err }
	if err = resourceSecret.Set("web_launcher_requires_incognito_mode", model.WebLauncherRequiresIncognitoMode); err != nil { return err }
	// Leave generate_ssh_keys and generate_ssh_passphrase as they are in the state.
	// The server does not return these values in the response body. They are one-way
	// parameters in the Thycotic API, appearing only in the request bodies of a Create
	// Secret or Update Secret request.

	resourceItems := resourceSecret.Get("item").([]interface{})
	if model.Fields == nil {
		resourceItems = make([]interface{}, 0)
	} else {
		mappedFields := make(map[string]string)

		// First iterate items in the configuration and map in values from the model.
		// Mapping in this way preserves the order in the configuration, making state
		// comparisons more predictable.
		for _, item := range resourceItems {
			resourceItem := item.(map[string]interface{})
			var field server.SecretField
			for _, field = range model.Fields {
				if field.Slug == resourceItem["field"] { mappedFields[field.Slug] = field.Slug; break }
			}
			if &field == nil {
				return fmt.Errorf("[ERROR] an item on the secret named '%s' has an unrecognized field name '%s'", model.Name, resourceItem["field"])
			}
			mapModelFieldToResourceSecretItem(&field, resourceItem)
			// If the configuration indicates that the value is encoded, re-encode
			// the value for state maintenance
			if resourceEncoded, found := resourceItem["file_encoded"]; field.IsFile && found && resourceEncoded.(bool) {
				log.Printf("[DEBUG] Encoding file to base64 before returning to Terraform state")
				resourceItem["value"] = base64.StdEncoding.EncodeToString([]byte(resourceItem["value"].(string)))
			}
		}

		// Now iterate fields in the model. For every field that has a value and has
		// not already been mapped, add it to the resource items to pick up changes
		// made outside the configuration.
		for _, field := range model.Fields {
			slug := field.Slug
			if _, found := mappedFields[slug]; !found {
				hasValue := false
				if field.IsFile {
					hasValue = field.Filename != ""
				} else {
					hasValue = field.ItemValue != ""
				}
				if hasValue {
					resourceItem := make(map[string]interface{})
					mapModelFieldToResourceSecretItem(&field, resourceItem)
					resourceItems = append(resourceItems, resourceItem)
				}
			}
		}
	}
	if err = resourceSecret.Set("item", resourceItems); err != nil {
		return err
	}

	return nil
}

func mapModelFieldToResourceSecretItem(field *server.SecretField, resourceItem map[string]interface{}) {
	resourceItem["field"] = field.Slug
	resourceItem["value"] = field.ItemValue
	resourceItem["filename"] = field.Filename
	resourceItem["field_description"] = field.FieldDescription
	resourceItem["field_id"] = field.FieldID
	resourceItem["field_name"] = field.FieldName
	resourceItem["file_attachment_id"] = field.FileAttachmentID
	resourceItem["is_file"] = field.IsFile
	resourceItem["is_notes"] = field.IsNotes
	resourceItem["is_password"] = field.IsPassword
	resourceItem["item_id"] = field.ItemID
}

func resourceSecret() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecretCreate,
		Read:   resourceSecretRead,
		Update: resourceSecretUpdate,
		Delete: resourceSecretDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "the display name of the secret",
				Required:    true,
			},
			"secret_template_id": {
				Type:        schema.TypeInt,
				Description: "the id of the template that defines the fields for secret",
				Required:    true,
			},
			"site_id": {
				Type:        schema.TypeInt,
				Description: "the id of the distributed engine site that is used by this secret for operations such " +
					"as password changing",
				Optional:    true,
				Default:     1,
			},
			"folder_id": {
				Type:        schema.TypeInt,
				Description: "the id of the folder which contains the secret. Set to nil, to -1, or leave unset for " +
					"secrets in the root folder",
				Optional:    true,
			},
			"secret_policy_id": {
				Type:        schema.TypeInt,
				Description: "the id of the secret policy that controls the security and other settings of the secret",
				Optional:    true,
				Default:     -1,
			},
			"generate_ssh_keys": {
				Type: schema.TypeBool,
				Description: "whether to generate an SSH public/private key pair for the secret. If true, the " +
					"template for this secret must have extended mappings that support SSH keys. Also, the item " +
					"value for the mapped file fields must be undeclared or empty in this configuration",
				ForceNew: true,
				Optional: true,
				Default: false,
			},
			"generate_ssh_passphrase": {
				Type: schema.TypeBool,
				Description: "whether to generate a passphrase to protect the SSH private key. If true, " +
					"generate_ssh_keys must also be true, and the template for this secret must have extended " +
					"mappings that support SSH keys. Finally, the item value for the mapped field must be " +
					"undeclared or empty in this configuration",
				ForceNew: true,
				Optional: true,
				Default: false,
			},
			"item": {
				Type:        schema.TypeList,
				Description: "array of values for the fields defined in the secret template",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"field": {
							Type:        schema.TypeString,
							Description: "unique name for the field on the secret template, also known in the " +
								"Thycotic API as a 'slug'. Field names/slugs are used in many places to easily refer " +
								"to a field without having to know the field id",
							Required:    true,
						},
						"value": {
							Type:        schema.TypeString,
							Description: "the value for the field. If this item is a file item, you may provide " +
								"the contents of the file here directly as plain text, or you may use one of the " +
								"Terraform functions to read in the contents of a file on disk, such as " +
								"'file(\"/some/file/path.txt\")' or 'filebase64(\"/some/file/path.txt\")'. If " +
								"'generate_ssh_keys' is true, leave this blank or undefined for the mapped public " +
								"and private key file items, as any value specified here will be ignored, . Likewise, " +
								"if 'generate_ssh_passphrase' is true, leave this blank or undefined for the mapped " +
								"passphrase item, as any value specified here will be ignored upon creation, *BUT* " +
								"flagged as a diff upon update",
							Optional:    true,
							Computed:    true,
							Sensitive:   true,
						},
						"filename": {
							Type:        schema.TypeString,
							Description: "the name of the file attachment. This should be provided if this item is " +
								"a file item and is ignored otherwise. Default is 'File.txt' if a name is not " +
								"provided here. Keep in mind that the Thycotic Secret Server has a configurable " +
								"list of acceptable filename extensions, and this filename must comply with that list",
							Optional:    true,
							Computed:    true,
						},
						"file_encoded": {
							Type:        schema.TypeBool,
							Description: "whether the contents of the file attachment have been Base64 encoded and " +
								"should therefore be decoded by this plugin before posting the file contents to the " +
								"Thycotic Secret Server. You should set this to 'true' if you're using the Terraform " +
								"'filebase64()' function to read in the contents of a file that is _not_ UTF-8 " +
								"encoded, which is a requirement for the Terraform 'file()' function. This value is " +
								"ignored if this item is not a file item",
							Optional:    true,
							Default:     false,
						},
						"field_description": {
							Type:        schema.TypeString,
							Description: "longer description of the secret field",
							Computed:    true,
						},
						"field_id": {
							Type:        schema.TypeInt,
							Description: "the id of the field definition from the secret template",
							Computed:    true,
						},
						"field_name": {
							Type:        schema.TypeString,
							Description: "the display name of the secret field",
							Computed:    true,
						},
						"file_attachment_id": {
							Type:        schema.TypeInt,
							Description: "ID of the file attachment",
							Computed:    true,
						},
						"is_file": {
							Type:        schema.TypeBool,
							Description: "whether the field is a file attachment",
							Computed:    true,
						},
						"is_notes": {
							Type:        schema.TypeBool,
							Description: "whether the field is represented as a multi-line text box used for " +
								"long-form text fields",
							Computed:    true,
						},
						"is_password": {
							Type:        schema.TypeBool,
							Description: "whether the field is a password attachment",
							Computed:    true,
						},
						"item_id": {
							Type:        schema.TypeInt,
							Description: "the id of the secret field item",
							Computed:    true,
						},
					},
				},
				DiffSuppressFunc: func(key, old, new string, resourceSecret *schema.ResourceData) bool {
					if old == new { return true }

					generateSshKeys := resourceSecret.Get("generate_ssh_keys").(bool)
					generateSshPassphrase := resourceSecret.Get("generate_ssh_passphrase").(bool)
					if generateSshKeys || generateSshPassphrase {
						// If the user is generating SSH keys or passphrases, it's not necessary
						// for them to declare items on their secret resources for the public and
						// private key fields, or for the passphrase field. If they choose not to,
						// there will be state differences for the "field" attribute on those
						// undeclared items. The "old" value will have the field name since it is
						// returned by the server after POST and PUT operations and saved off in
						// the state. The "new" value, however, will remain empty as long as it's
						// undeclared in the configuration. When this happens, print off a warning
						// for the Terraform logs, and return "true" to indicate nothing's changed.
						if match, _ := regexp.Match("^item\\.\\d+\\.field$", []byte(key)); match {
							if new == "" {
								log.Printf("[WARN] ignoring state differences for the '%s' item on the secret " +
									"named '%s' since SSH generation is enabled", old, resourceSecret.Get("name"))
								return true
							}
						}
						// For the same reasons as above, ignore the state differences between the
						// old number of "item" blocks and the new number.
						if key == "item.#" {
							numOldItems, oldErr := strconv.Atoi(old)
							numNewItems, newErr := strconv.Atoi(new)
							if oldErr == nil && newErr == nil && numNewItems < numOldItems {
								log.Printf("[WARN] ignoring state differences between the number of items on " +
									"the secret named '%s' since SSH generation is enabled", resourceSecret.Get("name"))
								return true
							}
						}
					}
					return false
				},
			},
			"active": {
				Type:        schema.TypeBool,
				Description: "whether the secret is in an active or deleted state",
				Computed:    true,
			},
			"auto_change_enabled": {
				Type:        schema.TypeBool,
				Description: "whether the secret’s password is automatically rotated on a schedule, default is false",
				Optional:    true,
			},
			"check_out_change_password_enabled": {
				Type:        schema.TypeBool,
				Description: "whether the secret’s password is automatically changed when a secret is checked in, " +
					"default is false. This is a security feature that prevents use of the password " +
					"retrieved from check-out after the secret is checked in",
				Optional:    true,
			},
			"check_out_enabled": {
				Type:        schema.TypeBool,
				Description: "whether the user must check-out the secret to view it, default is false. Checking out " +
					"gives the user exclusive access to the secret for a specified period or until the " +
					"secret is checked in.",
				Optional:    true,
			},
			"check_out_interval_minutes": {
				Type:        schema.TypeInt,
				Description: "the number of minutes that a secret will remain checked out",
				Optional:    true,
				Default:     -1,
			},
			"delay_indexing": {
				Type:        schema.TypeBool,
				Description: "whether the search indexing should be delayed to the background process, default is " +
					"false. This can speed up bulk secret creation scripts by offloading the task of " +
					"indexing the new secrets to the background task at the trade-off of not having search " +
					"indexes immediately available",
				Optional:    true,
			},
			"enable_inherit_permissions": {
				Type:        schema.TypeBool,
				Description: "whether the secret inherits permissions from the containing folder, default is true",
				Optional:    true,
				Default:     true,
			},
			"enable_inherit_secret_policy": {
				Type:        schema.TypeBool,
				Description: "whether the secret policy is inherited from the containing folder, default is false",
				Optional:    true,
			},
			"proxy_enabled": {
				Type:        schema.TypeBool,
				Description: "whether sessions launched on this secret use Secret Server’s proxying or connect " +
					"directly, default is false",
				Optional:    true,
			},
			"requires_comment": {
				Type:        schema.TypeBool,
				Description: "whether the user must enter a comment to view the secret, default is false",
				Optional:    true,
			},
			"session_recording_enabled": {
				Type:        schema.TypeBool,
				Description: "whether session recording is enabled, default is false",
				Optional:    true,
			},
			"web_launcher_requires_incognito_mode": {
				Type:        schema.TypeBool,
				Description: "whether the web launcher will require the browser to run in incognito mode, default is " +
					"false",
				Optional:    true,
			},
		},
	}
}
