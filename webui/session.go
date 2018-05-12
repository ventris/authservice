package webui

import (
	"fmt"
	"log"

	"github.com/dhtech/authservice/verify"
	pb "github.com/dhtech/proto/auth"
	"github.com/google/uuid"
)

type loginSession struct {
	p *webuiServer
	// Structured attempt queue
	atq chan verify.Attempt
	// Redirect queue
	rq chan string
	// Attempt error queue
	eq chan error

	id string
	Ident string
	NextUrl string
}

type attempt struct {
	username string
	credential string
}

func (a *attempt) Username() string {
	return a.username
}

func (a *attempt) Credential() string {
	return a.credential
}

func (s *webuiServer) NewSession(r *pb.UserCredentialRequest, atq chan verify.Attempt, eq chan error) verify.Session {
	id := uuid.New().String()
	rq := make(chan string, 1)
	sess := &loginSession{
		Ident: r.ClientValidation.Ident,
		NextUrl: fmt.Sprintf("/next?session=%s", id),
		id: id,
		eq: eq,
		atq: atq,
		rq: rq,
		p: s,
	}
	s.sessionLock.Lock()
	s.sessions[id] = sess
	s.sessionLock.Unlock()
	return sess
}

func (s *loginSession) ChallengeLogin() *pb.UserAction {
	return &pb.UserAction{Url: fmt.Sprintf("/login?session=%s", s.id)}
}

func (s *loginSession) ChallengeReview() *pb.UserAction {
	s.rq <- fmt.Sprintf("/review?session=%s", s.id)
	return nil
}

func (s *loginSession) ChallengeComplete() *pb.UserAction {
	s.rq <- "/complete"
	return nil
}

func (s *loginSession) ProcessLogin(username string, password string) error {
	select {
	case s.atq <- &attempt{username, password}:
		return <-s.eq
	default:
		return fmt.Errorf("Session is gone")
	}
}

func (s *loginSession) NextStep() string {
	return <-s.rq
}

func (s *loginSession) Destroy() {
	s.p.sessionLock.Lock()
	delete(s.p.sessions, s.id)
	s.p.sessionLock.Unlock()
	close(s.rq)
	log.Printf("Cleaned up session %s", s.id)
}

