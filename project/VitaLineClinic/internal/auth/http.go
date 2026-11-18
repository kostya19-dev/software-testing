package auth

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type input struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	NewPassword string `json:"new_password"`
	Challenge   string `json:"challenge"`
	Code        string `json:"code"`
}

func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func fail(w http.ResponseWriter, status int, code string) {
	reply(w, status, map[string]string{"error": code})
}

func decode(w http.ResponseWriter, r *http.Request) (input, bool) {
	var v input
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	d.DisallowUnknownFields()
	if d.Decode(&v) != nil {
		fail(w, 400, "invalid_request")
		return v, false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		fail(w, 400, "invalid_request")
		return v, false
	}
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	return v, true
}

func validPassword(value string) bool { return len(value) >= 16 && len(value) <= 72 }

func (h *Handler) persist(w http.ResponseWriter, r *http.Request, u *user) bool {
	if err := h.save(r.Context(), u); err != nil {
		if errors.Is(err, errConflict) {
			fail(w, 409, "retry_request")
		} else {
			fail(w, 503, "auth_unavailable")
		}
		return false
	}
	return true
}

func (h *Handler) Wrap(next http.Handler) http.Handler {
	auth := http.NewServeMux()
	auth.HandleFunc("POST /auth/login", h.login)
	auth.HandleFunc("POST /auth/verify", h.verify)
	auth.HandleFunc("POST /auth/recover", h.recover)
	auth.HandleFunc("POST /auth/reset", h.reset)
	auth.HandleFunc("POST /auth/password", h.password)
	auth.HandleFunc("POST /auth/logout", h.logout)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/auth/") {
			auth.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			u, ok := h.session(w, r)
			if !ok {
				return
			}
			if time.Since(u.ChangedAt) >= passwordLifetime {
				fail(w, 403, "password_expired")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) find(w http.ResponseWriter, r *http.Request, email string) (user, bool) {
	var u user
	if err := h.users.FindOne(r.Context(), bson.M{"_id": email}).Decode(&u); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			fail(w, 401, "invalid_credentials")
		} else {
			fail(w, 503, "auth_unavailable")
		}
		return u, false
	}
	return u, true
}

func (h *Handler) session(w http.ResponseWriter, r *http.Request) (user, bool) {
	var u user
	header := r.Header.Get("Authorization")
	token := strings.TrimPrefix(header, "Bearer ")
	if !strings.HasPrefix(header, "Bearer ") || len(token) != 64 {
		fail(w, 401, "invalid_session")
		return u, false
	}
	err := h.users.FindOne(r.Context(), bson.M{"session_hash": digest(token), "session_expires": bson.M{"$gt": time.Now()}, "locked": false}).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			fail(w, 401, "invalid_session")
		} else {
			fail(w, 503, "auth_unavailable")
		}
		return u, false
	}
	return u, true
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	v, ok := decode(w, r)
	if !ok {
		return
	}
	u, ok := h.find(w, r, v.Email)
	if !ok {
		return
	}
	if u.Locked {
		fail(w, 423, "account_locked")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(v.Password)) != nil {
		h.badPassword(w, r, &u)
		return
	}
	if time.Now().Before(u.Login.Expires) {
		fail(w, 429, "challenge_pending")
		return
	}
	c, code, err := newChallenge()
	if err != nil {
		fail(w, 503, "auth_unavailable")
		return
	}
	u.Login = c
	if !h.persist(w, r, &u) {
		return
	}
	if h.sendCode(u.Email, c, code) != nil {
		fail(w, 503, "mail_unavailable")
		return
	}
	reply(w, 202, map[string]string{"challenge": c.ID})
}

func (h *Handler) badPassword(w http.ResponseWriter, r *http.Request, u *user) {
	u.Failed++
	status := 401
	if u.Failed >= maxAttempts {
		u.Locked = true
		u.revoke()
		status = 423
	}
	if h.persist(w, r, u) {
		fail(w, status, "invalid_credentials")
	}
}

func (c *challenge) open(id string) bool {
	return c.ID != "" && c.ID == id && time.Now().Before(c.Expires) && c.Attempts < maxAttempts
}

func (c *challenge) matches(code string) bool {
	return subtle.ConstantTimeCompare([]byte(c.Hash), []byte(digest(c.ID+":"+code))) == 1
}

func (h *Handler) checkCode(w http.ResponseWriter, r *http.Request, u *user, c *challenge, v input, recovery bool) bool {
	if (!recovery && u.Locked) || !c.open(v.Challenge) {
		fail(w, 401, "invalid_code")
		return false
	}
	if !c.matches(v.Code) {
		h.wrongCode(w, r, u, c, recovery)
		return false
	}
	return true
}

func (h *Handler) wrongCode(w http.ResponseWriter, r *http.Request, u *user, c *challenge, recovery bool) {
	c.Attempts++
	status := 401
	if c.Attempts >= maxAttempts {
		status = 423
		if !recovery {
			u.Locked = true
			u.revoke()
		}
	}
	if h.persist(w, r, u) {
		fail(w, status, "invalid_code")
	}
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	v, ok := decode(w, r)
	if !ok {
		return
	}
	u, ok := h.find(w, r, v.Email)
	if !ok {
		return
	}
	if !h.checkCode(w, r, &u, &u.Login, v, false) {
		return
	}
	token, err := randomToken()
	if err != nil {
		fail(w, 503, "auth_unavailable")
		return
	}
	u.Login = challenge{}
	u.Failed = 0
	u.SessionHash = digest(token)
	u.SessionExpires = time.Now().Add(sessionLifetime)
	if h.persist(w, r, &u) {
		reply(w, 200, map[string]any{"token": token, "password_change_required": time.Since(u.ChangedAt) >= passwordLifetime})
	}
}

func (h *Handler) recover(w http.ResponseWriter, r *http.Request) {
	v, ok := decode(w, r)
	if !ok {
		return
	}
	if !validEmail(v.Email) {
		fail(w, 400, "invalid_email")
		return
	}
	var u user
	err := h.users.FindOne(r.Context(), bson.M{"_id": v.Email}).Decode(&u)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		fail(w, 503, "auth_unavailable")
		return
	}
	c, code, randomErr := newChallenge()
	if randomErr != nil {
		fail(w, 503, "auth_unavailable")
		return
	}
	if err == nil && !h.startRecovery(w, r, &u, c, code) {
		return
	}
	reply(w, 202, map[string]string{"challenge": c.ID})
}

func validEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email && !strings.ContainsAny(email, "\r\n")
}

func (h *Handler) startRecovery(w http.ResponseWriter, r *http.Request, u *user, c challenge, code string) bool {
	if time.Now().Before(u.Recovery.Expires) {
		fail(w, 429, "challenge_pending")
		return false
	}
	u.Recovery = c
	if !h.persist(w, r, u) {
		return false
	}
	if h.sendCode(u.Email, c, code) != nil {
		fail(w, 503, "mail_unavailable")
		return false
	}
	return true
}

func (h *Handler) reset(w http.ResponseWriter, r *http.Request) {
	v, ok := decode(w, r)
	if !ok {
		return
	}
	u, ok := h.find(w, r, v.Email)
	if !ok {
		return
	}
	if !h.checkCode(w, r, &u, &u.Recovery, v, true) {
		return
	}
	if !h.setPassword(w, &u, v.NewPassword) {
		return
	}
	u.Locked = false
	u.Failed = 0
	u.Recovery = challenge{}
	u.revoke()
	if h.persist(w, r, &u) {
		reply(w, 200, map[string]string{"status": "password_reset"})
	}
}

func (h *Handler) setPassword(w http.ResponseWriter, u *user, password string) bool {
	if !validPassword(password) || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil {
		fail(w, 400, "invalid_new_password")
		return false
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 503, "auth_unavailable")
		return false
	}
	u.PasswordHash = string(hash)
	u.ChangedAt = time.Now()
	return true
}

func (h *Handler) password(w http.ResponseWriter, r *http.Request) {
	u, ok := h.session(w, r)
	if !ok {
		return
	}
	v, ok := decode(w, r)
	if !ok {
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(v.Password)) != nil {
		h.badPassword(w, r, &u)
		return
	}
	if !h.setPassword(w, &u, v.NewPassword) {
		return
	}
	u.Failed = 0
	u.Recovery = challenge{}
	u.revoke()
	if h.persist(w, r, &u) {
		reply(w, 200, map[string]string{"status": "password_changed"})
	}
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	u, ok := h.session(w, r)
	if !ok {
		return
	}
	u.revoke()
	if h.persist(w, r, &u) {
		reply(w, 200, map[string]string{"status": "logged_out"})
	}
}
