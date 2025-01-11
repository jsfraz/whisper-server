package models

type Message struct {
	Topic   string      `json:"topic"`
	Payload interface{} `json:"payload"`
}
