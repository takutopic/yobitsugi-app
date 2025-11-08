// Package main
package main

import (
	"log"
)

// PrivateMessage is a struct to wrap a message with the intended client ID.
type PrivateMessage struct {
	ClientID string
	Payload  []byte
}

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	clients     map[string]*Client   // Registered clients
	sendPrivate chan *PrivateMessage // Inbound messages from clients
	register    chan *Client         // Register requests from clients
	unregister  chan *Client         // Unregister requests from clients
}

// newHub creates a new Hub.
func newHub() *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		sendPrivate: make(chan *PrivateMessage),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
	}
}

// run is the main loop for the Hub. It must be run as a goroutine.
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			// A new client has connected.
			h.clients[client.ID] = client
			log.Printf("A new client '%s' has connected. Total clients: %d", client.ID, len(h.clients))

		case client := <-h.unregister:
			// A client has disconnected.
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.send)
				log.Printf("A client '%s' has disconnected. Total clients: %d", client.ID, len(h.clients))
			}

		case message := <-h.sendPrivate:
			// Find the specific client by its ID
			if client, ok := h.clients[message.ClientID]; ok {
				select {
				case client.send <- message.Payload:
				default:
					close(client.send)
					delete(h.clients, client.ID)
				}
			}
		}
	}
}
