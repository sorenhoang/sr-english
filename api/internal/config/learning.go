package config

import (
	"fmt"
	"os"
	"strconv"
)

// Defaults for learning limits; each can be overridden by the env var named on its field.
const (
	DefaultDailyNew         = 10   // matches the "10 từ mới" design
	DefaultDailyNewMax      = 50   // keeps a day's session reviewable on a phone
	DefaultMaxReviewsPerDay = 200  // caps the backlog after missed days
	DefaultDesiredRetention = 0.9  // go-fsrs default
	DefaultCoverage         = 0.95 // Hu & Nation (2000): minimum known-word coverage for reading
)

type Learning struct {
	DailyNewDefault  int     // DAILY_NEW_DEFAULT
	DailyNewMax      int     // DAILY_NEW_MAX
	MaxReviewsPerDay int     // MAX_REVIEWS_PER_DAY
	DesiredRetention float64 // DESIRED_RETENTION
	Coverage         float64 // COVERAGE_THRESHOLD
}

func DefaultLearning() Learning {
	return Learning{
		DailyNewDefault:  DefaultDailyNew,
		DailyNewMax:      DefaultDailyNewMax,
		MaxReviewsPerDay: DefaultMaxReviewsPerDay,
		DesiredRetention: DefaultDesiredRetention,
		Coverage:         DefaultCoverage,
	}
}

func loadLearning() (Learning, error) {
	l := DefaultLearning()
	ints := []struct {
		key string
		dst *int
	}{{"DAILY_NEW_DEFAULT", &l.DailyNewDefault}, {"DAILY_NEW_MAX", &l.DailyNewMax}, {"MAX_REVIEWS_PER_DAY", &l.MaxReviewsPerDay}}
	for _, it := range ints {
		if v := os.Getenv(it.key); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return Learning{}, fmt.Errorf("%s must be a positive integer", it.key)
			}
			*it.dst = n
		}
	}
	floats := []struct {
		key string
		dst *float64
	}{{"DESIRED_RETENTION", &l.DesiredRetention}, {"COVERAGE_THRESHOLD", &l.Coverage}}
	for _, it := range floats {
		if v := os.Getenv(it.key); v != "" {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil || f <= 0 || f >= 1 {
				return Learning{}, fmt.Errorf("%s must be between 0 and 1", it.key)
			}
			*it.dst = f
		}
	}
	if l.DailyNewDefault > l.DailyNewMax {
		return Learning{}, fmt.Errorf("DAILY_NEW_DEFAULT (%d) exceeds DAILY_NEW_MAX (%d)", l.DailyNewDefault, l.DailyNewMax)
	}
	return l, nil
}
