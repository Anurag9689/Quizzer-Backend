package sockets

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"OnlineQuizSystem/db"
	"OnlineQuizSystem/models"

	"gorm.io/datatypes"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type QuizAnswer struct {
	QuestionID int    `json:"question_id"`
	Answer     any    `json:"answer"`
	Timestamp  int64  `json:"timestamp"`
}

type QuizSession struct {
	sync.Mutex
	Answers map[uint]map[int]QuizAnswer // userID -> questionID -> answer
}

var activeSessions = make(map[uint]*QuizSession) // quizEventID -> session


var msg struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}


const (
	MsgTypeQuestion     = "question"
	MsgTypeAnswer       = "answer"
	MsgTypeQuizProgress = "progress"
	MsgTypeEndQuiz      = "end_quiz"
)


func HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket Upgrade Error:", err)
		return
	}
	defer conn.Close()

	channelCode := r.URL.Query().Get("channel")
	userID, _ := strconv.Atoi(r.URL.Query().Get("user_id"))

	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		conn.WriteJSON(map[string]string{"error": "invalid user"})
		return
	}

	manager := GetManager()
	room, exists := manager.GetRoom(channelCode)
	if !exists {
		conn.WriteJSON(map[string]string{"error": "room not found"})
		return
	}

	if user.UserType != "teacher" && !room.Participants[uint(userID)] {
		conn.WriteJSON(map[string]string{"error": "not a participant"})
		return
	}

	client := &Client{
		Conn:     conn,
		UserID:   uint(userID),
		UserType: user.UserType,
	}

	room.Register <- client
	defer func() { room.Unregister <- client }()


	for {
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}

		switch msg.Type {
		case MsgTypeAnswer:
			handleAnswerSubmission(room, client, msg.Payload)
		// case MsgTypeEndQuiz:
		// 	break;
		default:
			log.Printf("Unknown message type: %s", msg.Type)
		}
	}
}





func handleAnswerSubmission(room *Room, client *Client, payload json.RawMessage) {
	var answer QuizAnswer
	if err := json.Unmarshal(payload, &answer); err != nil {
		log.Printf("Error decoding answer: %v", err)
		return
	}

	if _, exists := activeSessions[room.QuizEventID]; !exists {
		activeSessions[room.QuizEventID] = &QuizSession{
			Answers: make(map[uint]map[int]QuizAnswer),
		}
	}

	session := activeSessions[room.QuizEventID]
	session.Lock()
	defer session.Unlock()

	if _, exists := session.Answers[client.UserID]; !exists {
		session.Answers[client.UserID] = make(map[int]QuizAnswer)
	}

	session.Answers[client.UserID][answer.QuestionID] = answer

	room.BroadcastToTeacher(map[string]any{
		"type":        "answer_update",
		"user_id":     client.UserID,
		"question_id": answer.QuestionID,
		"timestamp":   answer.Timestamp,
	})
}

func FinalizeQuiz(quizEventID uint) {
	session, exists := activeSessions[quizEventID]
	if !exists {
		return
	}

	var quizEvent models.QuizEvent
	if err := db.DB.First(&quizEvent, quizEventID).Error; err != nil {
		log.Printf("Error loading quiz event: %v", err)
		return
	}

	quizData, err := quizEvent.GetQuizJsonFileMap()
	if err != nil {
		log.Printf("Error loading quiz data: %v", err)
		return
	}

	for userID, answers := range session.Answers {
		score, analytics := calculateResults(answers, quizData)
		analyticsByted, _ := json.Marshal(analytics)
		analyticsJson := datatypes.JSON(analyticsByted)

		result := models.EventResult{
			UserID:        userID,
			QuizEventID:   quizEventID,
			ExpScore:      score,
			ExtraInfoJson: &analyticsJson,
		}

		if err := db.DB.Create(&result).Error; err != nil {
			log.Printf("Error saving final result: %v", err)
		}
	}

	delete(activeSessions, quizEventID)
}

func calculateResults(answers map[int]QuizAnswer, quizData map[string]any) (int, map[string]any) {
	score := 0
	analytics := map[string]any{
		"answers":         answers,
		"correct_answers": 0,
		"wrong_answers":   0,
		"time_stats":      make(map[int]float64),
	}

	questions := quizData["questions"].([]any)
	for _, q := range questions {
		question := q.(map[string]any)
		qID := int(question["id"].(float64))
		correctAns := question["correct_answer"]
		
		if ans, exists := answers[qID]; exists {
			if ans.Answer == correctAns {
				score += int(question["points"].(float64))
				analytics["correct_answers"] = analytics["correct_answers"].(int) + 1
			} else {
				analytics["wrong_answers"] = analytics["wrong_answers"].(int) + 1
			}
			
			analytics["time_stats"].(map[int]float64)[qID] = 
				float64(ans.Timestamp - quizData["start_time"].(int64))
		}
	}

	return score, analytics
}