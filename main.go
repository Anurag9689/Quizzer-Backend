package main

import (
	"fmt"
	"net/http"
	"OnlineQuizSystem/db"
	"OnlineQuizSystem/api"
	"OnlineQuizSystem/utils"
	"OnlineQuizSystem/sockets"
	// "OnlineQuizSystem/models"
)



func main() {
	db.DB = db.Init()
	fmt.Println("DB Initialized: ", db.DB, db.DB.Config)

	// Auth apis
	http.HandleFunc("/register", api.RegisterHandler)
	http.HandleFunc("/verify-email", api.VerifyEmailHandler)
	http.HandleFunc("/login", api.LoginHandler)
	http.HandleFunc("/forgot-password", api.ForgotPasswordHandler)
	http.HandleFunc("/change-password", api.ChangePasswordHandler)

	// Current User profile
	http.HandleFunc("/user/profile", api.RetrieveCurrentUserProfileHandler)

	// User model apis
	http.HandleFunc("/user/create", api.CreateUserHandler)
	http.HandleFunc("/user/list", api.RetrieveUserListHandler)
	http.HandleFunc("/user/detail", api.RetrieveUserDetailHandler)
	http.HandleFunc("/user/update", api.UpdateUserPatchHandler)
	http.HandleFunc("/user/delete", api.SoftDeleteUserHandler)

	// UserDetails model apis
	http.HandleFunc("/user-detail/create", api.CreateUserDetailsHandler)
	http.HandleFunc("/user-detail/list", api.RetrieveUserDetailsListHandler)
	http.HandleFunc("/user-detail/detail", api.RetrieveUserDetailsDetailHandler)
	http.HandleFunc("/user-detail/update", api.UpdateUserDetailsPatchHandler)
	http.HandleFunc("/user-detail/delete", api.SoftDeleteUserDetailsHandler)


	// QuizEvent model apis
	http.HandleFunc("/quiz-event/create", api.CreateQuizEventHandler)
	http.HandleFunc("/quiz-event/list", api.RetrieveQuizEventListHandler)
	http.HandleFunc("/quiz-event/detail", api.RetrieveQuizEventDetailHandler)
	http.HandleFunc("/quiz-event/update", api.UpdateQuizEventPatchHandler)
	http.HandleFunc("/quiz-event/delete", api.SoftDeleteQuizEventHandler)


	// EventResult model apis
	http.HandleFunc("/event-result/create", api.CreateEventResultHandler)
	http.HandleFunc("/event-result/list", api.RetrieveEventResultListHandler)
	http.HandleFunc("/event-result/detail", api.RetrieveEventResultDetailHandler)
	http.HandleFunc("/event-result/update", api.UpdateEventResultPatchHandler)
	http.HandleFunc("/event-result/delete", api.SoftDeleteEventResultHandler)


	// Teacher events api
	http.HandleFunc("/quiz", api.CreateQuizEvent)
	http.HandleFunc("/quiz/{id}/start", api.StartQuiz)

	// Student Join api
	http.HandleFunc("/quiz/join", api.JoinQuizEvent)
	// http.HandleFunc("/quiz/{id}/submit", api.SubmitAnswers)

	http.HandleFunc("/ws", sockets.HandleWS)

	println("Server running on http://localhost:8080")
	http.ListenAndServe(utils.GetServerBaseUrl(), nil)
}