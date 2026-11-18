package lab5_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (s *world) login() error {
	var result struct {
		Challenge string
		Token     string
		Code      string
	}
	if err := s.request("POST", "/auth/login", map[string]string{"email": s.email, "password": s.password}, 202, &result, false); err != nil {
		return err
	}
	if result.Challenge == "" || result.Token != "" || result.Code != "" {
		return errors.New("password alone must return only a challenge")
	}
	s.challenge = result.Challenge
	return nil
}

func (s *world) mailCode(challenge string) (string, error) {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		res, err := s.client.Get(os.Getenv("MAIL_URL") + "/api/v1/messages")
		if err != nil {
			return "", errors.New("mail API unavailable")
		}
		var list struct {
			Messages []struct {
				ID      string
				To      []struct{ Address string }
				Subject string
			}
		}
		err = json.NewDecoder(res.Body).Decode(&list)
		res.Body.Close()
		if err != nil || res.StatusCode != 200 {
			return "", errors.New("cannot list test messages")
		}
		for _, msg := range list.Messages {
			if msg.Subject != "VitaLine verification "+challenge {
				continue
			}
			if len(msg.To) != 1 || msg.To[0].Address != s.email {
				continue
			}
			res, err = s.client.Get(os.Getenv("MAIL_URL") + "/api/v1/message/" + msg.ID)
			if err != nil {
				return "", errors.New("cannot read test message")
			}
			var detail struct{ Text string }
			err = json.NewDecoder(res.Body).Decode(&detail)
			res.Body.Close()
			if err != nil {
				return "", errors.New("invalid test message")
			}
			match := regexp.MustCompile(`Code: ([0-9]{6})`).FindStringSubmatch(detail.Text)
			if len(match) != 2 {
				return "", errors.New("email verification code missing")
			}
			s.messageIDs = append(s.messageIDs, msg.ID)
			return match[1], nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", errors.New("verification email not received")
}

func (s *world) verify() error {
	code, err := s.mailCode(s.challenge)
	if err != nil {
		return err
	}
	s.code = code
	var result struct {
		Token                  string
		PasswordChangeRequired *bool `json:"password_change_required"`
	}
	if err = s.request("POST", "/auth/verify", map[string]string{"email": s.email, "challenge": s.challenge, "code": code}, 200, &result, false); err != nil {
		return err
	}
	if result.Token == "" {
		return errors.New("session token missing after second factor")
	}
	if result.PasswordChangeRequired == nil {
		return errors.New("password_change_required missing after second factor")
	}
	s.passwordChangeRequired = *result.PasswordChangeRequired
	s.token = result.Token
	return nil
}

func (s *world) denyPasswords() error {
	for i := 1; i <= 5; i++ {
		status := 401
		if i == 5 {
			status = 423
		}
		if err := s.request("POST", "/auth/login", map[string]string{"email": s.email, "password": randomSecret()}, status, nil, false); err != nil {
			return err
		}
	}
	return nil
}

func (s *world) requestRecovery() error {
	var result struct{ Challenge string }
	if err := s.request("POST", "/auth/recover", map[string]string{"email": s.email}, 202, &result, false); err != nil {
		return err
	}
	s.recovery = result.Challenge
	return nil
}

func (s *world) reset() error {
	code, err := s.mailCode(s.recovery)
	if err != nil {
		return err
	}
	s.recoveryCode = code
	if err = s.request("POST", "/auth/reset", map[string]string{"email": s.email, "challenge": s.recovery, "code": code, "new_password": s.replacement}, 200, nil, false); err != nil {
		return err
	}
	s.password = s.replacement
	return nil
}

func (s *world) changeWith(old, new string, status int) error {
	return s.request("POST", "/auth/password", map[string]string{"password": old, "new_password": new}, status, nil, true)
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

func (s *world) wrongCodes(recovery bool) error {
	challenge, path := s.challenge, "/auth/verify"
	if recovery {
		challenge, path = s.recovery, "/auth/reset"
	}
	actual, err := s.mailCode(challenge)
	if err != nil {
		return err
	}
	wrong := "000000"
	if actual == wrong {
		wrong = "000001"
	}
	for i := 1; i <= 5; i++ {
		status := 401
		if i == 5 {
			status = 423
		}
		if err := s.request("POST", path, map[string]string{"email": s.email, "challenge": challenge, "code": wrong, "new_password": s.replacement}, status, nil, false); err != nil {
			return err
		}
	}
	return nil
}

func (s *world) rejectCode(recovery bool) error {
	challenge, path := s.challenge, "/auth/verify"
	if recovery {
		challenge, path = s.recovery, "/auth/reset"
	}
	code, err := s.mailCode(challenge)
	if err != nil {
		return err
	}
	return s.request("POST", path, map[string]string{"email": s.email, "challenge": challenge, "code": code, "new_password": s.replacement}, 401, nil, false)
}

func (s *world) concurrentVerify() error {
	code, err := s.mailCode(s.challenge)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"email": s.email, "challenge": s.challenge, "code": code})
	type outcome struct {
		status int
		token  string
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			res, err := s.client.Post(os.Getenv("API_URL")+"/auth/verify", "application/json", bytes.NewReader(body))
			if err != nil {
				results <- outcome{}
				return
			}
			defer res.Body.Close()
			var value struct{ Token string }
			_ = json.NewDecoder(res.Body).Decode(&value)
			results <- outcome{res.StatusCode, value.Token}
		}()
	}
	close(start)
	winners := 0
	for i := 0; i < 2; i++ {
		result := <-results
		if result.status == 200 && result.token != "" {
			winners++
			s.token = result.token
		} else if result.status != 401 && result.status != 409 {
			return fmt.Errorf("unexpected concurrent verification status: %d", result.status)
		}
	}
	if winners != 1 {
		return fmt.Errorf("single-use code produced %d sessions", winners)
	}
	return nil
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

func TestAuthentication(t *testing.T) {
	if os.Getenv("LAB5_E2E") != "1" {
		t.Skip("run make test for Docker E2E")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	dbName := os.Getenv("MONGODB_DATABASE")
	if !strings.HasPrefix(dbName, "lab5_") {
		t.Fatal("E2E requires an isolated lab5_ database")
	}
	db, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		t.Fatal("MongoDB connection failed")
	}
	defer db.Disconnect(context.Background())
	if err = db.Ping(ctx, nil); err != nil {
		t.Fatal("MongoDB unavailable")
	}
	users := db.Database(dbName).Collection("auth_users")
	var s *world
	initialize := func(sc *godog.ScenarioContext) {
		sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
			s = &world{users: users, client: &http.Client{Timeout: 10 * time.Second}}
			return ctx, nil
		})
		sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
			s.client.CloseIdleConnections()
			return ctx, s.cleanup()
		})
		sc.Step(`^создан отдельный технический пользователь$`, func() error { return s.create() })
		sc.Step(`^введён правильный пароль$`, func() error { return s.login() })
		sc.Step(`^введён код из письма$`, func() error { return s.verify() })
		sc.Step(`^код одновременно отправлен дважды$`, func() error { return s.concurrentVerify() })
		sc.Step(`^повторный запрос кода отклоняется$`, func() error {
			return s.request("POST", "/auth/login", map[string]string{"email": s.email, "password": s.password}, 429, nil, false)
		})
		sc.Step(`^API клиники недоступен без сессии$`, func() error { return s.request("GET", "/api/patients", nil, 401, nil, false) })
		sc.Step(`^API клиники доступен$`, func() error { return s.request("GET", "/api/patients", nil, 200, nil, true) })
		sc.Step(`^признак обязательной смены пароля равен (true|false)$`, func(expected string) error {
			if s.passwordChangeRequired != (expected == "true") {
				return fmt.Errorf("expected password_change_required=%s, got %t", expected, s.passwordChangeRequired)
			}
			return nil
		})
		sc.Step(`^API требует сменить пароль$`, func() error {
			var result struct{ Error string }
			if err := s.request("GET", "/api/patients", nil, 403, &result, true); err != nil {
				return err
			}
			if result.Error != "password_expired" {
				return errors.New("expected password_expired from protected API")
			}
			return nil
		})
		sc.Step(`^сохранённая сессия недействительна$`, func() error { return s.request("GET", "/api/patients", nil, 401, nil, true) })
		sc.Step(`^повторное использование кода отклоняется$`, func() error { return s.rejectCode(false) })
		sc.Step(`^выполнен выход$`, func() error { return s.request("POST", "/auth/logout", nil, 200, nil, true) })
		sc.Step(`^неверный пароль введён пять раз$`, func() error { return s.denyPasswords() })
		sc.Step(`^правильный пароль также отклоняется$`, func() error {
			return s.request("POST", "/auth/login", map[string]string{"email": s.email, "password": s.password}, 423, nil, false)
		})
		sc.Step(`^запрошено восстановление доступа$`, func() error { return s.requestRecovery() })
		sc.Step(`^пароль восстановлен по коду из письма$`, func() error { return s.reset() })
		sc.Step(`^старый пароль не подходит$`, func() error {
			return s.request("POST", "/auth/login", map[string]string{"email": s.email, "password": s.original}, 401, nil, false)
		})
		sc.Step(`^повторное восстановление тем же кодом отклоняется$`, func() error { return s.rejectCode(true) })
		sc.Step(`^неверный код введён пять раз$`, func() error { return s.wrongCodes(false) })
		sc.Step(`^правильный код также отклоняется$`, func() error { return s.rejectCode(false) })
		sc.Step(`^неверный код восстановления введён пять раз$`, func() error { return s.wrongCodes(true) })
		sc.Step(`^правильный код восстановления также отклоняется$`, func() error { return s.rejectCode(true) })
		sc.Step(`^срок кода истёк$`, func() error { return s.update(bson.M{"login.expires": time.Now().Add(-time.Minute)}) })
		sc.Step(`^срок кода восстановления истёк$`, func() error { return s.update(bson.M{"recovery.expires": time.Now().Add(-time.Minute)}) })
		sc.Step(`^срок сессии истёк$`, func() error { return s.update(bson.M{"session_expires": time.Now().Add(-time.Minute)}) })
		sc.Step(`^пароль не менялся 91 день$`, func() error { return s.update(bson.M{"changed_at": time.Now().Add(-91 * 24 * time.Hour)}) })
		sc.Step(`^срок пароля истечёт через минуту$`, func() error {
			return s.update(bson.M{"changed_at": time.Now().Add(-90*24*time.Hour + time.Minute)})
		})
		sc.Step(`^срок пароля истёк минуту назад$`, func() error {
			return s.update(bson.M{"changed_at": time.Now().Add(-90*24*time.Hour - time.Minute)})
		})
		sc.Step(`^смена пароля с неверным текущим паролем отклоняется$`, func() error { return s.changeWith(randomSecret(), s.replacement, 401) })
		sc.Step(`^повторное назначение текущего пароля отклоняется$`, func() error { return s.changeWith(s.password, s.password, 400) })
		sc.Step(`^выполнена смена пароля$`, func() error {
			if err := s.changeWith(s.password, s.replacement, 200); err != nil {
				return err
			}
			s.password = s.replacement
			return nil
		})
	}
	if err = os.MkdirAll("results", 0755); err != nil {
		t.Fatal(err)
	}
	status := godog.TestSuite{Name: "VitaLine authentication", ScenarioInitializer: initialize, Options: &godog.Options{
		Format: "pretty,junit:results/junit.xml,cucumber:results/cucumber.json", Paths: []string{"features"}, Strict: true, TestingT: t, Concurrency: 1, Output: io.Writer(os.Stdout),
	}}.Run()
	if status != 0 {
		t.Fatal("BDD scenarios failed")
	}
}
