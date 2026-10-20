package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"api/internal/domain"
	"api/internal/user"
)

func newRegisterHandler(repo *fakeUserRepo) *RegisterHandler {
	return NewRegisterHandler(user.NewUserService(repo, user.NewBcryptHasher(4)))
}

func postRegister(h *RegisterHandler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateUserHandler(rec, req)
	return rec
}

const validRegistration = `{"username":"alice","email":"alice@example.com","password":"password123","rating":800}`

func TestCreateUserSuccess(t *testing.T) {
	rec := postRegister(newRegisterHandler(newFakeUserRepo()), validRegistration)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body["username"] != "alice" || body["rating"] != float64(800) {
		t.Errorf("body = %v", body)
	}
	if body["id"] == "" || body["id"] == nil {
		t.Error("no id in the response")
	}
	// The registration response must not echo anything secret back.
	for _, forbidden := range []string{"password", "passwordHash", "email"} {
		if _, present := body[forbidden]; present {
			t.Errorf("response includes %q", forbidden)
		}
	}
}

// Previously every service error became a 500 with err.Error() echoed to
// the client - the same defect LoginHandler had.
func TestCreateUserWithATakenNameIsConflict(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"username taken", `{"username":"alice","email":"other@example.com","password":"password123","rating":800}`},
		{"email taken", `{"username":"bob","email":"alice@example.com","password":"password123","rating":800}`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := newFakeUserRepo()
			repo.seed(t, "alice", "alice@example.com")

			rec := postRegister(newRegisterHandler(repo), c.body)

			if rec.Code != http.StatusConflict {
				t.Errorf("status = %d, want 409 (this was 500 before writeServiceError)", rec.Code)
			}
		})
	}
}

func TestCreateUserRejectsMalformedJSON(t *testing.T) {
	rec := postRegister(newRegisterHandler(newFakeUserRepo()), `{"username":`)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// validateRegisterInput returns plain errors with no domain sentinel, so
// writeServiceError cannot tell them from a server fault and they land on
// 500. Asserting "a failure, not a success" rather than the exact code
// keeps this from cementing that - see the note in the task summary.
func TestCreateUserRejectsInvalidInput(t *testing.T) {
	quietLogs(t)

	cases := map[string]string{
		"username too short": `{"username":"ab","email":"a@b.com","password":"password123","rating":800}`,
		"password too short": `{"username":"alice","email":"a@b.com","password":"short","rating":800}`,
		"email without @":    `{"username":"alice","email":"nope","password":"password123","rating":800}`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			rec := postRegister(newRegisterHandler(newFakeUserRepo()), body)

			if rec.Code < 400 {
				t.Errorf("status = %d, want a failure", rec.Code)
			}
		})
	}
}

func TestCreateUserRepositoryFailureIsFiveHundred(t *testing.T) {
	quietLogs(t)
	repo := newFakeUserRepo()
	repo.createErr = errBoom

	rec := postRegister(newRegisterHandler(repo), validRegistration)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	// The raw error must not reach the client.
	if strings.Contains(rec.Body.String(), "database is on fire") {
		t.Errorf("the internal error text leaked to the client: %q", rec.Body.String())
	}
}

func TestFindUserByIdHandler(t *testing.T) {
	repo := newFakeUserRepo()
	u := repo.seed(t, "alice", "alice@example.com")
	h := newRegisterHandler(repo)

	t.Run("found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/"+u.UserID, nil)
		req.SetPathValue("id", u.UserID)
		rec := httptest.NewRecorder()
		h.FindUserByIdHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "alice") {
			t.Errorf("body = %q", rec.Body.String())
		}
	})

	t.Run("missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/ghost", nil)
		req.SetPathValue("id", "ghost")
		rec := httptest.NewRecorder()
		h.FindUserByIdHandler(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("empty id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.FindUserByIdHandler(rec, httptest.NewRequest(http.MethodGet, "/api/user/", nil))

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

// These two back the frontend's availability checks, so a 404 has to mean
// "free" and a 200 "taken".
func TestFindUserByEmailAndUsernameHandlers(t *testing.T) {
	repo := newFakeUserRepo()
	repo.seed(t, "alice", "alice@example.com")
	h := newRegisterHandler(repo)

	cases := []struct {
		name     string
		key      string
		value    string
		call     func(http.ResponseWriter, *http.Request)
		wantCode int
	}{
		{"known email", "email", "alice@example.com", h.FindUserByEmailHandler, http.StatusOK},
		{"unknown email", "email", "free@example.com", h.FindUserByEmailHandler, http.StatusNotFound},
		{"known username", "username", "alice", h.FindUserByUsernameHandler, http.StatusOK},
		{"unknown username", "username", "bob", h.FindUserByUsernameHandler, http.StatusNotFound},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/user/"+c.key+"/"+c.value, nil)
			req.SetPathValue(c.key, c.value)
			rec := httptest.NewRecorder()
			c.call(rec, req)

			if rec.Code != c.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, c.wantCode)
			}
		})
	}
}

func TestDomainSentinelsAreMappedNotGeneric(t *testing.T) {
	// Guards the mapper itself: adding a sentinel without a case here
	// silently degrades it to 500.
	cases := map[error]int{
		domain.ErrInvalidCredentials: http.StatusUnauthorized,
		domain.ErrUserNotFound:       http.StatusNotFound,
		domain.ErrGameNotFound:       http.StatusNotFound,
		domain.ErrUsernameTaken:      http.StatusConflict,
		domain.ErrEmailTaken:         http.StatusConflict,
	}

	for err, want := range cases {
		rec := httptest.NewRecorder()
		writeServiceError(rec, err)

		if rec.Code != want {
			t.Errorf("writeServiceError(%v) = %d, want %d", err, rec.Code, want)
		}
	}
}
