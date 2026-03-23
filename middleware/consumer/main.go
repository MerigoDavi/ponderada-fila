package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// TaskPayload is a shared structure consumed from RabbitMQ.
type TaskPayload struct {
	ID      uuid.UUID `json:"id"`
	Action  string    `json:"action"`
	Payload string    `json:"payload"`
}

type TaskMessage struct {
	Task TaskPayload `json:"task"`
	Time time.Time   `json:"time"`
}

func rabbitURL() string {
	if v := os.Getenv("RABBIT_URL"); v != "" {
		return v
	}
	return "amqp://guest:guest@localhost:5672/"
}

func main() {
	conn, err := amqp.Dial(rabbitURL())
	if err != nil {
		log.Fatalf("consumer dial: %v", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("consumer channel: %v", err)
	}
	defer ch.Close()

	msgs, err := ch.Consume("tasks", "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("consume queue: %v", err)
	}

	log.Println("consumer ready")
	for d := range msgs {
		var msg TaskMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("unmarshal: %v", err)
			continue
		}
		log.Printf("received %s action=%s", msg.Task.ID, msg.Task.Action)
		// TODO: integrate with Postgres, emit logs
		time.Sleep(500 * time.Millisecond)
	}
	log.Println("consumer exiting")
}
