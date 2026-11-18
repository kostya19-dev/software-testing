package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	maxAttempts      = 5
	codeLifetime     = 5 * time.Minute
	passwordLifetime = 90 * 24 * time.Hour
	sessionLifetime  = 30 * time.Minute
)

type challenge struct {
	ID       string    `bson:"id"`
	Hash     string    `bson:"hash"`
	Expires  time.Time `bson:"expires"`
	Attempts int       `bson:"attempts"`
}

type user struct {
	Email          string    `bson:"_id"`
	PasswordHash   string    `bson:"password_hash"`
	ChangedAt      time.Time `bson:"changed_at"`
	Revision       int       `bson:"revision"`
	Failed         int       `bson:"failed"`
	Locked         bool      `bson:"locked"`
	Login          challenge `bson:"login"`
	Recovery       challenge `bson:"recovery"`
	SessionHash    string    `bson:"session_hash"`
	SessionExpires time.Time `bson:"session_expires"`
}

type Handler struct {
	client *mongo.Client
	users  *mongo.Collection
	smtp   string
}

func Open(ctx context.Context) (*Handler, error) {
	uri, name, smtp := os.Getenv("DATABASE_URL"), os.Getenv("MONGODB_DATABASE"), os.Getenv("VITALINE_AUTH_SMTP")
	if uri == "" || name == "" || smtp == "" {
		return nil, errors.New("authentication requires DATABASE_URL, MONGODB_DATABASE and VITALINE_AUTH_SMTP")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, errors.New("authentication database connection failed")
	}
	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, errors.New("authentication database unavailable")
	}
	h := &Handler{client: client, users: client.Database(name).Collection("auth_users"), smtp: smtp}
	_, err = h.users.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "session_hash", Value: 1}}})
	if err != nil {
		_ = client.Disconnect(ctx)
		return nil, errors.New("authentication index creation failed")
	}
	return h, nil
}

func (h *Handler) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = h.client.Disconnect(ctx)
}

func (h *Handler) save(ctx context.Context, u *user) error {
	previous := u.Revision
	u.Revision++
	res, err := h.users.ReplaceOne(ctx, bson.M{"_id": u.Email, "revision": previous}, u)
	if err != nil {
		return err
	}
	if res.MatchedCount != 1 {
		return errConflict
	}
	return nil
}

var errConflict = errors.New("authentication state changed; retry")

func digest(value string) string { b := sha256.Sum256([]byte(value)); return hex.EncodeToString(b[:]) }

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (u *user) revoke() { u.SessionHash = ""; u.SessionExpires = time.Time{}; u.Login = challenge{} }
