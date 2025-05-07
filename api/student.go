package api

import (
	"log"
	"strconv"
	"net/http"
	"encoding/json"

	// "github.com/gorilla/mux"
	"OnlineQuizSystem/db"
	"OnlineQuizSystem/utils"
	"OnlineQuizSystem/models"
	"OnlineQuizSystem/sockets"
)

func JoinQuizEvent(w http.ResponseWriter, r *http.Request) {
	user, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		ChannelCode string `json:"channel_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var quizEvent models.QuizEvent
	if err := db.DB.Where("channel_code = ?", req.ChannelCode).First(&quizEvent).Error; err != nil {
		http.Error(w, "Quiz event not found", http.StatusNotFound)
		return
	}

	quizJson, err := quizEvent.GetQuizJsonFileMap()
	if err != nil {
		http.Error(w, "QuizEvent is currupted and Do not have extra details needed for joining, Please contact the developer " + err.Error(), http.StatusInternalServerError)
		return 
	}

	// Check if quiz is joinable (status pending)
	if quizJson["status"] != "pending" {
		http.Error(w, "Quiz is no longer joinable", http.StatusForbidden)
		return
	}

	// Add participant (you might need a Participant model)
	// For now, we'll just track in WebSocket room
	manager := sockets.GetManager()
	if room, exists := manager.GetRoom(req.ChannelCode); exists {
		room.Participants[user.ID] = false // false = not ready yet
	}

	response := map[string]any{
		"status":      "joined",
		"quiz_event":  quizEvent,
		"websocket_url": "ws://"+utils.GetServerBaseUrl()+"/ws?channel=" + req.ChannelCode + "&user_id=" + strconv.Itoa(int(user.ID)),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("User %d joined quiz %d", user.ID, quizEvent.ID)
}