package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

type Server struct {
	conns map[*websocket.Conn]bool
}

func NewServer() *Server {
	return &Server{
		conns: make(map[*websocket.Conn]bool),
	}
}

func (s *Server) handleWS(ws *websocket.Conn) {
	fmt.Println("New connection:", ws.RemoteAddr())
	s.conns[ws] = true
	defer func() {
		fmt.Println("Connection closed:", ws.RemoteAddr())
		delete(s.conns, ws)
		ws.Close()
	}()

	s.readLoop(ws)
}

func (s *Server) readLoop(ws *websocket.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := ws.Read(buf)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Println("Error reading", err)
			continue
		}
		msg := string(buf[:n])
		fmt.Println("Received:", msg)

		if strings.TrimSpace(msg) == "/count" {
			msg = fmt.Sprintf("Connected clients: %d", len(s.conns))
			ws.Write([]byte(msg))
		} else {
			s.broadcast(msg, ws)
		}
	}
}

func (s *Server) handleWSOrderbook(ws *websocket.Conn) {
	fmt.Println("New Connection for Orderbook:", ws.RemoteAddr())
	defer func() {
		fmt.Println("Connection closed for Orderbook:", ws.RemoteAddr())
		ws.Close()
	}()

	stop := make(chan struct{})
	go s.writeOrderbook(ws, stop)

	buf := make([]byte, 1024)
	for {
		_, err := ws.Read(buf)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Println("Error reading from Orderbook connection:", err)
			break
		}
	}
	close(stop)
}

func (s *Server) writeOrderbook(ws *websocket.Conn, stop chan struct{}) {
	for {
		select {
		case <-stop:
			return
		default:
			trades := fmt.Sprintf("Time of trade: %s, Price: %s, Quantity: %s", time.Now().Format("2006-01-02T15:04:05.000Z07:00"), "100", "100")
			_, err := ws.Write([]byte(trades))
			if err != nil {
				fmt.Println("Error in Orderbook:", err)
				return
			}
			time.Sleep(5 * time.Second)
		}
	}
}

func (s *Server) broadcast(msg string, origin *websocket.Conn) {
	for conn := range s.conns {
		if conn != origin {
			_, err := conn.Write([]byte(msg))
			if err != nil {
				fmt.Println("Error broadcasting:", err)
				continue
			}
		}
	}
}

func main() {
	server := NewServer()

	http.Handle("/ws", websocket.Handler(server.handleWS))
	http.Handle("/wsorderbook", websocket.Handler(server.handleWSOrderbook))
	fmt.Println("WebSocket server started on :3000")
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		fmt.Println("ListenAndServe:", err)
	}
}
