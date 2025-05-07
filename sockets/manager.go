package sockets

import (
	"sync"
	"log"
	"github.com/gorilla/websocket"
)

type Client struct {
	Conn     *websocket.Conn
	UserID   uint
	UserType string
}

type Room struct {
    ID           string
    Clients      map[uint]*Client  // userID -> Client
    TeacherID    uint
    QuizEventID  uint
    Broadcast    chan any  // Broadcast to all
    TeacherChan  chan any  // Messages only for teacher
    Register     chan *Client
    Unregister   chan *Client
    Participants map[uint]bool     // Track allowed participants
    sync.RWMutex
}

type Manager struct {
	Rooms map[string]*Room
	sync.RWMutex
}

var manager = &Manager{
	Rooms: make(map[string]*Room),
}

func GetManager() *Manager {
	return manager
}

func (m *Manager) CreateRoom(quizEventID uint, channelCode string, teacherID uint) *Room {
    room := &Room{
        ID:           channelCode,
        QuizEventID:  quizEventID,
        TeacherID:    teacherID,
        Clients:      make(map[uint]*Client),
        Participants: make(map[uint]bool),
        Broadcast:    make(chan any),
        TeacherChan:  make(chan any),
        Register:     make(chan *Client),
        Unregister:   make(chan *Client),
    }
    
    m.Lock()
    m.Rooms[channelCode] = room
    m.Unlock()
    
    go room.Run()
    return room
}

func (m *Manager) GetRoom(channelCode string) (*Room, bool) {
	m.RLock()
	defer m.RUnlock()
	room, exists := m.Rooms[channelCode]
	return room, exists
}

func (r *Room) Run() {
    for {
        select {
        case client := <-r.Register:
            r.Lock()
            r.Clients[client.UserID] = client
            r.Unlock()
            log.Printf("Client %d joined room %s", client.UserID, r.ID)
            
        case client := <-r.Unregister:
            r.Lock()
            if _, ok := r.Clients[client.UserID]; ok {
                client.Conn.Close()
                delete(r.Clients, client.UserID)
                log.Printf("Client %d left room %s", client.UserID, r.ID)
            }
            r.Unlock()
            
        case message := <-r.Broadcast:
            r.RLock()
            for _, client := range r.Clients {
                if err := client.Conn.WriteJSON(message); err != nil {
                    log.Printf("Broadcast error to %d: %v", client.UserID, err)
                    client.Conn.Close()
                    delete(r.Clients, client.UserID)
                }
            }
            r.RUnlock()
            
        case message := <-r.TeacherChan:
            r.RLock()
            if teacherClient, exists := r.Clients[r.TeacherID]; exists {
                if err := teacherClient.Conn.WriteJSON(message); err != nil {
                    log.Printf("Teacher message error: %v", err)
                    teacherClient.Conn.Close()
                    delete(r.Clients, r.TeacherID)
                }
            }
            r.RUnlock()
        }
    }
}



func (r *Room) BroadcastToTeacher(message interface{}) {
    r.RLock()
    defer r.RUnlock()
    
    if teacherClient, exists := r.Clients[r.TeacherID]; exists {
        if err := teacherClient.Conn.WriteJSON(message); err != nil {
            log.Printf("Error sending to teacher %d: %v", r.TeacherID, err)
            teacherClient.Conn.Close()
            delete(r.Clients, r.TeacherID)
        }
    }
}



func (r *Room) BroadcastToStudent(userID uint, message interface{}) {
    r.RLock()
    defer r.RUnlock()
    
    if client, exists := r.Clients[userID]; exists {
        if err := client.Conn.WriteJSON(message); err != nil {
            log.Printf("Error sending to student %d: %v", userID, err)
            client.Conn.Close()
            delete(r.Clients, userID)
        }
    }
}


// func (r *Room) Broadcast(message interface{}) {
//     r.RLock()
//     defer r.RUnlock()
//     for _, client := range r.Clients {
//         if err := client.Conn.WriteJSON(message); err != nil {
//             log.Printf("Broadcast error to %d: %v", client.UserID, err)
//             client.Conn.Close()
//             delete(r.Clients, client.UserID)
//         }
//     }
// }