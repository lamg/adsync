package tesis

import (
	"github.com/dgrijalva/jwt-go"
	"time"
)

//cuantos correos has mandado y recibido
//cuantos megas has consumido de tu cuenta
//de correo y de internet

type Info struct {
	SentMessages int `json: "sentMessages"`
	RecvMessages int
	MailStorage  int
	InternetDwnl int
	WifiLogons   []WifiL
}

type WifiL struct {
	Ip    string
	Place string
	Date  time.Time
}

type Credentials struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

type User struct {
	UserName string `json:"user"`
	jwt.StandardClaims
}

type Authenticator interface {
	Authenticate(user, password string) bool
}

type DBManager interface {
	UserInfo(string) (*Info, error)
}
