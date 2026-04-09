package utils

import (
	"fmt"
	"net/smtp"
	"strings"

	"marvaron/internal/config"
)

func smtpConfigured() bool {
	e := config.AppConfig.Email
	return e.SMTPHost != "" && e.From != ""
}

// SendEmail sends a plain-text email when SMTP is configured; otherwise logs the body.
func SendEmail(to, subject, body string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("empty recipient")
	}
	e := config.AppConfig.Email
	if !smtpConfigured() {
		fmt.Printf("[email:dev] To: %s\nSubject: %s\n%s\n", to, subject, body)
		return nil
	}

	addr := fmt.Sprintf("%s:%s", e.SMTPHost, e.SMTPPort)
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		e.From, to, subject, body))

	var auth smtp.Auth
	if e.SMTPUser != "" {
		auth = smtp.PlainAuth("", e.SMTPUser, e.SMTPPassword, e.SMTPHost)
	}
	return smtp.SendMail(addr, auth, e.From, []string{to}, msg)
}

// SendOTPEmail delivers a signup / verification code to the user's inbox.
func SendOTPEmail(to, otp string) error {
	subject := "Your verification code"
	body := fmt.Sprintf("Your verification code is: %s\n\nIt expires in %d minutes.",
		otp, config.AppConfig.OTP.ExpiryMinutes)
	return SendEmail(to, subject, body)
}

// SendPasswordResetEmail sends a link the user can open to set a new password.
func SendPasswordResetEmail(to, resetURL string) error {
	subject := "Reset your password"
	body := fmt.Sprintf("You requested a password reset.\n\nOpen this link to choose a new password (valid for 1 hour):\n%s\n\nIf you did not request this, you can ignore this email.", resetURL)
	return SendEmail(to, subject, body)
}
