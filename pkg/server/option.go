package server

type Option func(s *Server)

func WithFilePersistence(path string) Option {
	return func(s *Server) {
		s.filePath = path
	}
}
