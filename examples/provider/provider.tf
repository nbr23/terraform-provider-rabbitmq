# Configure the RabbitMQ provider
provider "rabbitmq" {
  endpoint = "http://127.0.0.1"
  username = "guest"
  password = "guest"
}

# Create a virtual host
resource "rabbitmq_vhost" "vhost_1" {
  name = "vhost_1"
}
