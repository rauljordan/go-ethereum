// Package p2p handles peer-to-peer networking for the sharding package.
package p2p

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

// Server is a placeholder for a shardp2p service. To be designed.
type Server struct {
	transactionsFeed *event.Feed
}

// NewServer creates a new shardp2p service instance.
func NewServer() (*Server, error) {
	return &Server{transactionsFeed: new(event.Feed)}, nil
}

// Start the main routine for an shardp2p server.
func (s *Server) Start() error {
	log.Info("Starting shardp2p server")
	go s.generateTestTransactions()
	return nil
}

// Stop the main shardp2p loop..
func (s *Server) Stop() error {
	log.Info("Stopping shardp2p server")
	return nil
}

func (s *Server) TransactionsFeed() *event.Feed {
	return s.transactionsFeed
}

func (s *Server) generateTestTransactions() {
	for {
		nsent := s.transactionsFeed.Send(1)
		log.Info(fmt.Sprintf("Sent transaction to %d subscribers", nsent))
		time.Sleep(time.Second)
	}
}
