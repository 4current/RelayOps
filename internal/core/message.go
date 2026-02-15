package core

import (
	"time"

	"github.com/google/uuid"
)

type Mode string

const (
	ModeAny    Mode = "any"
	ModeTelnet Mode = "telnet"

	ModePacket Mode = "packet"
	ModeARDOP  Mode = "ardop"
	ModeVARAHF Mode = "vara_hf"
	ModeVARAFM Mode = "vara_fm"
)

type TransportIntent struct {
	Allowed   []Mode `json:"allowed,omitempty"`
	Preferred []Mode `json:"preferred,omitempty"`
}

type SessionMode string

const (
	SessionWinlink    SessionMode = "winlink"
	SessionRadioOnly  SessionMode = "radio_only"
	SessionPostOffice SessionMode = "post_office"
	SessionP2P        SessionMode = "p2p"
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
	Status    MessageStatus
	UpdatedAt time.Time
	SentAt    *time.Time
	LastError string
}

type MessageMeta struct {
	Transport         TransportIntent
	Session           SessionMode
	Constraints       Constraints
	AutomationProfile string
	Priority          int
}

type Constraints struct {
	MaxAirTimeSeconds int
	MaxAttachmentSize int64
	PlainTextOnly     bool
}

type MessageStatus string

const (
	StatusDraft   MessageStatus = "draft"
	StatusQueued  MessageStatus = "queued"
	StatusSending MessageStatus = "sending"
	StatusSent    MessageStatus = "sent"
	StatusFailed  MessageStatus = "failed"
)

func NewMessage(subject, body string) *Message {
	now := time.Now()
	return &Message{
		ID:        uuid.NewString(),
		Subject:   subject,
		Body:      body,
		CreatedAt: now,
		UpdatedAt: now,
		Status:    StatusDraft,
		LastError: "",
		Meta:      DefaultMeta(),
	}
}

func DefaultMeta() MessageMeta {
	return MessageMeta{
		Transport: TransportIntent{
			Allowed:   []Mode{ModeAny},
			Preferred: []Mode{},
		},
		Session:           SessionWinlink,
		Constraints:       Constraints{},
		AutomationProfile: "",
		Priority:          0,
	}
}
