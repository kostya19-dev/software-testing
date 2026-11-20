package lab5_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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
