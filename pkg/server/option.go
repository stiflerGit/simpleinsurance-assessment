package server

import (
	"log"
)

type Option func(s *Server)

// WithFilePersistencePath set the path of where the server save its persistence file
func WithFilePersistencePath(path string) Option {
	return func(s *Server) {
		s.filePath = path
	}
}

// WithPerIPRequestLimiter if limit is lte 0 than no limit is applied
func WithPerIPRequestLimiter(limit int64) Option {
	return func(s *Server) {
		s.limit = limit
	}
}

func WithLogger(logger *log.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}
