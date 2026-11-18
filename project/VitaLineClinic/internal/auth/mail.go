package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"net/smtp"
	"time"
)

func newChallenge() (challenge, string, error) {
	id, err := randomToken()
	if err != nil {
		return challenge{}, "", err
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return challenge{}, "", err
	}
	code := fmt.Sprintf("%06d", n.Int64())
	return challenge{ID: id, Hash: digest(id + ":" + code), Expires: time.Now().Add(codeLifetime)}, code, nil
}

func (h *Handler) sendCode(email string, c challenge, code string) error {
	host, _, err := net.SplitHostPort(h.smtp)
	if err != nil {
		return err
	}
	conn, err := net.DialTimeout("tcp", h.smtp, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err = conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if err = client.Mail("noreply@vitaline.test"); err != nil {
		return err
	}
	if err = client.Rcpt(email); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "From: noreply@vitaline.test\r\nTo: %s\r\nSubject: VitaLine verification %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nCode: %s\r\nExpires in 5 minutes.\r\n", email, c.ID, code)
	if err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return client.Quit()
}
