package main

import (
	"fmt"
	"net/http"
	"OnlineQuizSystem/db"
	"OnlineQuizSystem/api"
	"OnlineQuizSystem/utils"
	"OnlineQuizSystem/sockets"

	"github.com/gorilla/mux"
)



func main() {
	db.DB = db.Init()
	fmt.Println("DB Initialized: ", db.DB, db.DB.Config)
	router := mux.NewRouter()

	// Auth apis
	router.HandleFunc("/register", api.RegisterHandler).Methods("POST")
	router.HandleFunc("/verify-email", api.VerifyEmailHandler).Methods("POST")
	router.HandleFunc("/login", api.LoginHandler).Methods("POST")
	router.HandleFunc("/forgot-password", api.ForgotPasswordHandler).Methods("POST")
	router.HandleFunc("/change-password", api.ChangePasswordHandler).Methods("POST")

	// Current User profile
	router.HandleFunc("/user/profile", api.RetrieveCurrentUserProfileHandler).Methods("GET")
	
	// User model apis
	router.HandleFunc("/user/create", api.CreateUserHandler).Methods("POST")
	router.HandleFunc("/user/list", api.RetrieveUserListHandler).Methods("GET")
	router.HandleFunc("/user/detail", api.RetrieveUserDetailHandler).Methods("GET")
	router.HandleFunc("/user/update", api.UpdateUserPatchHandler).Methods("PATCH")
	router.HandleFunc("/user/delete", api.SoftDeleteUserHandler).Methods("DELETE")

	// UserDetails model apis
	router.HandleFunc("/user-detail/create", api.CreateUserDetailsHandler).Methods("POST")
	router.HandleFunc("/user-detail/list", api.RetrieveUserDetailsListHandler).Methods("GET")
	router.HandleFunc("/user-detail/detail", api.RetrieveUserDetailsDetailHandler).Methods("GET")
	router.HandleFunc("/user-detail/update", api.UpdateUserDetailsPatchHandler).Methods("PATCH")
	router.HandleFunc("/user-detail/delete", api.SoftDeleteUserDetailsHandler).Methods("DELETE")


	// QuizEvent model apis
	router.HandleFunc("/quiz-event/create", api.CreateQuizEventHandler).Methods("POST")
	router.HandleFunc("/quiz-event/list", api.RetrieveQuizEventListHandler).Methods("GET")
	router.HandleFunc("/quiz-event/detail", api.RetrieveQuizEventDetailHandler).Methods("GET")
	router.HandleFunc("/quiz-event/update", api.UpdateQuizEventPatchHandler).Methods("PATCH")
	router.HandleFunc("/quiz-event/delete", api.SoftDeleteQuizEventHandler).Methods("DELETE")


	// EventResult model apis
	router.HandleFunc("/event-result/create", api.CreateEventResultHandler).Methods("POST")
	router.HandleFunc("/event-result/list", api.RetrieveEventResultListHandler).Methods("GET")
	router.HandleFunc("/event-result/detail", api.RetrieveEventResultDetailHandler).Methods("GET")
	router.HandleFunc("/event-result/update", api.UpdateEventResultPatchHandler).Methods("PATCH")
	router.HandleFunc("/event-result/delete", api.SoftDeleteEventResultHandler).Methods("DELETE")


	// Teacher events api
	router.HandleFunc("/quiz", api.CreateQuizEvent).Methods("POST")
	router.HandleFunc("/quiz/{id}/start", api.StartQuiz).Methods("GET")
	router.HandleFunc("/quiz/{id}/end", api.EndQuiz).Methods("GET")

	// Student Join api
	router.HandleFunc("/quiz/join", api.JoinQuizEvent).Methods("POST")

	router.HandleFunc("/ws", sockets.HandleWS)

	println("Server running on http://localhost:8080")
	http.ListenAndServe(utils.GetServerBaseUrl(), router)
}