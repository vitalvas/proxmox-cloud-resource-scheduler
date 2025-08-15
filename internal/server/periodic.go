package server

import (
	"fmt"
)

const periodicTime = 30

func (s *Server) runPeriodic() error {
	if err := s.SetupCRS(); err != nil {
		return fmt.Errorf("setup CRS: %w", err)
	}

	return nil
}
