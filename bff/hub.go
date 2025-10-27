// Package main
package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Upgrader configures the parameters for upgrading an HTTP connection to a WebSocket Connection.
var upgrader = websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(r *http.Request) bool {
		return true
	}
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *hub
	conn *websocket.Conn
	send chan []byte // A buffered channel for outbound messages
}

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	clients		map[*Client]bool	// Registered clients
	broadcast 	chan []byte			// Inbound messages from clients
	register 	chan *Client		// Register requests from clients
	unregister	chan *Client 		// Unregister requests from clients
}

// newHub creates a new Hub.
func newHub() *Hub {
	return &Hub{
		broadcast:	make(chan []byte).
		register: 	make(chan *Client),
		unregister: make(chan *Client),
		clients: 	make(map[*Client]bool),
	}
}

// run is the main loop for the Hub. It must be run as a goroutine
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			// A new client has connected.
			h.clients[client] = true
			log.Println("A new client has connected. Total clients:", len(h.clients))
		
		case client := <-h.unregister:
			// A client has disconnected.
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Println("A client has disconnected. Total clients:", len(h.clients))
			}
		
		case message := <-h.broadcast:
			// A message needs to be broadcast to all clients.
			for client := range h.clients {
				select {
				case client.send <- message:
					// Send the message to the client's send channel.
				default:
					// If the send channel is full, assume the client is dead.
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}