package worker

import (
	"errors"
	"github.com/goccy/go-json"
	"github.com/united-manufacturing-hub/united-manufacturing-hub/cmd/kafka-to-postgresql-v2/shared"
)

func parseWorkOrderCreate(value []byte) (shared.WorkOrderCreateMessage, error) {
	// Try parse to WorkOrderCreateMessage
	var message shared.WorkOrderCreateMessage
	err := json.Unmarshal(value, &message)

	// Validate that ExternalWorkOrderId, Product.ExternalProductId & Quantity are set
	if message.ExternalWorkOrderId == "" {
		return message, errors.New("externalWorkOrderId is required")
	}
	if message.Product.ExternalProductId == "" {
		return message, errors.New("product.externalProductId is required")
	}
	if message.Quantity == 0 {
		return message, errors.New("quantity is required")
	}
	// Status falls back to zero, if not set

	return message, err
}

func parseWorkOrderStart(value []byte) (shared.WorkOrderStartMessage, error) {
	// Try parse to WorkOrderStartMessage
	var message shared.WorkOrderStartMessage
	err := json.Unmarshal(value, &message)

	// Validate that ExternalWorkOrderId & StartTimeUnixMs are set
	if message.ExternalWorkOrderId == "" {
		return message, errors.New("externalWorkOrderId is required")
	}
	if message.StartTimeUnixMs == 0 {
		return message, errors.New("start_time_unix_ms is required")
	}
	return message, err
}

func parseWorkOrderStop(value []byte) (shared.WorkOrderStopMessage, error) {
	// Try parse to WorkOrderStopMessage
	var message shared.WorkOrderStopMessage
	err := json.Unmarshal(value, &message)

	// Validate that ExternalWorkOrderId & EndTimeUnixMs are set
	if message.ExternalWorkOrderId == "" {
		return message, errors.New("externalWorkOrderId is required")
	}
	if message.EndTimeUnixMs == 0 {
		return message, errors.New("end_time_unix_ms is required")
	}
	return message, err
}
