package Server

import "time"

func Daemon(s *Server) {
	// TODO implement

	timer := time.NewTicker(30 * time.Second)

	for {
		select {
		case <-s.ctx:
			return
		case <-s.operations:
			// TODO: Handle operations from queue
			break
		case <-timer.C:
			// TODO: Periodic operations
		}
	}
}
