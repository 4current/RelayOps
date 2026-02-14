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
