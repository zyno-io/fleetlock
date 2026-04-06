package fleetlock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseMaintenanceTime(t *testing.T) {
	tests := []struct {
		input   string
		hour    int
		minute  int
		wantErr bool
	}{
		{"02:00", 2, 0, false},
		{"23:59", 23, 59, false},
		{"00:00", 0, 0, false},
		{"25:00", 0, 0, true},
		{"12:60", 0, 0, true},
		{"invalid", 0, 0, true},
		{"", 0, 0, true},
	}
	for _, tt := range tests {
		mt, err := parseMaintenanceTime(tt.input)
		if tt.wantErr {
			assert.Error(t, err, "input: %s", tt.input)
		} else {
			assert.NoError(t, err, "input: %s", tt.input)
			assert.Equal(t, tt.hour, mt.hour)
			assert.Equal(t, tt.minute, mt.minute)
		}
	}
}

func TestInMaintenanceWindow(t *testing.T) {
	tests := []struct {
		name   string
		now    time.Time
		start  string
		end    string
		expect bool
	}{
		{
			name:   "within same-day window",
			now:    time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC),
			start:  "02:00",
			end:    "06:00",
			expect: true,
		},
		{
			name:   "before same-day window",
			now:    time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
			start:  "02:00",
			end:    "06:00",
			expect: false,
		},
		{
			name:   "after same-day window",
			now:    time.Date(2024, 1, 1, 7, 0, 0, 0, time.UTC),
			start:  "02:00",
			end:    "06:00",
			expect: false,
		},
		{
			name:   "within cross-midnight window (before midnight)",
			now:    time.Date(2024, 1, 1, 23, 0, 0, 0, time.UTC),
			start:  "22:00",
			end:    "06:00",
			expect: true,
		},
		{
			name:   "within cross-midnight window (after midnight)",
			now:    time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC),
			start:  "22:00",
			end:    "06:00",
			expect: true,
		},
		{
			name:   "outside cross-midnight window",
			now:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			start:  "22:00",
			end:    "06:00",
			expect: false,
		},
		{
			name:   "at exact start time (inclusive)",
			now:    time.Date(2024, 1, 1, 2, 0, 0, 0, time.UTC),
			start:  "02:00",
			end:    "06:00",
			expect: true,
		},
		{
			name:   "at exact end time (exclusive)",
			now:    time.Date(2024, 1, 1, 6, 0, 0, 0, time.UTC),
			start:  "02:00",
			end:    "06:00",
			expect: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, _ := parseMaintenanceTime(tt.start)
			end, _ := parseMaintenanceTime(tt.end)
			result := inMaintenanceWindow(tt.now, start, end)
			assert.Equal(t, tt.expect, result)
		})
	}
}
