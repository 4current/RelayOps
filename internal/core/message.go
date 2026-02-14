package core

import (
	"time"

	"github.com/google/uuid"
)

type Address struct {
	Callsign string
	Email    string
}

type Message struct {
	ID        string
	Subject   string
	Body      string
	To        []Address
	From      Address
	Tags      []string
	CreatedAt time.Time
	Meta      MessageMeta
}

type MessageMeta struct {
	PreferredTransports []string
	Constraints         Constraints
	AutomationProfile   string
	Priority            int
}

type Constraints struct {
	MaxAirTimeSeconds int
	MaxAttachmentSize int64
	PlainTextOnly     bool
}

func NewMessage(subject, body string) *Message {
	return &Message{
		ID:        uuid.NewString(),
		Subject:   subject,
		Body:      body,
		CreatedAt: time.Now(),
		Meta:      DefaultMeta(),
	}
}

func DefaultMeta() MessageMeta {
	return MessageMeta{
		PreferredTransports: []string{},
		Constraints:         Constraints{},
		AutomationProfile:   "",
		Priority:            0,
	}
}
