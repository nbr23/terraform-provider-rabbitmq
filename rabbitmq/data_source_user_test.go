package rabbitmq

import (
	"fmt"
	"testing"

	rabbithole "github.com/michaelklishin/rabbit-hole/v2"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceUser(t *testing.T) {
	resourceName := "rabbitmq_user.test"
	dataSourceName := "data.rabbitmq_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceUserConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserExists(resourceName),
					testAccCheckDataSourceUserExists(dataSourceName),
					resource.TestCheckResourceAttr(dataSourceName, "name", "test"),
					resource.TestCheckResourceAttr(dataSourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "tags.0", "administrator"),
					resource.TestCheckResourceAttr(dataSourceName, "tags.1", "management"),
				),
			},
		},
	})
}

func testAccCheckDataSourceUserExists(n string) resource.TestCheckFunc {
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

		user, err := rmqc.GetUser(name)
		if err != nil {
			return fmt.Errorf("Error retrieving user: %s", err)
		}

		if user.Name != name {
			return fmt.Errorf("User not found")
		}

		return nil
	}
}

func testAccCheckUserExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("user id not set")
		}

		rmqc := testAccProvider.Meta().(*rabbithole.Client)
		name := rs.Primary.Attributes["name"]

		userInfo, err := rmqc.GetUser(name)
		if err != nil {
			return fmt.Errorf("Error retrieving user: %s", err)
		}

		if userInfo.Name != name {
			return fmt.Errorf("User not found")
		}

		return nil
	}
}

const testAccDataSourceUserConfig = `
resource "rabbitmq_user" "test" {
    name     = "test"
    password = "password"
    tags     = ["administrator", "management"]
}

data "rabbitmq_user" "test" {
    name = rabbitmq_user.test.name
}
`
