package models

import "time"

// DeliveryAck is sent to the sender once a message was either handed to the
// receiver's WebSocket connection or persisted in the receiver's offline queue.
// The client flips the message from "sending" to "sent" only when it arrives.
type DeliveryAck struct {
	ReceiverId string    `json:"receiverId"`
	SentAt     time.Time `json:"sentAt"`
}

// AckedMessage identifies one queued message of the receiver: the sender and the
// client timestamp that key the queue entry.
type AckedMessage struct {
	SenderId string    `json:"senderId" validate:"required,uuid"`
	SentAt   time.Time `json:"sentAt" validate:"required"`
}

// WsMessageAck is the payload a client sends after storing the received
// messages locally. The server removes those entries from the offline queue.
type WsMessageAck struct {
	Messages []AckedMessage `json:"messages" validate:"required,min=1,max=100,dive"`
}
