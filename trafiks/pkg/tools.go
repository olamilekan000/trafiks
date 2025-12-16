package pkg

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func ToSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9\s-]`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, " ", "-")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")

	return s
}

// Convert memory string (e.g., "2GB") to bytes
func ConvertToBytes(memoryStr string) (int64, error) {
	memoryStr = strings.ToUpper(strings.TrimSpace(memoryStr))

	// Define units in decreasing order to avoid partial matches (e.g., "GB" vs "B")
	units := []struct {
		Suffix     string
		Multiplier int64
	}{
		{"PB", 1024 * 1024 * 1024 * 1024 * 1024},
		{"TB", 1024 * 1024 * 1024 * 1024},
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}

	for _, unit := range units {
		if strings.HasSuffix(memoryStr, unit.Suffix) {
			valueStr := strings.TrimSuffix(memoryStr, unit.Suffix)
			value, err := strconv.ParseFloat(strings.TrimSpace(valueStr), 64)
			if err != nil {
				return 0, fmt.Errorf("invalid memory format: %w", err)
			}
			if value <= 0 {
				return 0, fmt.Errorf("memory value must be greater than zero")
			}
			return int64(value * float64(unit.Multiplier)), nil
		}
	}

	return 0, fmt.Errorf("unsupported memory unit in %q", memoryStr)
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	cost := 14

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

func GenerateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func FriendlyDuration(d time.Duration) string {
	minutes := int(d.Minutes())

	switch {
	case minutes < 60:
		return fmt.Sprintf("%d minute%s", minutes, Pluralize(minutes))
	case minutes%60 == 0:
		hours := minutes / 60
		return fmt.Sprintf("%d hour%s", hours, Pluralize(hours))
	default:
		hours := minutes / 60
		mins := minutes % 60
		return fmt.Sprintf("%d hour%s %d minute%s", hours, Pluralize(hours), mins, Pluralize(mins))
	}
}

func Pluralize(n int) string {
	if n == 1 {
		return ""
	}

	return "s"
}

func BoolPtr(b bool) *bool {
	return &b
}

func ToBoolValue(b *bool) bool {
	if b == nil {
		return false
	}

	return *b
}

func FirstDayOfMonth() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}

func LastDayOfMonth() time.Time {
	firstDay := FirstDayOfMonth()
	return firstDay.AddDate(0, 1, 0).Add(-time.Nanosecond)
}

func TotalHoursInMonth() float64 {
	firstDay := FirstDayOfMonth()
	lastDay := LastDayOfMonth()
	return lastDay.Sub(firstDay).Hours()
}

func HourlyRate(fullMonthlyCost float64) float64 {
	totalHours := TotalHoursInMonth()
	if totalHours <= 0 {
		return 0
	}

	hourly := fullMonthlyCost / float64(totalHours)
	return math.Round(hourly*100) / 100
}

func ExtractSizeInGB(storage string) (int, error) {
	// Remove non-digit suffix (like "GB", "GiB", etc.)
	sizeStr := strings.TrimRight(storage, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	return strconv.Atoi(sizeStr)
}

func GenerateSecurePassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

	b := make([]byte, 16)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}

	return string(b)
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}

	return false
}

func GenerateUUIDV7() string {
	if id, err := uuid.NewV7(); err == nil {
		return id.String()
	}

	return uuid.NewString()
}
