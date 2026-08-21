package calibration

import (
	"fmt"
	"sync"
	"time"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

type Service struct {
	store         *store.Store
	now           func() time.Time
	sequence      uint64
	sequenceMu    sync.Mutex
	cacheMu       sync.Mutex
	listCache     map[string]sessionListCacheEntry
	progressCache map[string]progressCacheEntry
}

type sessionListCacheEntry struct {
	revision int64
	sessions []domain.CalibrationSession
}

type progressCacheEntry struct {
	progress domain.SessionProgress
}

func NewService(repository *store.Store) *Service {
	return &Service{store: repository, now: time.Now, listCache: make(map[string]sessionListCacheEntry), progressCache: make(map[string]progressCacheEntry)}
}

func (s *Service) StorePath() string {
	return s.store.Path()
}

func (s *Service) newID(prefix string) string {
	s.sequenceMu.Lock()
	defer s.sequenceMu.Unlock()
	value := s.sequence + 1
	s.sequence = value
	return fmt.Sprintf("%s-%d-%d", prefix, s.now().UnixNano(), value)
}
