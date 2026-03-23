package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type TelemetryData struct {
	ID         uuid.UUID `json:"id"`
	DeviceID   string    `json:"device_id"`
	SensorType string    `json:"sensor_type"`
	Nature     string    `json:"nature"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
}

func main() {
	log.Println("Iniciando o Consumer...")

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost:5432/app_db?sslmode=disable"
	}

	// Conexão direta com o Banco de Dados
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Erro ao abrir banco de dados: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("Erro ao conectar no banco de dados: %v", err)
	}

	rabbitURL := os.Getenv("RABBIT_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	// Conexão direta com o RabbitMQ
	rabConn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Erro ao conectar no RabbitMQ: %v", err)
	}
	defer rabConn.Close()

	rabCh, err := rabConn.Channel()
	if err != nil {
		log.Fatalf("Erro ao abrir o canal do RabbitMQ: %v", err)
	}
	defer rabCh.Close()

	q, err := rabCh.QueueDeclare("telemetry", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Falha ao declarar a fila: %v", err)
	}

	// Começa a consumir da fila
	msgs, err := rabCh.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Falha ao registrar o consumidor: %v", err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var t TelemetryData
			err := json.Unmarshal(d.Body, &t)
			if err != nil {
				log.Printf("Erro lendo JSON: %v", err)
				d.Ack(false)
				continue
			}

			// Insere no banco
			_, err = db.Exec(
				"INSERT INTO telemetry (id, device_id, sensor_type, nature, value, timestamp) VALUES ($1, $2, $3, $4, $5, $6)",
				t.ID, t.DeviceID, t.SensorType, t.Nature, t.Value, t.Timestamp,
			)

			if err != nil {
				log.Printf("Erro inserindo no DB: %v", err)
				d.Nack(false, true) // Devolve pra fila se o DB falhar
			} else {
				log.Printf("Telemetria salva! Dispositivo: %s", t.DeviceID)
				d.Ack(false) // Confirma que deu tudo certo e remove da fila
			}
		}
	}()

	log.Printf(" Esperando mensagens...")
	<-forever
}
