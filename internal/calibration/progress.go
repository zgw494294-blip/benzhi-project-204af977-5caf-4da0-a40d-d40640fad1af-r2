package calibration

import (
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
)

func (s *Service) GetProgress(id string) (domain.SessionProgress, error) {
	ledger := s.store.Snapshot()
	if _, ok := ledger.Sessions[id]; !ok {
		return domain.SessionProgress{}, notFound("校准会话")
	}
	s.cacheMu.Lock()
	if cached, ok := s.progressCache[id]; ok {
		s.cacheMu.Unlock()
		return cached.progress, nil
	}
	s.cacheMu.Unlock()
	progress := domain.BuildProgress(ledger.Samples[id], ledger.Measurements[id], ledger.Reviews[id])
	s.cacheMu.Lock()
	s.progressCache[id] = progressCacheEntry{progress: progress}
	s.cacheMu.Unlock()
	return progress, nil
}
