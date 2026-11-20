package lab5_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"
)

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
