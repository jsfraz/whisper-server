package utils

// https://dev.to/kittipat1413/understanding-the-singleton-pattern-in-go-5h99

import "gorm.io/gorm"

var (
	instance *Singleton
)

type Singleton struct {
	MailTemlplate string
	VerifyMail    MailData
	VerifiedMail  MailData
	PostgresDb    gorm.DB
}

// Gets Singleton instance
//
//	@return *Singleton
func GetSingleton() *Singleton {
	if instance == nil {
		instance = new(Singleton)
	}
	return instance
}
