package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// O teste unitário do consumer valida o processamento primário da mensagem
// sem precisar levantar instâncias Docker no meio do teste de CI.
func TestDecodeTelemetryJson_Success(t *testing.T) {
	sampleID := uuid.New()
	sampleTime := time.Now()

	// Simulando o JSON provindo do RabbitMQ
	payload := map[string]interface{}{
		"id":          sampleID.String(),
		"device_id":   "sensor-xyz",
		"sensor_type": "velocidade",
		"nature":      "analog",
		"value":       90.5,
		"timestamp":   sampleTime.Format(time.RFC3339Nano),
	}

	bodyBytes, _ := json.Marshal(payload)

	var telemetry TelemetryData
	err := json.Unmarshal(bodyBytes, &telemetry)

	if err != nil {
		t.Fatalf("Esperava que o Unmarshal funcionasse sem erros: %v", err)
	}

	if telemetry.DeviceID != "sensor-xyz" {
		t.Errorf("DeviceID recebido incorreto. Recebeu %s", telemetry.DeviceID)
	}

	if telemetry.Value != 90.5 {
		t.Errorf("Value incorreto. Recebeu %f", telemetry.Value)
	}

	if telemetry.Nature != "analog" {
		t.Errorf("Nature incorreta. Recebeu %s", telemetry.Nature)
	}
}

func TestDecodeTelemetryJson_Failure_Malformed(t *testing.T) {
	// JSON completamente inválido/quebrado
	bodyBytes := []byte(`{"device_id": "sensor-xyz", "value": "AQUI_DEVERIA_SER_NUMERO"}`)

	var telemetry TelemetryData
	err := json.Unmarshal(bodyBytes, &telemetry)

	if err == nil {
		t.Fatalf("O desempacotamento deveria ter falhado por tipagem incorreta")
	}
}
