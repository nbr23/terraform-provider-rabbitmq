package rabbitmq

import (
	"fmt"
	"testing"

	rabbithole "github.com/michaelklishin/rabbit-hole/v2"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceExchange(t *testing.T) {
	resourceName := "rabbitmq_exchange.test"
	dataSourceName := "data.rabbitmq_exchange.test"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceExchangeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckExchangeExists(resourceName),
					testAccCheckDataSourceExchangeExists(dataSourceName),
					resource.TestCheckResourceAttr(dataSourceName, "name", "test"),
					resource.TestCheckResourceAttr(dataSourceName, "vhost", "testvhost"),
					resource.TestCheckResourceAttr(dataSourceName, "settings.0.type", "fanout"),
					resource.TestCheckResourceAttr(dataSourceName, "settings.0.durable", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "settings.0.auto_delete", "false"),
				),
			},
		},
	})
}

func testAccCheckDataSourceExchangeExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("data source not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("data source ID is not set")
		}

		rmqc := testAccProvider.Meta().(*rabbithole.Client)
		name := rs.Primary.Attributes["name"]
		vhost := rs.Primary.Attributes["vhost"]

		exchange, err := rmqc.GetExchange(vhost, name)
		if err != nil {
			return fmt.Errorf("Error retrieving exchange: %s", err)
		}

		if exchange.Name != name {
			return fmt.Errorf("Exchange not found")
		}

		return nil
	}
}

func testAccCheckExchangeExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("exchange id not set")
		}

		rmqc := testAccProvider.Meta().(*rabbithole.Client)
		name := rs.Primary.Attributes["name"]
		vhost := rs.Primary.Attributes["vhost"]

		exchangeInfo, err := rmqc.GetExchange(vhost, name)
		if err != nil {
			return fmt.Errorf("Error retrieving exchange: %s", err)
		}

		if exchangeInfo.Name != name {
			return fmt.Errorf("Exchange not found")
		}

		return nil
	}
}

const testAccDataSourceExchangeConfig = `
resource "rabbitmq_vhost" "test" {
    name = "testvhost"
}

resource "rabbitmq_exchange" "test" {
    name  = "test"
    vhost = rabbitmq_vhost.test.name
    settings {
        type        = "fanout"
        durable     = true
        auto_delete = false
    }
}

data "rabbitmq_exchange" "test" {
    name  = rabbitmq_exchange.test.name
    vhost = rabbitmq_exchange.test.vhost
}
`
