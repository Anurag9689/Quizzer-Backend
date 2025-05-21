package sockets

import (
	"fmt"
	"log"
	"time"
	"strconv"
	"net/http"
	"encoding/json"

	"OnlineQuizSystem/db"
	"OnlineQuizSystem/utils"
	"OnlineQuizSystem/models"
	"OnlineQuizSystem/socManager"

	"github.com/gorilla/websocket"
	// "gorm.io/datatypes"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var activeSessions = make(map[uint]*utils.QuizSession) // quizEventID -> session


var msg struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}


const (
	MsgTypeAnswer       = "answer"
	MsgTypeQuizProgress = "progress"
	MsgTypeEndQuiz      = "end_quiz_event"
	MsgTypeQuizStarted  = "start_quiz_event"
	MsgTypeGetClients   = "get_clients"
	MsgTypeRemoveClients = "remove_clients"
	MsgTypeRemoveClient = "remove_client"
)

type BroadcastedData struct {
	QuizID uint 			`json:"quiz_id"`
	EventStartTime int64 	`json:"start_time"`
	EventEndTime int64 		`json:"end_time"`
	QuizJson map[string]any `json:"quiz_json"`
}


func GetActiveSessions() (*map[uint]*utils.QuizSession){
	return &activeSessions
}

func HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket Upgrade Error:", err)
		return
	}
	defer conn.Close()

	channelCode := r.URL.Query().Get("channel_code")
	userID, _ := strconv.Atoi(r.URL.Query().Get("user_id"))

	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		conn.WriteJSON(map[string]string{"error": "invalid user"})
		return
	}

	manager := socManager.GetManager()
	room, exists := manager.GetRoom(channelCode)
	if !exists {
		conn.WriteJSON(map[string]string{"error": "room not found"})
		return
	}

	var quizEvent models.QuizEvent
	if err := db.DB.First(&quizEvent, "channel_code = ?", channelCode).Error; err != nil {
		conn.WriteJSON(map[string]string{"error": "QuizEvent record not found! Please first create a quiz Event."})
		conn.Close()
		return
	}

	if user.UserType != "teacher" && !room.Participants[uint(userID)] {
		conn.WriteJSON(map[string]string{"error": "not a participant"})
		return
	}

	client := &socManager.Client{
		Conn:     conn,
		UserID:   uint(userID),
		UserType: user.UserType,
	}

	room.Register <- client
	defer func() { room.Unregister <- client }()

	isTeacher := false
	log.Printf("[main] Pointer to startQuiz: %p, value: %v", &room.StartQuiz, room.StartQuiz.Load())


	if user.UserType == "teacher" || user.UserType == "admin" && client.UserID == room.TeacherID {
		isTeacher = true
		conn.WriteJSON(map[string]string{"message" : "Congrats, You have joined the room you created! You can start the quiz event any time you want. Only those Student's who have already joined this room will be allowed to give quizzes. Other's who did not join will not be allowed to join this event after it starts."})
	} else if user.UserType == "student" {
		conn.WriteJSON(map[string]string{"message" : "Congrats, You have joined the room! Please wait for quiz event to start."})
	}

	// Auto-Ending event
	// delay
	// time.AfterFunc()


	joinedClients := make(map[uint]models.User)
	clientList := make(map[string][]uint)
	if (isTeacher){
		go func(){
			routineLoop: for {
				message := <-room.TeacherChan
				log.Println("message from Teacher channel: ", message.(map[string]any))
				log.Println("IsTeacher: ", isTeacher)
				if (isTeacher && MsgTypeQuizStarted == message.(map[string]any)["type"].(string)){
					// var broadcastedData BroadcastedData;
					// _ := payload["quiz_id"].(uint)
					// _ := payload["quiz_json"].(map[string]any)
					// if err := json.Unmarshal(payload, &broadcastedData); err != nil {
						// 	fmt.Println("Some error in unmarshalling Broadcasted data of the payload : "+err.Error())
						// }
					payload:=message.(map[string]any)["payload"].(map[string]any)
					EventStartTime := payload["start_time"].(int64)
					EventEndTime := payload["end_time"].(int64)
					log.Printf("Received EventStartTime: %v", EventStartTime)
					log.Printf("Received EventEndTime: %v", EventEndTime)
					room.EventStartTime = EventStartTime
					room.EventEndTime = EventEndTime
					targetUnixMillis := room.EventEndTime
					log.Printf("room.EventStartTime: %d, room.EventEndTime: %d, targetUnixMillis %d", room.EventStartTime, room.EventEndTime, targetUnixMillis)
					delay := time.Until(time.UnixMilli(targetUnixMillis))
					log.Printf("Delay is set for %f seconds", delay.Seconds())
					if delay > 0 {
						time.AfterFunc(delay, func(){
							log.Printf("Auto-scheduled function executed after %f seconds", delay.Seconds())
							utils.PrepareEndQuiz(quizEvent, &user, &activeSessions);
							room.BroadcastToTeacher( map[string]any{ 
								"type": "end_quiz_event", 
								"payload" : map[string]bool{"results" : true} }, );
						})
					}
					log.Printf("[goroutine] Pointer to startQuiz: %p", &room.StartQuiz)
				    log.Println("[goroutine] Setting startQuiz to true")
					room.StartQuiz.Store(true)
					log.Printf("[goroutine] After Store, startQuiz: %v", room.StartQuiz.Load())
				} else if(isTeacher && MsgTypeEndQuiz == message.(map[string]any)["type"].(string)){
					log.Printf("Creating Teacher's EventResult 1 - ")
					result := models.EventResult{
						UserID:        room.TeacherID,
						QuizEventID:   room.QuizEventID,
						ExpScore:      50,
					}
					log.Printf("Creating Teacher's EventResult 2 - ")
					if err := db.DB.FirstOrCreate(&result).Error; err != nil {
						log.Printf("Error saving final result: %v", err)
					}
					room.StopRoom <- true
					room = &socManager.Room{}
					break routineLoop
				} else if (MsgTypeRemoveClient == message.(map[string]any)["type"].(string)){
					log.Printf("client %d, Just got removed\n", userID)
					break routineLoop
				}
			}
		}()
	}

	outerLoop: for {
		
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Read error: %v", err)
			break outerLoop
		}
		log.Println("Inside HandleWS for loop, type of msg got - ", msg.Type)
		
		switch msg.Type {
			case MsgTypeGetClients:
				if(isTeacher){
					for _, client := range room.Clients {
						var tmpUser models.User
						if err := db.DB.First(&tmpUser, client.UserID).Error; err != nil {
							log.Printf("Joined client do not exists in the database - %d: %v", client.UserID, err);
							client.Conn.Close()
							delete(room.Clients, client.UserID)
							continue
						}
						joinedClients[client.UserID] = tmpUser
					}
					conn.WriteJSON(map[string]any{"JoinedStudents" : joinedClients})
				}
			case MsgTypeRemoveClients:
				if(isTeacher){
					fmt.Println(msg.Payload)
					fmt.Printf("Type of msg.Payload - %T:\n", msg.Payload)
					json.Unmarshal(msg.Payload, &clientList)
					log.Println("clientList: ", clientList)
					for _, clientId := range clientList["client_list"] {
						innerLoop: for _, client := range room.Clients {
							if client.UserID == uint(clientId){
								conn.WriteJSON(map[string]string{"message" : fmt.Sprintf("Removing client/user %d", client.UserID) })
								client.Conn.WriteJSON(map[string]string{"message":"WARNING: You are being removed by the teacher, Sorry ¯\\_(ツ)_/¯"})
								room.BroadcastToStudent(client.UserID, map[string]any{"type" : "remove_client", "payload" : make(map[string]any) })
								client.Conn.Close()
								delete(room.Clients, client.UserID)
								break innerLoop
							}
						}
					}
				}
			case MsgTypeAnswer:
				log.Println("startquiz while msg is of type 'answer' : ", room.StartQuiz.Load())
				log.Printf("[MsgTypeAnswer] Pointer to startQuiz: %p", &room.StartQuiz)
				if (room.StartQuiz.Load()){
					handleAnswerSubmission(room, client, msg.Payload)
				}
			default:
				log.Printf("Unknown message type: %s", msg.Type)
		}
	}


}


func handleAnswerSubmission(room *socManager.Room, client *socManager.Client, payload json.RawMessage) {
	var answer utils.QuizAnswer
	if err := json.Unmarshal(payload, &answer); err != nil {
		log.Printf("Error decoding answer: %v", err)
		return
	}
	answer.Timestamp = time.Now().UnixMilli()

	if _, exists := activeSessions[room.QuizEventID]; !exists {
		activeSessions[room.QuizEventID] = &utils.QuizSession{
			Answers: make(map[uint]map[int]utils.QuizAnswer),
		}
	}

	session := activeSessions[room.QuizEventID]
	session.Lock()
	defer session.Unlock()

	if _, exists := session.Answers[client.UserID]; !exists {
		session.Answers[client.UserID] = make(map[int]utils.QuizAnswer)
	}

	session.Answers[client.UserID][answer.QuestionID] = answer

	log.Printf("Student's answer is submitted - client.UserID: %d,  answer.QuestionID: %d \n", client.UserID, answer.QuestionID)

	room.BroadcastToTeacher(map[string]any{
		"type":        "answer_update",
		"user_id":     client.UserID,
		"question_id": answer.QuestionID,
		"timestamp":   answer.Timestamp,
	})
}





/*
websocket special msg

	MsgTypeAnswer       = "answer"
	MsgTypeEndQuiz      = "end_quiz_event"
	MsgTypeQuizStarted  = "start_quiz_event"
	MsgTypeGetClients   = "get_clients"
	MsgTypeRemoveClients = "remove_clients"


from teacher
- { "type" : "get_clients", "payload" : {} } // to get all the joined students
- { "type" : "remove_clients", "payload" : { "client_list" : [1, 2, 3, 4] } } // payload is the list of userId's of client to remove


from broadcast
- { "type" : "start_quiz_event", payload : {"quiz_id", "start_time", "end_time", "quiz_json"}}


from student
- { "type" : "answer", "payload" : { "question_id" : 1, "answer" : <["London", "Paris"](string array) | 23.0(float64)> } }
- { "type" : "exit_event", "payload" : {}}






### QuizJson

{
  "quiz_event_name": "Science Quiz 1",
  "quiz_json": {
    "questions": [
        {
            "id": 1,
            "text": "What is the capital of France?",
            "type": "mcq",
            "options": [
                { "option" : "London", "correct" : true },
                { "option" : "Paris", "correct" : false },
                { "option" : "Berlin", "correct" : false },
                { "option" : "Madrid", "correct" : false }
            ],
            "points": 5
        },
        {
            "id": 2,
            "text": "Select mammals ?",
            "type": "msq",
            "options": [
                { "option" : "Humans", "correct" : true },
                { "option" : "Elephants", "correct" : true },
                { "option" : "Birds", "correct" : false },
                { "option" : "Dinasaurs", "correct" : false }
            ],
            "points": 2
        },
        {
            "id": 3,
            "text": "What is 7x3 ?",
            "type": "numeric",
            "correct_answer" : 27,
            "points": 5
        }
    ],
    "duration": 30,
    "status": "pending"
  }
}

############################ Answers ##########################


# mcq or msq type answers
{ 
	"type" : "answer", 
	"payload" : { 
		"question_id" : 1, 
		"answer" : ["London"]
	}
}
# numeric type answers
{ 
	"type" : "answer", 
	"payload" : { 
		"question_id" : 1, 
		"answer" : 29.7
	}
}

*/
