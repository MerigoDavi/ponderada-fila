package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type TelemetryData struct {
	ID         uuid.UUID `json:"id"`
	DeviceID   string    `json:"device_id" binding:"required"`
	SensorType string    `json:"sensor_type" binding:"required"`
	Nature     string    `json:"nature" binding:"required,oneof=discrete analog"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
}

func main() {
	router := gin.Default()

	// Pega a URL do RabbitMQ nas variáveis de ambiente
	rabbitURL := os.Getenv("RABBIT_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	// Conecta ao RabbitMQ de forma direta (o Docker garante que ele suba antes)
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Falha ao conectar no RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Falha ao abrir o canal: %v", err)
	}
	defer ch.Close()

	// Declara a fila
	q, err := ch.QueueDeclare("telemetry", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Falha ao declarar a fila: %v", err)
	}

	router.POST("/telemetry", func(c *gin.Context) {
		var payload TelemetryData

		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido ou campos faltando"})
			return
		}
		
		// Preenche campos automáticos que o sensor não mandou
		payload.ID = uuid.New()
		if payload.Timestamp.IsZero() {
			payload.Timestamp = time.Now()
		}
		
		// Converte a struct para JSON para enviar para a fila
		body, err := json.Marshal(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao processar dados"})
			return
		}

		err = ch.Publish("", q.Name, false, false, amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao enviar para a fila"})
			return
		}

		// Retorna 202 dizendo que foi pra fila
		c.JSON(http.StatusAccepted, gin.H{"message": "Dados na fila", "id": payload.ID})
	})

	router.Run(":8080")
}
