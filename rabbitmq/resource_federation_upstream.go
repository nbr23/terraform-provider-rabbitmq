package rabbitmq

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	rabbithole "github.com/michaelklishin/rabbit-hole/v2"
)

func resourceFederationUpstream() *schema.Resource {
	return &schema.Resource{
		Create:      CreateFederationUpstream,
		Read:        ReadFederationUpstream,
		Update:      UpdateFederationUpstream,
		Delete:      DeleteFederationUpstream,
		Description: "The `rabbitmq_federation_upstream` resource creates and manages a federation upstream parameter.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the federation upstream.",
			},

			"vhost": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The vhost to create the resource in.",
			},

			// "federation-upstream"
			"component": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Set to `federation-upstream` by the underlying RabbitMQ provider. You do not set this attribute but will see it in state and plan output.",
			},

			"definition": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "The configuration of the federation upstream.",
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// applicable to both federated exchanges and queues
						"uri": {
							Type:        schema.TypeString,
							Required:    true,
							Sensitive:   true,
							Description: "The AMQP URI(s) for the upstream. Note that the URI may contain sensitive information, such as a password.",
						},

						"prefetch_count": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     1000,
							Description: "Maximum number of unacknowledged messages that may be in flight over a federation link at one time.",
						},

						"reconnect_delay": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     5,
							Description: "Time in seconds to wait after a network link goes down before attempting reconnection.",
						},

						"ack_mode": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "on-confirm",
							Description: "Determines how the link should acknowledge messages. Valid values are `on-confirm`, `on-publish`, and `no-ack`.",
							ValidateFunc: validation.StringInSlice([]string{
								"on-confirm",
								"on-publish",
								"no-ack",
							}, false),
						},

						"trust_user_id": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Determines how federation should interact with the validated user-id feature.",
						},
						// applicable to federated exchanges only
						"exchange": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The name of the upstream exchange. This is only applicable to federated exchanges.",
						},

						"max_hops": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     1,
							Description: "Maximum number of federation links that messages can traverse before being dropped. This is only applicable to federated exchanges.",
						},

						"expires": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "The expiry time (in milliseconds) after which an upstream queue for a federated exchange may be deleted if a connection to the upstream is lost. This is only applicable to federated exchanges.",
						},

						"message_ttl": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "The expiry time (in milliseconds) for messages in the upstream queue for a federated exchange (see expires). This is only applicable to federated exchanges.",
						},
						// applicable to federated queues only
						"queue": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The name of the upstream queue. This is only applicable to federated queues.",
						},
					},
				},
			},
		},
	}
}

func CreateFederationUpstream(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	name := d.Get("name").(string)
	vhost := d.Get("vhost").(string)
	defList := d.Get("definition").([]interface{})

	defMap, ok := defList[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unable to parse federation upstream definition")
	}

	if err := putFederationUpstream(rmqc, vhost, name, defMap); err != nil {
		return err
	}

	id := fmt.Sprintf("%s@%s", name, vhost)
	d.SetId(id)

	return ReadFederationUpstream(d, meta)
}

func ReadFederationUpstream(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	name, vhost, err := parseResourceId(d)
	if err != nil {
		return err
	}

	upstream, err := rmqc.GetFederationUpstream(vhost, name)
	if err != nil {
		return checkDeleted(d, err)
	}

	log.Printf("[DEBUG] RabbitMQ: Federation upstream retrieved for %s: %#v", d.Id(), upstream)

	d.Set("name", upstream.Name)
	d.Set("vhost", upstream.Vhost)
	d.Set("component", upstream.Component)

	var uri string
	if len(upstream.Definition.Uri) > 0 {
		uri = upstream.Definition.Uri[0]
	}
	defMap := map[string]interface{}{
		"uri":             uri,
		"prefetch_count":  upstream.Definition.PrefetchCount,
		"reconnect_delay": upstream.Definition.ReconnectDelay,
		"ack_mode":        upstream.Definition.AckMode,
		"trust_user_id":   upstream.Definition.TrustUserId,
		"exchange":        upstream.Definition.Exchange,
		"max_hops":        upstream.Definition.MaxHops,
		"expires":         upstream.Definition.Expires,
		"message_ttl":     upstream.Definition.MessageTTL,
		"queue":           upstream.Definition.Queue,
	}

	defList := [1]map[string]interface{}{defMap}
	d.Set("definition", defList)

	return nil
}

func UpdateFederationUpstream(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	name, vhost, err := parseResourceId(d)
	if err != nil {
		return err
	}

	if d.HasChange("definition") {
		_, newDef := d.GetChange("definition")

		defList := newDef.([]interface{})
		defMap, ok := defList[0].(map[string]interface{})
		if !ok {
			return fmt.Errorf("Unable to parse federation definition")
		}

		if err := putFederationUpstream(rmqc, vhost, name, defMap); err != nil {
			return err
		}
	}

	return ReadFederationUpstream(d, meta)
}

func DeleteFederationUpstream(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	name, vhost, err := parseResourceId(d)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] RabbitMQ: Attempting to delete federation upstream for %s", d.Id())

	resp, err := rmqc.DeleteFederationUpstream(vhost, name)
	log.Printf("[DEBUG] RabbitMQ: Federation upstream delete response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		// the upstream was automatically deleted
		return nil
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error deleting RabbitMQ federation upstream: %s", resp.Status)
	}

	return nil
}

func putFederationUpstream(rmqc *rabbithole.Client, vhost string, name string, defMap map[string]interface{}) error {
	definition := rabbithole.FederationDefinition{}

	log.Printf("[DEBUG] RabbitMQ: Attempting to put federation definition for %s@%s: %#v", name, vhost, defMap)

	if v, ok := defMap["uri"].(string); ok {
		definition.Uri = []string{v}
	}

	if v, ok := defMap["expires"].(int); ok {
		definition.Expires = v
	}

	if v, ok := defMap["message_ttl"].(int); ok {
		definition.MessageTTL = int32(v)
	}

	if v, ok := defMap["max_hops"].(int); ok {
		definition.MaxHops = v
	}

	if v, ok := defMap["prefetch_count"].(int); ok {
		definition.PrefetchCount = v
	}

	if v, ok := defMap["reconnect_delay"].(int); ok {
		definition.ReconnectDelay = v
	}

	if v, ok := defMap["ack_mode"].(string); ok {
		definition.AckMode = v
	}

	if v, ok := defMap["trust_user_id"].(bool); ok {
		definition.TrustUserId = v
	}

	if v, ok := defMap["exchange"].(string); ok {
		definition.Exchange = v
	}

	if v, ok := defMap["queue"].(string); ok {
		definition.Queue = v
	}

	log.Printf("[DEBUG] RabbitMQ: Attempting to declare federation upstream for %s@%s: %#v", name, vhost, definition)

	resp, err := rmqc.PutFederationUpstream(vhost, name, definition)
	log.Printf("[DEBUG] RabbitMQ: Federation upstream declare response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error creating RabbitMQ federation upstream: %s", resp.Status)
	}

	return nil
}
