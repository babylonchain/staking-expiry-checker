package handlers

import "github.com/babylonchain/staking-expiry-checker/internal/services"

type QueueHandler struct {
	Services *services.Services
}

type MessageHandler func(messageBody string) error

func NewQueueHandler(services *services.Services) *QueueHandler {
	return &QueueHandler{
		Services: services,
	}
}
