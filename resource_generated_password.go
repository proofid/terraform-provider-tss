package main

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/lang/funcs"
	"github.com/thycotic/tss-sdk-go/server"
	"log"
)

func resourceGeneratedPasswordCreate(d *schema.ResourceData, meta interface{}) error {

	templateId := d.Get("template_id").(int)
	field := d.Get("field").(string)

	log.Printf("[DEBUG] generating password for the '%s' field on template with id '%d'", field, templateId)

	tss, err := server.New(meta.(server.Configuration))
	if err != nil {
		log.Printf("[ERROR] configuration error: %s", err)
		return err
	}

	template, err := tss.SecretTemplate(templateId)
	if err != nil {
		log.Printf("[ERROR] unable to retrieve the template with ID '%d': %s", templateId, err)
		return err
	}

	generatedPassword, err := tss.GeneratePassword(field, template)
	if err != nil {
		log.Printf("[ERROR] unable to generate a password for the '%s' field on the template with ID '%d': %s",
			field, templateId, err)
		return err
	}

	if err := d.Set("value", generatedPassword); err != nil {
		log.Printf("[ERROR] unable to save the password value to state for the '%s' field on the template " +
			"with ID '%d': %s", field, templateId, err)
		return err
	}
	val, _ := funcs.UUID()
	d.SetId(val.AsString())

	return nil
}

// resourceGeneratedPasswordOther is a no-op method for all the CRUD methods except 'create'. Since passwords generated
// by the Thycotic SDK are ephemeral, we don't want to re-read or update the password, and we don't need to delete it.
//
// This resource probably would have been better modelled as a data source, but a data source has only a read
// method, and since a data source has no recorded state (outside its configuration), each read would produce a new
// value. This would eliminate a generated password's utility as a persistent datum.
func resourceGeneratedPasswordOther(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceGeneratedPassword() *schema.Resource {
	return &schema.Resource{
		Create: resourceGeneratedPasswordCreate,
		Read:   resourceGeneratedPasswordOther,
		Update: resourceGeneratedPasswordOther,
		Delete: resourceGeneratedPasswordOther,

		Schema: map[string]*schema.Schema{
			"value": {
				Computed:    true,
				Description: "the value of the generated password",
				Sensitive:   true,
				Type:        schema.TypeString,
			},
			"field": {
				Description: "the name (aka: slug) of the field for which the password is generated. This field must " +
					"be declared on the secret template as a password field",
				Required:    true,
				Type:        schema.TypeString,
			},
			"template_id": {
				Description: "the id of the secret template which contains the password field, and which declares " +
					"minimum password requirements for that field",
				Required:    true,
				Type:        schema.TypeInt,
			},
		},
	}
}
