package rabbit

import (
	"log"
	"os"
	"time"

	"github.com/streadway/amqp"
)

var (
	RabbitConn *amqp.Connection
	RabbitChan *amqp.Channel
)

func InitRabbit() error {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@rabbitmq:5672/"
	}
	var err error
	for i := 0; i < 10; i++ {
		RabbitConn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("RabbitMQ connection failed: %v, retrying...", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return err
	}
	RabbitChan, err = RabbitConn.Channel()
	if err != nil {
		return err
	}
	return nil
}

func CloseRabbit() {
	if RabbitChan != nil {
		RabbitChan.Close()
	}
	if RabbitConn != nil {
		RabbitConn.Close()
	}
}
