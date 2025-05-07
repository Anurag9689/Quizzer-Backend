package api

import (
	"encoding/json"
	"log"
	"net/http"
	// "strconv"
	"time"

	"OnlineQuizSystem/db"
	"OnlineQuizSystem/models"
	"OnlineQuizSystem/sockets"
	"OnlineQuizSystem/utils"

	"github.com/gorilla/mux"
	// "gorm.io/datatypes"
)

func StartQuiz(w http.ResponseWriter, r *http.Request) {
	user, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	quizID := vars["id"]

	var quizEvent models.QuizEvent
	if err := db.DB.First(&quizEvent, quizID).Error; err != nil {
		http.Error(w, "Quiz not found", http.StatusNotFound)
		return
	}

	// Verify teacher owns this quiz
	if quizEvent.UserID != user.ID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	quizJson, err := quizEvent.GetQuizJsonFileMap()
	if err != nil {
		http.Error(w, "Internal Error, Not able to read this QuizEvent's Json file: "+err.Error(), http.StatusInternalServerError)
		return 
	}

	// Update quiz status
	quizJson["Status"] = "active"
	quizEvent.SetQuizJsonFileMap(quizJson)

	if err := db.DB.Save(&quizEvent).Error; err != nil {
		http.Error(w, "Failed to start quiz", http.StatusInternalServerError)
		return
	}

	// Notify participants via WebSocket
	manager := sockets.GetManager()
	if room, exists := manager.GetRoom(*quizEvent.ChannelCode); exists {
		// Load quiz questions from JSON file
		if quizJson == nil {
			log.Printf("Error reading quiz data: %v", err)
			http.Error(w, "Failed to load quiz data", http.StatusInternalServerError)
			return
		}

		room.Broadcast <- map[string]any{
			"type":       "quiz_started",
			"quiz_id":    quizEvent.ID,
			"start_time": time.Now().Unix(),
			"quiz_json":  quizJson,
		}
		log.Printf("Broadcast quiz start to room %s", *quizEvent.ChannelCode)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "quiz started"})
}

// func SubmitAnswers(w http.ResponseWriter, r *http.Request) {
// 	user, _, err := utils.AuthorizeUser(r)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusUnauthorized)
// 		return
// 	}

// 	vars := mux.Vars(r)
// 	quizID, err := strconv.ParseUint(vars["id"], 10, 32)
// 	if err != nil {
// 		http.Error(w, "Not able to parse vars['"+vars["id"]+"'] from http.request : ", http.StatusInternalServerError)
// 		return
// 	}

// 	var submission struct {
// 		Answers    datatypes.JSON `json:"answers"`
// 		TimeTaken  int            `json:"time_taken"`
// 		IsLate     bool           `json:"is_late"`
// 	}

// 	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
// 		http.Error(w, "Invalid request", http.StatusBadRequest)
// 		return
// 	}

// 	// Create event result
// 	result := models.EventResult{
// 		UserID:        user.ID,
// 		QuizEventID:   uint(quizID),
// 		ExpScore:      0, // Will be calculated later
// 		ExtraInfoJson: &submission.Answers,
// 	}

// 	if err := db.DB.Create(&result).Error; err != nil {
// 		http.Error(w, "Failed to save results", http.StatusInternalServerError)
// 		return
// 	}

// 	// Award teacher experience
// 	// if err := db.DB.Model(&models.User{}).Where("id = ?", quizEvent.UserID).
// 	// 	Update("exp_score", gorm.Expr("exp_score + ?", 20)).Error; err != nil {
// 	// 	log.Printf("Failed to award teacher exp: %v", err)
// 	// }

// 	w.WriteHeader(http.StatusCreated)
// 	json.NewEncoder(w).Encode(result)
// 	log.Printf("Answers submitted for quiz %d by user %d", quizID, user.ID)
// }


// func EndQuiz(w http.ResponseWriter, r *http.Request) {
// 	user, _, err := utils.AuthorizeUser(r)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusUnauthorized)
// 		return
// 	}

// 	vars := mux.Vars(r)
// 	quizID, _ := strconv.Atoi(vars["id"])

// 	// Verify teacher owns this quiz
// 	var quiz models.QuizEvent
// 	if err := db.DB.First(&quiz, quizID).Error; err != nil || quiz.UserID != user.ID {
// 		http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 		return
// 	}

// 	// Finalize quiz (process answers)
// 	sockets.FinalizeQuiz(uint(quizID))

// 	// Update quiz status
// 	quiz.Status = "completed"
// 	db.DB.Save(&quiz)

// 	// Notify participants
// 	if room, exists := sockets.GetManager().GetRoom(*quiz.ChannelCode); exists {
// 		room.Broadcast(map[string]interface{}{
// 			"type": "quiz_ended",
// 			"results_available": true,
// 		})
// 	}

// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]string{"status": "quiz finalized"})
// }