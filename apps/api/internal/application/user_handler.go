package application

import (
	"api/internal/repository"
	"api/internal/service"
	"encoding/json"
	"net/http"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// RegisterUserRequest defines the expected incoming JSON payload
type RegisterUserRequest struct {
	Username  string `json:"username"`
	Email string `json:"email"`
	PasswordHash string `json:"password_hash"`
	Rating int32 `json:"rating"`
}

// RegisterUserResponse defines the structural JSON output
type RegisterUserResponse struct {
	ID    int64  `json:"id"`
	Username  string `json:"username"`
	Email string `json:"email"`
	Rating int32 `json:"rating"`
}

func (h *UserHandler) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Enforce POST requests only
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Decode the raw JSON request body
	var req RegisterUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 3. Map request data into your core Domain/Repository struct
	newUser := &repository.User{
		Username:  req.Username,
		Email: req.Email,
		PasswordHash: req.PasswordHash,
		Rating: req.Rating,
	}

	// 4. Invoke the service layer, automatically forwarding the request context (r.Context())
	// This ensures that if the client disconnects, your database operation terminates immediately.
	err = h.userService.CreateUser(r.Context(), newUser)
	if err != nil {
		// In a production environment, parse the error type to send appropriate status codes
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Formulate and send the success JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // HTTP 201

	resp := RegisterUserResponse{
		ID:    newUser.ID,
		Username:  newUser.Username,
		Email: newUser.Email,
		Rating: newUser.Rating,
	}
	json.NewEncoder(w).Encode(resp)
}
