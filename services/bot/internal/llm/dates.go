package llm

import (
	"fmt"
	"strings"
	"time"
)

func periodRange(period, fromStr, toStr string, now time.Time, loc *time.Location) (*time.Time, *time.Time, error) {
	now = now.In(loc)
	period = strings.ToLower(strings.TrimSpace(period))

	switch period {
	case "", "all":
		// optional custom bounds below
	case "today", "сегодня":
		d := dateOnly(now, loc)
		return &d, &d, nil
	case "yesterday", "вчера":
		d := dateOnly(now.AddDate(0, 0, -1), loc)
		return &d, &d, nil
	case "week", "неделя", "за неделю":
		from, to := weekRange(now, loc)
		return &from, &to, nil
	case "month", "месяц", "этот месяц":
		from, to := monthRange(now, loc)
		return &from, &to, nil
	case "custom":
		// handled via from/to
	default:
		return nil, nil, fmt.Errorf("unknown period %q", period)
	}

	var from, to *time.Time
	if fromStr != "" {
		d, err := parseDate(fromStr, now, loc)
		if err != nil {
			return nil, nil, err
		}
		from = &d
	}
	if toStr != "" {
		d, err := parseDate(toStr, now, loc)
		if err != nil {
			return nil, nil, err
		}
		to = &d
	}
	return from, to, nil
}

func weekRange(now time.Time, loc *time.Location) (from, to time.Time) {
	now = now.In(loc)
	to = dateOnly(now, loc)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	from = dateOnly(now.AddDate(0, 0, -(weekday-1)), loc)
	return from, to
}

func monthRange(now time.Time, loc *time.Location) (from, to time.Time) {
	now = now.In(loc)
	from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	to = dateOnly(now, loc)
	return from, to
}

func parseDate(value string, now time.Time, loc *time.Location) (time.Time, error) {
	now = now.In(loc)
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "today", "сегодня":
		return dateOnly(now, loc), nil
	case "yesterday", "вчера":
		return dateOnly(now.AddDate(0, 0, -1), loc), nil
	default:
		t, err := time.ParseInLocation("2006-01-02", value, loc)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid date %q", value)
		}
		return t, nil
	}
}

func dateOnly(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}
