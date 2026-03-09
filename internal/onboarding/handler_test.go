package onboarding

import (
	"testing"
	"time"

	tele "gopkg.in/telebot.v4"
)

func TestCalcAge_BirthdayAlreadyPassed(t *testing.T) {
	now := time.Now()
	bd := tele.Birthdate{
		Year:  now.Year() - 25,
		Month: int(now.Month()) - 1,
		Day:   1,
	}
	// If month already passed, no need to worry about month=0.
	if now.Month() == time.January {
		bd.Month = 12
		bd.Year = now.Year() - 26
	}
	got := calcAge(bd)
	want := now.Year() - bd.Year
	if got != want {
		t.Errorf("calcAge(birthday already passed) = %d, want %d", got, want)
	}
}

func TestCalcAge_BirthdayNotYetThisYear(t *testing.T) {
	now := time.Now()
	bd := tele.Birthdate{
		Year:  now.Year() - 25,
		Month: int(now.Month()) + 1,
		Day:   1,
	}
	if now.Month() == time.December {
		// Birthday next month would be Jan next year — test a future month instead.
		bd.Month = 12
		bd.Day = 31
		if now.Day() == 31 {
			bd.Day = 30 // edge: if today is Dec 31, use Dec 30 so it's still "not yet"
			// Actually Dec 30 < Dec 31 so birthday passed. Use a different approach.
			bd.Month = int(now.Month())
			bd.Day = now.Day() + 1
			// This won't work if Day > days in month, but Dec has 31 days.
		}
	}
	got := calcAge(bd)
	want := now.Year() - bd.Year - 1
	if got != want {
		t.Errorf("calcAge(birthday not yet) = %d, want %d", got, want)
	}
}

func TestCalcAge_BirthdayToday(t *testing.T) {
	now := time.Now()
	bd := tele.Birthdate{
		Year:  now.Year() - 30,
		Month: int(now.Month()),
		Day:   now.Day(),
	}
	got := calcAge(bd)
	want := 30
	if got != want {
		t.Errorf("calcAge(birthday today) = %d, want %d", got, want)
	}
}

func TestCalcAge_SameDayLaterMonth(t *testing.T) {
	// Same day number, but month hasn't arrived yet.
	now := time.Now()
	futureMonth := int(now.Month()) + 2
	year := now.Year() - 20
	if futureMonth > 12 {
		futureMonth -= 12
	}
	bd := tele.Birthdate{Year: year, Month: futureMonth, Day: now.Day()}
	got := calcAge(bd)
	want := now.Year() - year - 1
	if got != want {
		t.Errorf("calcAge(same day, later month) = %d, want %d", got, want)
	}
}

func TestCalcAge_SameMonthLaterDay(t *testing.T) {
	now := time.Now()
	futureDay := now.Day() + 5
	if futureDay > 28 { // Use a safe day to avoid month overflow.
		t.Skip("too close to end of month for reliable test")
	}
	bd := tele.Birthdate{Year: now.Year() - 22, Month: int(now.Month()), Day: futureDay}
	got := calcAge(bd)
	want := 21 // birthday hasn't happened yet this year
	if got != want {
		t.Errorf("calcAge(same month, later day) = %d, want %d", got, want)
	}
}

func TestCalcAge_Young(t *testing.T) {
	now := time.Now()
	// 16 years old, birthday already passed.
	bd := tele.Birthdate{Year: now.Year() - 16, Month: 1, Day: 1}
	if now.Month() == time.January && now.Day() == 1 {
		bd.Day = 1 // birthday is today, still 16
	}
	got := calcAge(bd)
	if got != 16 {
		t.Errorf("calcAge(16yo) = %d, want 16", got)
	}
}
