package rabbitmq

import (
	"fmt"
	"testing"

	rabbithole "github.com/michaelklishin/rabbit-hole/v2"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceVhost(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVhostConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVhostExists("rabbitmq_vhost.test"),
					testAccCheckDataSourceVhostExists("data.rabbitmq_vhost.test"),
					resource.TestCheckResourceAttr("data.rabbitmq_vhost.test", "name", "test"),
					resource.TestCheckResourceAttr("data.rabbitmq_vhost.test", "id", "test"),
				),
			},
		},
	})
}

func testAccCheckDataSourceVhostExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("data source not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("data source ID is not set")
		}

		rmqc := testAccProvider.Meta().(*rabbithole.Client)
		vhost, err := rmqc.GetVhost(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving vhost: %s", err)
		}

		if vhost.Name != rs.Primary.ID {
			return fmt.Errorf("Vhost not found")
		}

		return nil
	}
}

func testAccCheckVhostExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("vhost id not set")
		}

		rmqc := testAccProvider.Meta().(*rabbithole.Client)
		vhost, err := rmqc.GetVhost(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving vhost: %s", err)
		}

		if vhost.Name != rs.Primary.ID {
			return fmt.Errorf("Vhost not found")
		}

		return nil
	}
}

const testAccDataSourceVhostConfig = `
resource "rabbitmq_vhost" "test" {
    name = "test"
}

data "rabbitmq_vhost" "test" {
    name = rabbitmq_vhost.test.name
}
`
