package listcachealias

import (
	"path/filepath"
	"testing"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func TestListSessionsDoesNotExposeCachedBackingSlice(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := calibration.NewService(repository)
	created, _, err := service.CreateSession(calibration.CreateSessionRequest{
		DeviceID: "CACHE-1", DeviceName: "缓存测试仪", ObservingBand: "可见光", Owner: "负责人",
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.ListSessionsByFilters(calibration.SessionListFilters{})
	if err != nil || len(first) != 1 || first[0].ID != created.ID {
		t.Fatalf("initial list failed: %#v, %v", first, err)
	}
	first[0].DeviceName = "外部篡改"
	second, err := service.ListSessionsByFilters(calibration.SessionListFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if second[0].DeviceName != created.DeviceName {
		t.Fatalf("cached list exposed mutable backing data: got %q", second[0].DeviceName)
	}
}
