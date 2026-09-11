package service

import (
	"context"
	"fmt"

	"github.com/PithomLabs/conductor/internal/governance"
)

// GovernanceService provides governance projection reads.
type GovernanceService struct {
	readers map[string]governance.GovernanceReader // provider → reader
}

// NewGovernanceService creates a new governance service.
func NewGovernanceService() *GovernanceService {
	return &GovernanceService{
		readers: make(map[string]governance.GovernanceReader),
	}
}

// RegisterProvider registers a governance reader for a provider.
func (s *GovernanceService) RegisterProvider(provider string, reader governance.GovernanceReader) {
	s.readers[provider] = reader
}

// GetState returns governance state for a reference.
func (s *GovernanceService) GetState(ctx context.Context, ref governance.GovernanceReference) (*governance.GovernanceState, error) {
	reader, ok := s.readers[ref.Provider]
	if !ok {
		return &governance.GovernanceState{
			Reference: ref,
			Status:    governance.GovernanceStatusUnknown,
			Blockers:  []string{fmt.Sprintf("unknown governance provider: %s", ref.Provider)},
		}, nil
	}
	return reader.GetState(ctx, ref)
}

