package internal

import (
	"encoding/json"
	"time"
)

type ScheduleType string

const (
	ScheduleTypeCron     ScheduleType = "cron"
	ScheduleTypeAtTimes  ScheduleType = "at_times"
	ScheduleTypeDuration ScheduleType = "duration"
)

type Schedule struct {
	ID                 string             `json:"id" bson:"_id"`
	Name               string             `json:"name" bson:"name"`
	Type               ScheduleType       `json:"type" bson:"type"`
	ScheduleDefinition ScheduleDefinition `json:"definition" bson:"definition"`
}

type ScheduleDefinition struct {
	CronExpression string         `json:"cron_expression" bson:"cron_expression"`
	Times          []time.Time    `json:"times" bson:"times"`
	Interval       StringDuration `json:"interval" bson:"interval"`
}

// StringDuration is a custom type for marshaling/unmarshaling time.Duration as a string.
type StringDuration time.Duration

// MarshalJSON implements the json.Marshaler interface.
func (sd StringDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(sd).String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (sd *StringDuration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*sd = StringDuration(d)
	return nil
}

type MyStruct struct {
	Duration StringDuration `json:"duration"`
}
