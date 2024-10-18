package rabbitmq

import (
	"context"
	"fmt"
	"log"

	rabbithole "github.com/michaelklishin/rabbit-hole/v2"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcesExchange() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcesReadExchange,
		Description: "The `rabbitmq_exchange` data source retrieves information about an exchange.",
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The id of the exchange. This is a combination of the name and vhost.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the exchange.",
			},
			"vhost": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "/",
				Description: "The vhost of the exchange.",
			},
			"settings": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The type of the exchange.",
						},

						"durable": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Whether the exchange is durable or not.",
						},

						"auto_delete": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Whether the exchange is auto-deleted when no longer in use.",
						},

						"arguments": {
							Type:        schema.TypeMap,
							Optional:    true,
							Description: "Additional key/value settings for the exchange.",
						},
					},
				},
			},
		},
	}
}

func dataSourcesReadExchange(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	rmqc := meta.(*rabbithole.Client)

	name := d.Get("name").(string)
	vhost := d.Get("vhost").(string)
	id := fmt.Sprintf("%s@%s", name, vhost)

	exchangeSettings, err := rmqc.GetExchange(vhost, name)
	if err != nil {
		return diag.FromErr(checkDeleted(d, err))
	}

	log.Printf("[DEBUG] RabbitMQ: Exchange retrieved %s: %#v", id, exchangeSettings)

	d.Set("name", exchangeSettings.Name)
	d.Set("vhost", exchangeSettings.Vhost)

	exchange := make([]map[string]interface{}, 1)
	e := make(map[string]interface{})
	e["type"] = exchangeSettings.Type
	e["durable"] = exchangeSettings.Durable
	e["auto_delete"] = exchangeSettings.AutoDelete
	e["arguments"] = exchangeSettings.Arguments
	exchange[0] = e
	d.Set("settings", exchange)

	d.SetId(id)

	return diags
}
