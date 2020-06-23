package server

import (
	"errors"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/stellar/go/xdr"
)

type eventData struct {
	e xdr.TransactionEnvelope
	r xdr.TransactionResult
	m xdr.TransactionMeta
}

type eventStream struct {
	sync.Mutex

	log *logrus.Entry

	closed   bool
	streamCh chan eventData
}

func newEventStream(bufferSize int) *eventStream {
	return &eventStream{
		log:      logrus.StandardLogger().WithField("type", "account/server/stream"),
		streamCh: make(chan eventData, bufferSize),
	}
}

func (s *eventStream) notify(e xdr.TransactionEnvelope, r xdr.TransactionResult, m xdr.TransactionMeta) error {
	s.Lock()

	if s.closed {
		s.Unlock()
		return errors.New("cannot notify closed stream")
	}

	select {
	case s.streamCh <- eventData{e: e, r: r, m: m}:
	default:
		s.Unlock()
		s.close()
		return errors.New("account event stream channel full")
	}

	s.Unlock()
	return nil
}

func (s *eventStream) close() {
	s.Lock()
	defer s.Unlock()

	if s.closed {
		return
	}

	s.closed = true
	close(s.streamCh)
}
