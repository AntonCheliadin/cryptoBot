package indicator

import (
	"cryptoBot/pkg/service/date"
)

var sessionsServiceImpl *SessionsService

func NewSessionsService(clock date.Clock) *SessionsService {
	if sessionsServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	sessionsServiceImpl = &SessionsService{
		Clock: clock,
	}
	return sessionsServiceImpl
}

type SessionsService struct {
	Clock date.Clock
}

const LONDON_SESSION_START = 8
const LONDON_SESSION_END = 17

const NEW_YORK_SESSION_START = 13
const NEW_YORK_SESSION_END = 22

func (s *SessionsService) IsSuitableSessionNow() bool {
	currentHour := s.Clock.NowTime().Hour()

	isLondonSession := currentHour >= LONDON_SESSION_START && currentHour < LONDON_SESSION_END
	isNewYorkSession := currentHour >= NEW_YORK_SESSION_START && currentHour < NEW_YORK_SESSION_END

	return isLondonSession || isNewYorkSession
}
