// Package main
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	pongWait = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	writeWait = 10 * time.Second
	maxMessageSize = 512
)

// Upgrader configures the parameters for upgrading an HTTP connection to a WebSocket Connection.
var upgrader = websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub
	conn *websocket.Conn
	send chan []byte // A buffered channel for outbound messages
	ID string
}

// readPump pumps messages from the websocket connection to the hub.
// This is also used to detect when a client disconnects.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)

	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("error setting read deadline: %v", err)
		return
	}

	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// A loop for reading messages to keep the connection alive and detect a close.
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("error setting write deadline: %v", err)
				return
			}
			if !ok {
				// The hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send the message to the client
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			// Send a ping message to the client to keep the connection alive
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs handles the WebSocket connection request from the peer.
func ServeWs(hub *Hub, c *gin.Context) {
	clientId := c.Query("clientId")
	if clientId == "" {
		log.Println("Failed to upgrade: clientId is required")
		c.JSON(http.StatusBadRequest, "clientId query parameter is required")
		return
	}

	// Upgrade the HTTP connection to a WebSocket connection.
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}

	// Create a new Client
	client := &Client{
		hub:  hub, 
		conn: conn, 
		send: make(chan []byte, 256),
		ID:	  clientId,
	}
	// Register the new client with the hub
	client.hub.register <- client

	log.Println("Client successfully upgraded to WebSocket.")

	// Start the client's goroutines
	// Allow the client to read messages and write messages
	go client.writePump()
	go client.readPump()
}
