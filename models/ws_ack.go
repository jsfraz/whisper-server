package models

import "time"

// DeliveryAck confirms to the sender that the message is stored in the archive
// (and, when the recipient was online, already handed to their socket). The app
// flips the message from "sending" to "sent" only when it arrives.
type DeliveryAck struct {
	Id         string    `json:"id"`
	Seq        int64     `json:"seq"`
	ReceiverId string    `json:"receiverId"`
	SentAt     time.Time `json:"sentAt"`
}
