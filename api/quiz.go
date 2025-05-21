package api

import (
	"log"
	"time"
	"strconv"
	"net/http"
	"encoding/json"

	"OnlineQuizSystem/db"
	"OnlineQuizSystem/utils"
	"OnlineQuizSystem/models"
	"OnlineQuizSystem/sockets"
	"OnlineQuizSystem/socManager"

	"github.com/gorilla/mux"
)

func StartQuiz(w http.ResponseWriter, r *http.Request) {
	user, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	log.Println("vars: ", vars)
	quizID, _ := strconv.Atoi(vars["id"])
	log.Printf("quizID: %d", quizID)

	var quizEvent models.QuizEvent
	if err := db.DB.First(&quizEvent, "id = ?", quizID).Error; err != nil {
		http.Error(w, "Quiz not found", http.StatusNotFound)
		return
	}
	log.Printf("quizEvent.UserID: %d and user.ID: %d\n", quizEvent.UserID, user.ID)
	if quizEvent.UserID != user.ID {
		http.Error(w, "Unauthorized, You did not created this event.", http.StatusUnauthorized)
		return
	}

	quizJson, err := quizEvent.GetQuizJsonFileMap()
	if err != nil {
		http.Error(w, "Internal Error, Not able to read this QuizEvent's Json file: "+err.Error(), http.StatusInternalServerError)
		return 
	}

	quizJson["status"] = "active"
	quizEvent.SetQuizJsonFileMap(quizJson)

	if err := db.DB.Save(&quizEvent).Error; err != nil {
		http.Error(w, "Failed to start quiz", http.StatusInternalServerError)
		return
	}

	manager := socManager.GetManager()
	if room, exists := manager.GetRoom(*quizEvent.ChannelCode); exists {
		if quizJson == nil {
			log.Printf("Error reading quiz data: %v", err)
			http.Error(w, "Failed to load quiz data", http.StatusInternalServerError)
			return
		}

		quizDurationFloat, ok := quizJson["duration"].(float64)
		if !ok {
			http.Error(w, "Duration must be a number", http.StatusBadRequest)
			return
		}
		quizDurationSecs := int64(quizDurationFloat) + 1

		room.Broadcast <- map[string]any{
			"type":       "start_quiz_event",
			"payload" : map[string]any {
				"quiz_id":    quizEvent.ID,
				"start_time": time.Now().UnixMilli(),
				"end_time" : time.Now().UnixMilli() + quizDurationSecs*1000,
				"quiz_json":  quizJson,
			},
		}
		log.Printf("Broadcast quiz start to room %s", *quizEvent.ChannelCode)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "quiz started"})
}


func EndQuiz(w http.ResponseWriter, r *http.Request) {
	user, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	log.Println("vars: ", vars)
	quizID, _ := strconv.Atoi(vars["id"])
	var quiz models.QuizEvent
	if err := db.DB.First(&quiz, "id = ?", quizID).Error; err != nil || quiz.UserID != user.ID {
		http.Error(w, "Unauthorized user: " + err.Error(), http.StatusInternalServerError)
		return 
	}

	errStr := utils.PrepareEndQuiz(quiz, user, sockets.GetActiveSessions())
	if errStr != "Prepared" {
		http.Error(w, errStr, http.StatusInternalServerError)
		return 
	}
	// Notify teacher
	if room, exists := socManager.GetManager().GetRoom(*quiz.ChannelCode); exists {
		room.BroadcastToTeacher( map[string]any{ "type": "end_quiz_event", "payload" : map[string]bool{"results" : true} }, )
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "quiz finalized"})
}