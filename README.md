# Online Quiz System Backend

This project implements the backend logic for a real-time online quiz system. It provides APIs for teachers and administrators to create quiz events and allows users to join these events. The backend manages quiz data, user authentication, and facilitates the initial connection for real-time quiz interactions using WebSockets.

## Features

* **Create Quiz Event:** Authenticated teachers and administrators can create new quiz events by providing a name and a JSON payload defining the quiz structure. A unique channel code is generated for each event.
* **Join Quiz Event:** Authenticated users can join a quiz event using its unique channel code. The system verifies the code and prepares the user for real-time interaction.
* **User Authentication & Authorization:** Secure endpoints ensure that only authorized users can create quizzes, and all users need to be authenticated to join.
* **JSON-based Quiz Definition:** Quizzes are defined using a flexible JSON format, allowing for diverse question types.
* **WebSocket Integration:** The backend sets up the initial stage for WebSocket connections, enabling real-time communication during quizzes.

## Technology Stack

* **Language:** Go
* **Libraries:**
    * `net/http`: For handling HTTP requests and responses.
    * `encoding/json`: For encoding and decoding JSON data.
    * `strconv`: For converting data types.
    * `time`: For generating unique channel codes.
    * `OnlineQuizSystem/db`: For database interactions.
    * `OnlineQuizSystem/utils`: For utility functions, including authorization and server URL retrieval.
    * `OnlineQuizSystem/models`: For defining data structures.
    * `OnlineQuizSystem/sockets`: For managing WebSocket connections and rooms.

## System Architecture

The backend exposes HTTP API endpoints for quiz creation and joining. It likely interacts with a database to persist quiz event data. WebSocket functionality is managed to handle real-time communication during active quizzes.

## API Endpoints

* **`POST /create_quiz`:** Creates a new quiz event. Requires authentication and authorization (admin or teacher). Accepts a JSON payload with `quiz_event_name` and `quiz_json`. Returns a JSON response containing the `channel_code` for the created quiz.
* **`POST /join_quiz`:** Allows an authenticated user to join a quiz event. Accepts a JSON payload with the `channel_code`. Returns a JSON response with the status ("joined"), quiz details, and the `websocket_url` for connecting to the quiz.

## Running the Backend

*(Instructions on how to run the backend server would typically go here. This would involve steps like cloning the repository, setting up Go dependencies, configuring environment variables, and running the main application file.)*

## Future Enhancements

* **Full WebSocket Implementation:** Implement the complete logic for real-time quiz interactions (question delivery, answer submission, scoring).
* **Database Schema Definition:** Define the database schema explicitly (e.g., using GORM).
* **Detailed Quiz Question Handling:** Implement specific logic for different question types.
* **Quiz Scheduling and Management:** Add features for scheduling quizzes and managing their lifecycle.
* **Result Storage and Analysis:** Implement functionality to store and analyze quiz results.
* **Error Handling and Logging:** Enhance error handling and implement comprehensive logging.
* **Unit and Integration Tests:** Add tests to ensure code quality and reliability.

## Author

*(Anurag Pandey)*