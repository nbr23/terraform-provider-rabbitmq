package rabbitmq

import (
	"fmt"
	"log"

	rabbithole "github.com/michaelklishin/rabbit-hole/v2"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePermissions() *schema.Resource {
	return &schema.Resource{
		Create:      CreatePermissions,
		Update:      UpdatePermissions,
		Read:        ReadPermissions,
		Delete:      DeletePermissions,
		Description: "The `rabbitmq_permissions` resource creates and manages a user's set of permissions.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"user": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The user to apply the permissions to.",
			},

			"vhost": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "/",
				ForceNew:    true,
				Description: "The vhost to create the resource in.",
			},

			"permissions": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "The permissions for the user.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"configure": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The configure permission for the user.",
						},

						"write": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The write permission for the user.",
						},

						"read": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The read permission for the user.",
						},
					},
				},
			},
		},
	}
}

func CreatePermissions(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	user := d.Get("user").(string)
	vhost := d.Get("vhost").(string)
	permsList := d.Get("permissions").([]interface{})

	permsMap := map[string]interface{}{}
	if permsList[0] != nil {
		permsMap = permsList[0].(map[string]interface{})
	}

	if err := setPermissionsIn(rmqc, vhost, user, permsMap); err != nil {
		return err
	}

	id := fmt.Sprintf("%s@%s", user, vhost)
	d.SetId(id)

	return ReadPermissions(d, meta)
}

func ReadPermissions(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	user, vhost, err := parseResourceId(d)
	if err != nil {
		return err
	}

	userPerms, err := rmqc.GetPermissionsIn(vhost, user)
	if err != nil {
		return checkDeleted(d, err)
	}

	log.Printf("[DEBUG] RabbitMQ: Permission retrieved for %s: %#v", d.Id(), userPerms)

	d.Set("user", userPerms.User)
	d.Set("vhost", userPerms.Vhost)

	perms := make([]map[string]interface{}, 1)
	p := make(map[string]interface{})
	p["configure"] = userPerms.Configure
	p["write"] = userPerms.Write
	p["read"] = userPerms.Read
	perms[0] = p
	d.Set("permissions", perms)

	return nil
}

func UpdatePermissions(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	user, vhost, err := parseResourceId(d)
	if err != nil {
		return err
	}

	if d.HasChange("permissions") {
		_, newPerms := d.GetChange("permissions")

		newPermsList := newPerms.([]interface{})
		permsMap, ok := newPermsList[0].(map[string]interface{})
		if !ok {
			return fmt.Errorf("Unable to parse permissions")
		}

		if err := setPermissionsIn(rmqc, vhost, user, permsMap); err != nil {
			return err
		}
	}

	return ReadPermissions(d, meta)
}

func DeletePermissions(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	user, vhost, err := parseResourceId(d)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] RabbitMQ: Attempting to delete permission for %s", d.Id())

	resp, err := rmqc.ClearPermissionsIn(vhost, user)
	log.Printf("[DEBUG] RabbitMQ: Permission delete response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		// The permissions were already deleted
		return nil
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error deleting RabbitMQ permission: %s", resp.Status)
	}

	return nil
}

func setPermissionsIn(rmqc *rabbithole.Client, vhost string, user string, permsMap map[string]interface{}) error {
	perms := rabbithole.Permissions{}

	if v, ok := permsMap["configure"].(string); ok {
		perms.Configure = v
	}

	if v, ok := permsMap["write"].(string); ok {
		perms.Write = v
	}

	if v, ok := permsMap["read"].(string); ok {
		perms.Read = v
	}

	log.Printf("[DEBUG] RabbitMQ: Attempting to set permissions for %s@%s: %#v", user, vhost, perms)

	resp, err := rmqc.UpdatePermissionsIn(vhost, user, perms)
	log.Printf("[DEBUG] RabbitMQ: Permission response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error setting permissions: %s", resp.Status)
	}

	return nil
}
