package rabbitmq

import (
	"context"
	"log"

	rabbithole "github.com/michaelklishin/rabbit-hole/v2"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcesUser() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcesReadUser,
		Description: "The `rabbitmq_user` data source retrieves information about a user.",
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The id of the user.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the user.",
			},
			"tags": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "The tags of the user.",
			},
		},
	}
}

func dataSourcesReadUser(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	rmqc := meta.(*rabbithole.Client)

	name := d.Get("name").(string)

	user, err := rmqc.GetUser(name)
	if err != nil {
		return diag.FromErr(checkDeleted(d, err))
	}

	log.Printf("[DEBUG] RabbitMQ: User retrieved: %#v", user)

	d.Set("name", user.Name)

	if len(user.Tags) > 0 {
		var tagList []string
		for _, v := range user.Tags {
			if v != "" {
				tagList = append(tagList, v)
			}
		}
		if len(tagList) > 0 {
			d.Set("tags", tagList)
		}
	}

	d.SetId(name)

	return diags
}
