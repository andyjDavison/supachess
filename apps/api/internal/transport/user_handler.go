package transport

import (
	"api/internal/user"
	"encoding/json"
	"net/http"
)

type UserHandler struct {
	userService *user.UserService
}

func NewUserHandler(userService *user.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// RegisterUserRequest defines the expected incoming JSON payload
type RegisterUserRequest struct {
	Username  string `json:"username"`
	Email string `json:"email"`
	Password string `json:"password"`
	Rating int32 `json:"rating"`
}

// RegisterUserResponse defines the structural JSON output
type RegisterUserResponse struct {
	ID    string  `json:"id"`
	Username  string `json:"username"`
	Rating int32 `json:"rating"`
}

func (handler *UserHandler) CreateUserHandler(writer http.ResponseWriter, request *http.Request) {
	
	var req RegisterUserRequest
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		http.Error(writer, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	// 4. Invoke the service layer, automatically forwarding the request context (r.Context())
	// This ensures that if the client disconnects, your database operation terminates immediately.
	createdUser, err := handler.userService.CreateUser(request.Context(), user.UserRegistrationInput{
		Username: req.Username,
		Email: req.Email,
		Password: req.Password,
		Rating: req.Rating,
	})
	if err != nil {
		// In a production environment, parse the error type to send appropriate status codes
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated) // HTTP 201

	resp := RegisterUserResponse{
		ID:    createdUser.UserID,
		Username:  createdUser.Username,
		Rating: createdUser.Rating,
	}
	json.NewEncoder(writer).Encode(resp)
}
