package socManager

import (
    "log"
	"sync"
    "sync/atomic"
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
    EventStartTime int64
    EventEndTime int64
    StartQuiz    atomic.Bool
    StopRoom     chan bool
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
    log.Println("Getting the room manager - GetManager")
	return manager
}

func (m *Manager) CreateRoom(quizEventID uint, channelCode string, teacherID uint) *Room {
    log.Printf("Creating a room - quizEventID: %d  |  channel_code: %s  |  teacherID: %d ", quizEventID, channelCode, teacherID)
    room := &Room{
        ID:           channelCode,
        QuizEventID:  quizEventID,
        TeacherID:    teacherID,
        Clients:      make(map[uint]*Client),
        Participants: make(map[uint]bool),
        Broadcast:    make(chan any, 10),
        TeacherChan:  make(chan any, 10),
        StopRoom:     make(chan bool),
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
    log.Println("GetRoom method --- ")
	m.RLock()
	defer m.RUnlock()
	room, exists := m.Rooms[channelCode]
    log.Println("GetRoom method - rooms: ", m.Rooms, "room : ", room)
	return room, exists
}

func (r *Room) Run() {
    log.Printf("Room is running - Room.QuizEventID: %d\n", r.QuizEventID)
    keepLoop: for {
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
            r.Lock()
            for _, client := range r.Clients {
                if err := client.Conn.WriteJSON(message); err != nil {
                    log.Printf("Broadcast error to %d: %v", client.UserID, err)
                    client.Conn.Close()
                    delete(r.Clients, client.UserID)
                }
            }
            log.Println("Run() method - sending message data to TeacherChan channel ...")
            r.Unlock()
            r.TeacherChan <- message // Forwarding the broadcast to the teacher
            
        // case message := <-r.TeacherChan:
        //     r.RLock()
        //     if teacherClient, exists := r.Clients[r.TeacherID]; exists {
        //         if err := teacherClient.Conn.WriteJSON(message); err != nil {
        //             log.Printf("Teacher message error: %v", err)
        //             teacherClient.Conn.Close()
        //             delete(r.Clients, r.TeacherID)
        //         }
        //     }
        //     r.RUnlock()
        case IsRoomStop := <-r.StopRoom:
            if (IsRoomStop){
                break keepLoop;
            }
        }

    }
}



func (r *Room) BroadcastToTeacher(message any) {
    r.RLock()
    
    if teacherClient, exists := r.Clients[r.TeacherID]; exists {
        if err := teacherClient.Conn.WriteJSON(message); err != nil {
            log.Printf("Error sending to teacher %d: %v", r.TeacherID, err)
            teacherClient.Conn.Close()
            delete(r.Clients, r.TeacherID)
        }
    }
    r.RUnlock()
    r.TeacherChan <- message
}



func (r *Room) BroadcastToStudent(userID uint, message any) {
    r.RLock()
    
    if client, exists := r.Clients[userID]; exists {
        if err := client.Conn.WriteJSON(message); err != nil {
            log.Printf("Error sending to student %d: %v", userID, err)
            client.Conn.Close()
            delete(r.Clients, userID)
        }
    }
    r.RUnlock()
    r.TeacherChan <- message
}


// func (r *Room) Broadcast(message any) {
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