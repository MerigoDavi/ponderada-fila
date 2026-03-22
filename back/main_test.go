package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	// Setup just the router ignoring RabbitMQ for simple payload validation testing
	router := gin.Default()
	
	router.POST("/telemetry", func(c *gin.Context) {
		var payload TelemetryData

		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido ou campos faltando"})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{"message": "Dados na fila"})
	})

	return router
}

func TestTelemetryEndpoint_Success(t *testing.T) {
	router := setupTestRouter()

	// Criação de um payload JSON válido
	body := []byte(`{
		"device_id": "sensor-123",
		"sensor_type": "temperatura",
		"nature": "analog",
		"value": 42.5
	}`)

	req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Esperava status 202, obteve %d", w.Code)
	}
}

func TestTelemetryEndpoint_Failure_MissingFields(t *testing.T) {
	router := setupTestRouter()

	// JSON inválido faltando o sensor_type que é required
	body := []byte(`{
		"device_id": "sensor-123",
		"nature": "analog",
		"value": 42.5
	}`)

	req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// O esperado é 400 Bad Request devido à validação falhar
	if w.Code != http.StatusBadRequest {
		t.Errorf("Esperava status 400, obteve %d", w.Code)
	}
}

func TestTelemetryEndpoint_Failure_InvalidNature(t *testing.T) {
	router := setupTestRouter()

	// Nature deve ser "discrete" ou "analog", vamos passar algo errado
	body := []byte(`{
		"device_id": "sensor-123",
		"sensor_type": "umidade",
		"nature": "INVALID_NATURE",
		"value": 42.5
	}`)

	req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Esperava status 400 por erro de validação (nature inválida), obteve %d", w.Code)
	}
}
