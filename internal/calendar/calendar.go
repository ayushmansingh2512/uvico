package calendar

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"universal-copilot/internal/mailer"
)

var (
	calendarService *calendar.Service
)

// InitCalendarService initializes the Google Calendar client using service-account.json
func InitCalendarService(ctx context.Context) error {
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsPath == "" {
		credsPath = "service-account.json"
	}

	if _, err := os.Stat(credsPath); os.IsNotExist(err) {
		return fmt.Errorf("credentials file not found at %s", credsPath)
	}

	srv, err := calendar.NewService(ctx, option.WithCredentialsFile(credsPath), option.WithScopes(calendar.CalendarScope))
	if err != nil {
		return fmt.Errorf("failed to create calendar service: %w", err)
	}

	calendarService = srv
	return nil
}

// GetFreeBusy retrieves busy periods for the calendar within a time range
func GetFreeBusy(ctx context.Context, calendarID string, start, end time.Time) ([]*calendar.TimePeriod, error) {
	if calendarService == nil {
		if err := InitCalendarService(ctx); err != nil {
			return nil, err
		}
	}

	req := &calendar.FreeBusyRequest{
		TimeMin: start.Format(time.RFC3339),
		TimeMax: end.Format(time.RFC3339),
		Items: []*calendar.FreeBusyRequestItem{
			{Id: calendarID},
		},
	}

	resp, err := calendarService.Freebusy.Query(req).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to query freebusy: %w", err)
	}

	if cal, ok := resp.Calendars[calendarID]; ok {
		return cal.Busy, nil
	}

	return nil, nil
}

// GetAvailableSlotsSummary calculates available slots and busy periods for the upcoming days
func GetAvailableSlotsSummary(ctx context.Context, calendarID string, daysAhead int) string {
	if calendarID == "" {
		calendarID = os.Getenv("CALENDAR_OWNER_EMAIL")
		if calendarID == "" {
			calendarID = "ayushmansingh2512@gmail.com"
		}
	}

	loc, _ := time.LoadLocation("Local")
	now := time.Now().In(loc)

	start := now
	end := now.AddDate(0, 0, daysAhead)

	busyPeriods, err := GetFreeBusy(ctx, calendarID, start, end)
	if err != nil {
		return fmt.Sprintf("Live Calendar availability check note: (%v). Default availability: Mon-Fri between 10:00 AM – 06:00 PM.", err)
	}

	if len(busyPeriods) == 0 {
		return fmt.Sprintf("Current Live Calendar Status (checked at %s): Open schedule with no booked conflicts between 10:00 AM – 06:00 PM for the upcoming %d days.", now.Format("03:04 PM"), daysAhead)
	}

	summary := fmt.Sprintf("Current Live Calendar Status (checked at %s):\nBooked/Busy intervals:\n", now.Format("03:04 PM"))
	for _, b := range busyPeriods {
		st, _ := time.Parse(time.RFC3339, b.Start)
		en, _ := time.Parse(time.RFC3339, b.End)
		summary += fmt.Sprintf("- Busy from %s to %s\n", st.In(loc).Format("Mon Jan 02 at 03:04 PM"), en.In(loc).Format("03:04 PM"))
	}
	summary += "General working hours: 10:00 AM to 06:00 PM. Recommend open non-conflicting 30-min slots."

	return summary
}

// CheckAndBookMeeting parses user message for booking intent & email, then books on Google Calendar
func CheckAndBookMeeting(ctx context.Context, calendarID, hostName, userMsg string) (bool, string, error) {
	if calendarID == "" {
		calendarID = os.Getenv("CALENDAR_OWNER_EMAIL")
		if calendarID == "" {
			calendarID = "ayushmansingh2512@gmail.com"
		}
	}
	if hostName == "" {
		hostName = "Ayushman Singh"
	}

	// 1. Detect Email
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	email := emailRegex.FindString(userMsg)
	if email == "" {
		return false, "", nil
	}

	lower := strings.ToLower(userMsg)
	// Check if there is time/date intent (pm, am, tomorrow, today, monday, etc.)
	hasTimeIntent := strings.Contains(lower, "pm") || strings.Contains(lower, "am") ||
		strings.Contains(lower, ":00") || strings.Contains(lower, ":30") ||
		strings.Contains(lower, "tomorrow") || strings.Contains(lower, "today") ||
		strings.Contains(lower, "schedule") || strings.Contains(lower, "book") ||
		strings.Contains(lower, "meet") || strings.Contains(lower, "call")

	if !hasTimeIntent {
		return false, "", nil
	}

	loc, _ := time.LoadLocation("Local")
	now := time.Now().In(loc)

	// Target date: calculate for tomorrow, today, or specific weekday (Monday, Tuesday, etc.)
	targetDate := now
	if strings.Contains(lower, "tomorrow") {
		targetDate = now.AddDate(0, 0, 1)
	} else {
		weekdays := map[string]time.Weekday{
			"sunday":    time.Sunday,
			"monday":    time.Monday,
			"tuesday":   time.Tuesday,
			"wednesday": time.Wednesday,
			"thursday":  time.Thursday,
			"friday":    time.Friday,
			"saturday":  time.Saturday,
		}
		for name, targetWd := range weekdays {
			if strings.Contains(lower, name) {
				currentWd := int(now.Weekday())
				daysToAdd := (int(targetWd) - currentWd + 7) % 7
				if daysToAdd == 0 {
					daysToAdd = 7 // next week
				}
				targetDate = now.AddDate(0, 0, daysToAdd)
				break
			}
		}
	}

	// Parse hour (e.g. 3 pm, 3:00 pm, 11 am, 15:00)
	targetHour := 15 // default 3 PM
	targetMin := 0

	timeRegex := regexp.MustCompile(`(\d{1,2})(?::(\d{2}))?\s*(am|pm)?`)
	matches := timeRegex.FindAllStringSubmatch(lower, -1)
	for _, m := range matches {
		if len(m) > 1 {
			h, err := strconv.Atoi(m[1])
			if err == nil {
				if len(m) > 2 && m[2] != "" {
					targetMin, _ = strconv.Atoi(m[2])
				}
				if len(m) > 3 && m[3] == "pm" && h < 12 {
					h += 12
				} else if len(m) > 3 && m[3] == "am" && h == 12 {
					h = 0
				}
				targetHour = h
				break
			}
		}
	}

	startTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), targetHour, targetMin, 0, 0, loc)
	endTime := startTime.Add(30 * time.Minute)

	log.Printf("[Calendar] Auto-booking meeting on %s for %s (%s) at %s", calendarID, hostName, email, startTime.Format("Mon Jan 02 at 03:04 PM"))

	summary := fmt.Sprintf("Call with %s (%s)", hostName, email)
	description := fmt.Sprintf("Meeting scheduled via Universal AI Copilot with %s. Attendee email: %s", hostName, email)

	_, err := BookMeeting(ctx, calendarID, summary, description, email, startTime, endTime)
	if err != nil {
		log.Printf("[Calendar] Error booking meeting: %v", err)
		return false, "", err
	}

	// Dispatch email notification asynchronously to both parties
	go mailer.SendBookingNotification(email, calendarID, hostName, startTime, endTime)

	reply := fmt.Sprintf("Awesome! Your meeting has been officially scheduled with %s for %s (%s - %s).\n\nA confirmation email has been dispatched to %s. Looking forward to speaking with you!",
		hostName,
		startTime.Format("Monday, Jan 02"),
		startTime.Format("03:04 PM"),
		endTime.Format("03:04 PM"),
		email,
	)

	return true, reply, nil
}

// BookMeeting creates a Google Calendar event on the shared calendar
func BookMeeting(ctx context.Context, calendarID, summary, description, attendeeEmail string, start, end time.Time) (*calendar.Event, error) {
	if calendarService == nil {
		if err := InitCalendarService(ctx); err != nil {
			return nil, err
		}
	}
	if calendarID == "" {
		calendarID = os.Getenv("CALENDAR_OWNER_EMAIL")
		if calendarID == "" {
			calendarID = "ayushmansingh2512@gmail.com"
		}
	}

	event := &calendar.Event{
		Summary:     fmt.Sprintf("%s (%s)", summary, attendeeEmail),
		Description: fmt.Sprintf("%s\n\nContact / Attendee: %s", description, attendeeEmail),
		Start: &calendar.EventDateTime{
			DateTime: start.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: end.Format(time.RFC3339),
		},
	}

	createdEvent, err := calendarService.Events.Insert(calendarID, event).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to insert calendar event: %w", err)
	}

	return createdEvent, nil
}
