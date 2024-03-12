package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/mail"
	"os"
	"os/exec"
	"os/user"

	"github.com/emersion/go-smtp"
	"olowe.co/apub"
)

type Backend struct {
	auth string
}

func (be *Backend) NewSession(conn *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}

type Session struct {
	recipients []string
	User       *user.User
}

func (s *Session) AuthPlain(username, password string) error {
	u, err := user.Lookup(username)
	if err != nil {
		return errors.New("invalid username or password")
	}
	// TODO allow other users except for me lol
	// TODO implement BSD Auth and/or PAM?
	if u.Username != "otl" {
		return errors.New("invalid username or password")
	}
	if password != "yamum" {
		return errors.New("invalid username or password")
	}
	s.User = u
	return nil
}

func (s *Session) Logout() error { return nil }
func (s *Session) Reset()        {}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	log.Println("MAIL FROM:", from)
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	log.Println("RCPT TO:", to)
	addr, err := mail.ParseAddress(to)
	if err != nil {
		return err
	}
	if _, err = apub.Finger(addr.Address); err != nil {
		return err
	}
	s.recipients = append(s.recipients, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	args := append([]string{"-F"}, s.recipients...)
	cmd := exec.Command("apsend", args...)
	cmd.Stdin = r
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err1, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("execute mailer: %v", string(err1.Stderr))
	} else if err != nil {
		return fmt.Errorf("execute mailer: %v", err)
	}
	return nil
}
