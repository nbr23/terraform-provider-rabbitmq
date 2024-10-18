package rabbitmq

import (
	"fmt"
	"log"

	rabbithole "github.com/michaelklishin/rabbit-hole/v2"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceExchange() *schema.Resource {
	return &schema.Resource{
		Create:      CreateExchange,
		Read:        ReadExchange,
		Delete:      DeleteExchange,
		Description: "The `rabbitmq_exchange` resource creates and manages an exchange in a RabbitMQ server.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the exchange.",
			},

			"vhost": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "/",
				ForceNew:    true,
				Description: "The vhost to create the resource in.",
			},

			"settings": {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				Description: "The settings for the exchange.",
				MaxItems:    1,
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

func CreateExchange(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	name := d.Get("name").(string)
	vhost := d.Get("vhost").(string)
	settingsList := d.Get("settings").([]interface{})

	settingsMap, ok := settingsList[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unable to parse settings")
	}

	if err := declareExchange(rmqc, vhost, name, settingsMap); err != nil {
		return err
	}

	id := fmt.Sprintf("%s@%s", name, vhost)
	d.SetId(id)

	return ReadExchange(d, meta)
}

func ReadExchange(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	name, vhost, err := parseResourceId(d)
	if err != nil {
		return err
	}

	exchangeSettings, err := rmqc.GetExchange(vhost, name)
	if err != nil {
		return checkDeleted(d, err)
	}

	log.Printf("[DEBUG] RabbitMQ: Exchange retrieved %s: %#v", d.Id(), exchangeSettings)

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

	return nil
}

func DeleteExchange(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	name, vhost, err := parseResourceId(d)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] RabbitMQ: Attempting to delete exchange %s", d.Id())

	resp, err := rmqc.DeleteExchange(vhost, name)
	log.Printf("[DEBUG] RabbitMQ: Exchange delete response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		// The exchange was automatically deleted
		return nil
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error deleting RabbitMQ exchange: %s", resp.Status)
	}

	return nil
}

func declareExchange(rmqc *rabbithole.Client, vhost string, name string, settingsMap map[string]interface{}) error {
	exchangeSettings := rabbithole.ExchangeSettings{}

	if v, ok := settingsMap["type"].(string); ok {
		exchangeSettings.Type = v
	}

	if v, ok := settingsMap["durable"].(bool); ok {
		exchangeSettings.Durable = v
	}

	if v, ok := settingsMap["auto_delete"].(bool); ok {
		exchangeSettings.AutoDelete = v
	}

	if v, ok := settingsMap["arguments"].(map[string]interface{}); ok {
		exchangeSettings.Arguments = v
	}

	log.Printf("[DEBUG] RabbitMQ: Attempting to declare exchange %s@%s: %#v", name, vhost, exchangeSettings)

	resp, err := rmqc.DeclareExchange(vhost, name, exchangeSettings)
	log.Printf("[DEBUG] RabbitMQ: Exchange declare response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error declaring RabbitMQ exchange: %s", resp.Status)
	}

	return nil
}
