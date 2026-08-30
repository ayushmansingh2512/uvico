package mailer

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// SendBookingNotification sends an email notification to both the host and the attendee
func SendBookingNotification(attendeeEmail, hostEmail, hostName string, startTime, endTime time.Time) error {
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "smtp.gmail.com"
	}
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "587"
	}
	senderEmail := strings.TrimSpace(os.Getenv("SMTP_EMAIL"))
	senderPass := strings.ReplaceAll(strings.TrimSpace(os.Getenv("SMTP_PASSWORD")), " ", "")

	if senderEmail == "" || senderPass == "" {
		log.Println("[Mailer] Note: SMTP_EMAIL and SMTP_PASSWORD not set in .env. Skipping email dispatch.")
		return nil
	}

	if hostEmail == "" {
		hostEmail = "ayushmansingh2512@gmail.com"
	}
	if hostName == "" {
		hostName = "Ayushman Singh"
	}

	subject := fmt.Sprintf("Meeting Confirmed: Call with %s on %s", hostName, startTime.Format("Monday, Jan 02 at 03:04 PM"))

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #121212; color: #ffffff; margin: 0; padding: 24px; }
  .card { max-width: 520px; margin: 0 auto; background-color: #242424; border: 1px solid rgba(255,255,255,0.12); border-radius: 12px; padding: 28px; box-shadow: 0 12px 32px rgba(0,0,0,0.6); }
  h2 { color: #f3e9d6; margin-top: 0; font-size: 20px; border-bottom: 1px solid rgba(255,255,255,0.1); padding-bottom: 12px; }
  .detail-row { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid rgba(255,255,255,0.06); font-size: 14px; }
  .label { color: #8a8a8e; font-weight: 500; }
  .val { color: #ffffff; font-weight: 600; }
  .footer { margin-top: 24px; font-size: 12px; color: #6b6b70; text-align: center; }
</style>
</head>
<body>
  <div class="card">
    <h2>Meeting Scheduled</h2>
    <p style="color:#d1d1d6; font-size:14px; margin-bottom:20px;">Your meeting has been confirmed with %s via AI Copilot.</p>
    
    <div class="detail-row">
      <span class="label">Date:</span>
      <span class="val">%s</span>
    </div>
    <div class="detail-row">
      <span class="label">Time:</span>
      <span class="val">%s - %s</span>
    </div>
    <div class="detail-row">
      <span class="label">Host:</span>
      <span class="val">%s (%s)</span>
    </div>
    <div class="detail-row">
      <span class="label">Attendee:</span>
      <span class="val">%s</span>
    </div>

    <div class="footer">Universal AI Copilot • Automated Scheduling</div>
  </div>
</body>
</html>`,
		hostName,
		startTime.Format("Monday, January 02, 2006"),
		startTime.Format("03:04 PM"),
		endTime.Format("03:04 PM"),
		hostName,
		hostEmail,
		attendeeEmail,
	)

	body := fmt.Sprintf("From: AI Copilot <%s>\r\n"+
		"To: %s, %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n\r\n"+
		"%s",
		senderEmail, hostEmail, attendeeEmail, subject, htmlContent)

	auth := smtp.PlainAuth("", senderEmail, senderPass, smtpHost)

	// 1. Send confirmation to host
	if err := smtp.SendMail(smtpHost+":"+smtpPort, auth, senderEmail, []string{hostEmail}, []byte(body)); err != nil {
		log.Printf("[Mailer] ❌ Error sending email to host (%s): %v", hostEmail, err)
	} else {
		log.Printf("[Mailer] ✅ Confirmation email successfully sent to host (%s)", hostEmail)
	}

	// 2. Send confirmation to attendee
	if attendeeEmail != "" && attendeeEmail != hostEmail {
		if err := smtp.SendMail(smtpHost+":"+smtpPort, auth, senderEmail, []string{attendeeEmail}, []byte(body)); err != nil {
			log.Printf("[Mailer] ❌ Error sending email to attendee (%s): %v", attendeeEmail, err)
		} else {
			log.Printf("[Mailer] ✅ Confirmation email successfully sent to attendee (%s)", attendeeEmail)
		}
	}

	return nil
}
