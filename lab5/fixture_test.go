package lab5_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type world struct {
	users                                          *mongo.Collection
	client                                         *http.Client
	email, password, original, replacement         string
	challenge, recovery, code, recoveryCode, token string
	messageIDs                                     []string
	passwordChangeRequired                         bool
}

func randomSecret() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func (s *world) request(method, path string, input any, status int, output any, authenticated bool) error {
	b, err := json.Marshal(input)
	if err != nil {
		return err
	}
	r, err := http.NewRequest(method, os.Getenv("API_URL")+path, bytes.NewReader(b))
	if err != nil {
		return errors.New("invalid API URL")
	}
	r.Header.Set("Content-Type", "application/json")
	if authenticated {
		r.Header.Set("Authorization", "Bearer "+s.token)
	}
	res, err := s.client.Do(r)
	if err != nil {
		return errors.New("API request failed")
	}
	defer res.Body.Close()
	if res.StatusCode != status {
		return fmt.Errorf("%s %s: expected HTTP %d, got %d", method, path, status, res.StatusCode)
	}
	if output != nil {
		return json.NewDecoder(res.Body).Decode(output)
	}
	return nil
}

func (s *world) create() error {
	s.email = "lab5-" + randomSecret() + "@example.test"
	s.password = os.Getenv("LAB5_TEST_PASSWORD")
	if s.password == "" {
		return errors.New("LAB5_TEST_PASSWORD is required (use make test)")
	}
	s.original = s.password
	s.replacement = randomSecret()
	hash, err := bcrypt.GenerateFromPassword([]byte(s.password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("cannot hash technical user password")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = s.users.InsertOne(ctx, bson.M{"_id": s.email, "password_hash": string(hash), "changed_at": time.Now().UTC(), "revision": 0, "failed": 0})
	return err
}

func (s *world) update(fields bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := s.users.UpdateByID(ctx, s.email, bson.M{"$set": fields, "$inc": bson.M{"revision": 1}})
	if err == nil && result.MatchedCount != 1 {
		return errors.New("technical user missing")
	}
	return err
}

func (s *world) cleanup() error {
	if s.email == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := s.users.DeleteOne(ctx, bson.M{"_id": s.email})
	if err != nil {
		return errors.New("technical user cleanup failed")
	}
	n, err := s.users.CountDocuments(ctx, bson.M{"_id": s.email})
	if err != nil || n != 0 {
		return errors.New("technical user was not removed")
	}
	fmt.Println("Technical user deletion verified")
	return nil
}
