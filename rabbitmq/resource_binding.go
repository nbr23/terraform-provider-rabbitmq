package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	rabbithole "github.com/michaelklishin/rabbit-hole/v2"
)

func resourceBinding() *schema.Resource {
	return &schema.Resource{
		Create:      CreateBinding,
		Read:        ReadBinding,
		Delete:      DeleteBinding,
		Description: "The `rabbitmq_binding` resource creates and manages a binding relationship between a queue an exchange.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"source": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The source exchange.",
			},

			"vhost": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The vhost to create the resource in.",
			},

			"destination": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The destination queue or exchange.",
			},

			"destination_type": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The type of the destination. Must be either `queue` or `exchange`.",
			},

			"properties_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A unique key to refer to the binding.",
			},

			"routing_key": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "A routing key for the binding.",
			},

			"arguments": {
				Type:          schema.TypeMap,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"arguments_json"},
				Description:   "Additional key/value arguments for the binding. Conflicts with `arguments_json`",
			},
			"arguments_json": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				ValidateFunc:     validation.StringIsJSON,
				ConflictsWith:    []string{"arguments"},
				DiffSuppressFunc: structure.SuppressJsonDiff,
				Description:      "Additional key/value arguments for the binding in JSON format. Conflicts with `arguments`",
			},
		},
	}
}

func CreateBinding(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	vhost := d.Get("vhost").(string)
	arguments := d.Get("arguments").(map[string]interface{})

	// If arguments_json is used, unmarshal it into a generic interface
	// and use it as the "arguments" key for the binding.
	if v, ok := d.Get("arguments_json").(string); ok && v != "" {
		var arguments_json map[string]interface{}
		err := json.Unmarshal([]byte(v), &arguments_json)
		if err != nil {
			return err
		}

		arguments = arguments_json
	}

	bindingInfo := rabbithole.BindingInfo{
		Source:          d.Get("source").(string),
		Destination:     d.Get("destination").(string),
		DestinationType: d.Get("destination_type").(string),
		RoutingKey:      d.Get("routing_key").(string),
		Arguments:       arguments,
	}

	propertiesKey, err := declareBinding(rmqc, vhost, bindingInfo)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] RabbitMQ: Binding properties key: %s", propertiesKey)
	bindingInfo.PropertiesKey = propertiesKey
	name := fmt.Sprintf("%s/%s/%s/%s/%s", percentEncodeSlashes(vhost), bindingInfo.Source, bindingInfo.Destination, bindingInfo.DestinationType, bindingInfo.PropertiesKey)
	d.SetId(name)

	return ReadBinding(d, meta)
}

func ReadBinding(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	log.Printf("[TRACE] RabbitMQ: read binding resource ID (pre-split): %s", d.Id())
	bindingId := strings.Split(d.Id(), "/")
	log.Printf("[DEBUG] RabbitMQ: binding ID: %#v", bindingId)
	if len(bindingId) < 5 {
		return fmt.Errorf("Unable to determine binding ID")
	}

	vhost := percentDecodeSlashes(bindingId[0])
	source := bindingId[1]
	destination := bindingId[2]
	destinationType := bindingId[3]
	propertiesKey := bindingId[4]
	log.Printf("[DEBUG] RabbitMQ: Attempting to find binding for: vhost=%s source=%s destination=%s destinationType=%s propertiesKey=%s",
		vhost, source, destination, destinationType, propertiesKey)

	var bindings []rabbithole.BindingInfo
	var err error
	if destinationType == "queue" {
		bindings, err = rmqc.ListQueueBindingsBetween(vhost, source, destination)
		if err != nil {
			return err
		}
	} else if destinationType == "exchange" {
		bindings, err = rmqc.ListExchangeBindingsBetween(vhost, source, destination)
		if err != nil {
			return err
		}
	} else {
		bindings, err = rmqc.ListBindingsIn(vhost)
		if err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] RabbitMQ: Bindings retrieved: %#v", bindings)
	bindingFound := false
	for _, binding := range bindings {
		log.Printf("[TRACE] RabbitMQ: Assessing binding: %#v", binding)
		if binding.Source == source && binding.Destination == destination && binding.DestinationType == destinationType && binding.PropertiesKey == propertiesKey {
			log.Printf("[DEBUG] RabbitMQ: Found Binding: %#v", binding)
			bindingFound = true

			d.Set("vhost", binding.Vhost)
			d.Set("source", binding.Source)
			d.Set("destination", binding.Destination)
			d.Set("destination_type", binding.DestinationType)
			d.Set("routing_key", binding.RoutingKey)
			d.Set("properties_key", binding.PropertiesKey)

			if v, ok := d.Get("arguments_json").(string); ok && v != "" {
				bytes, err := json.Marshal(binding.Arguments)
				if err != nil {
					return fmt.Errorf("could not encode arguments as JSON: %w", err)
				}
				d.Set("arguments_json", string(bytes))
			} else {
				d.Set("arguments", binding.Arguments)
			}
		}
	}

	// The binding could not be found,
	// so consider it deleted and remove from state
	if !bindingFound {
		d.SetId("")
	}

	return nil
}

func DeleteBinding(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	bindingId := strings.Split(d.Id(), "/")
	if len(bindingId) < 5 {
		return fmt.Errorf("Unable to determine binding ID")
	}

	vhost := percentDecodeSlashes(bindingId[0])
	source := bindingId[1]
	destination := bindingId[2]
	destinationType := bindingId[3]
	propertiesKey := bindingId[4]

	bindingInfo := rabbithole.BindingInfo{
		Vhost:           vhost,
		Source:          source,
		Destination:     destination,
		DestinationType: destinationType,
		PropertiesKey:   propertiesKey,
	}

	log.Printf("[DEBUG] RabbitMQ: Attempting to delete binding for: vhost=%s source=%s destination=%s destinationType=%s propertiesKey=%s",
		vhost, source, destination, destinationType, propertiesKey)

	resp, err := rmqc.DeleteBinding(vhost, bindingInfo)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] RabbitMQ: Binding delete response: %#v", resp)

	if resp.StatusCode == 404 {
		// The binding was already deleted
		return nil
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error deleting RabbitMQ binding: %s", resp.Status)
	}

	return nil
}

func declareBinding(rmqc *rabbithole.Client, vhost string, bindingInfo rabbithole.BindingInfo) (string, error) {
	log.Printf("[DEBUG] RabbitMQ: Attempting to declare binding for: vhost=%s source=%s destination=%s destinationType=%s",
		vhost, bindingInfo.Source, bindingInfo.Destination, bindingInfo.DestinationType)

	resp, err := rmqc.DeclareBinding(vhost, bindingInfo)
	log.Printf("[DEBUG] RabbitMQ: Binding declare response: %#v", resp)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("Error declaring RabbitMQ binding: %s", resp.Status)
	}

	location := strings.Split(resp.Header.Get("Location"), "/")
	propertiesKey, err := url.PathUnescape(location[len(location)-1])

	if err != nil {
		return "", err
	}

	return propertiesKey, nil
}
