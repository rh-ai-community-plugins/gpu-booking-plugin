package kube

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/eformat/gpu-booking-plugin/pkg/database"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := database.Init(filepath.Join(dir, "test.db")); err != nil {
		t.Fatalf("Init test DB: %v", err)
	}
	t.Cleanup(database.Close)
}

func insertBookingWithType(t *testing.T, bookingType, id, user, resource string, slotIndex int, date string, startHour, endHour int, utcOffset float64) {
	t.Helper()
	db := database.DB()
	_, err := db.Exec(
		"INSERT INTO bookings ("+database.BookingColumns+") VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		id, user, "", resource, slotIndex, date, "full", "now", database.SourceReserved, "", startHour, endHour, utcOffset, bookingType,
	)
	if err != nil {
		t.Fatalf("insertBooking %s: %v", id, err)
	}
}

func insertBooking(t *testing.T, id, user, resource string, slotIndex int, date string, startHour, endHour int, utcOffset float64) {
	t.Helper()
	insertBookingWithType(t, database.BookingTypeUser, id, user, resource, slotIndex, date, startHour, endHour, utcOffset)
}

func TestActiveReservations_UTC_FullDay(t *testing.T) {
	setupTestDB(t)

	// UTC user books a full day for "2026-05-04" with offset=0.
	// Active window: May 4 00:00 UTC to May 5 00:00 UTC.
	insertBooking(t, "b1", "alice", "nvidia.com/gpu", 0, "2026-05-04", 0, 24, 0)

	// At May 4 12:00 UTC — should be active.
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 reservation, got %d", len(res))
	}
	if res[0].User != "alice" {
		t.Errorf("user = %q, want alice", res[0].User)
	}
	if res[0].Resources["nvidia.com/gpu"] != 1 {
		t.Errorf("gpu count = %d, want 1", res[0].Resources["nvidia.com/gpu"])
	}
}

func TestActiveReservations_UTC_FullDay_Expired(t *testing.T) {
	setupTestDB(t)

	// UTC user books May 4 full day. At May 5 01:00 it's expired.
	insertBooking(t, "b1", "alice", "nvidia.com/gpu", 0, "2026-05-04", 0, 24, 0)

	now := time.Date(2026, 5, 5, 1, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected 0 reservations (expired), got %d", len(res))
	}
}

func TestActiveReservations_AEST_FullDay(t *testing.T) {
	setupTestDB(t)

	// AEST user (UTC+10) books "2026-05-04" full day (local hours 0-24).
	// UTC active window: May 3 14:00 UTC to May 4 14:00 UTC.
	insertBooking(t, "b1", "bob", "nvidia.com/gpu", 0, "2026-05-04", 0, 24, 10)

	// At May 3 22:00 UTC (= May 4 08:00 AEST) — should be active.
	now := time.Date(2026, 5, 3, 22, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("AEST booking should be active at May 3 22:00 UTC, got %d reservations", len(res))
	}
	if res[0].User != "bob" {
		t.Errorf("user = %q, want bob", res[0].User)
	}
}

func TestActiveReservations_AEST_NotYetActive(t *testing.T) {
	setupTestDB(t)

	// AEST user (UTC+10) books "2026-05-04" full day.
	// UTC start is May 3 14:00. At May 3 13:00 it shouldn't be active yet.
	insertBooking(t, "b1", "bob", "nvidia.com/gpu", 0, "2026-05-04", 0, 24, 10)

	now := time.Date(2026, 5, 3, 13, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("AEST booking should NOT be active at May 3 13:00 UTC, got %d", len(res))
	}
}

func TestActiveReservations_EST_FullDay(t *testing.T) {
	setupTestDB(t)

	// EST user (UTC-5) books "2026-05-04" full day (local hours 0-24).
	// UTC active window: May 4 05:00 UTC to May 5 05:00 UTC.
	insertBooking(t, "b1", "carol", "nvidia.com/gpu", 0, "2026-05-04", 0, 24, -5)

	// At May 4 10:00 UTC (= May 4 05:00 EST) — should be active.
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("EST booking should be active at May 4 10:00 UTC, got %d", len(res))
	}

	// At May 4 04:00 UTC (= May 3 23:00 EST) — should NOT be active yet.
	now = time.Date(2026, 5, 4, 4, 0, 0, 0, time.UTC)
	res, err = getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("EST booking should NOT be active at May 4 04:00 UTC, got %d", len(res))
	}
}

func TestActiveReservations_PartialDay_CrossMidnight(t *testing.T) {
	setupTestDB(t)

	// AEST user (UTC+10) books 9am-5pm local on May 4.
	// UTC: start = (9 - 10) = -1h from May 4 midnight UTC = May 3 23:00 UTC
	//      end   = (17 - 10) = 7h from May 4 midnight UTC = May 4 07:00 UTC
	insertBooking(t, "b1", "dave", "nvidia.com/gpu", 0, "2026-05-04", 9, 17, 10)

	// At May 4 03:00 UTC (= May 4 13:00 AEST) — should be active.
	now := time.Date(2026, 5, 4, 3, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("partial-day AEST booking should be active at May 4 03:00 UTC, got %d", len(res))
	}

	// At May 3 22:00 UTC (= before 9am AEST, which is 23:00 UTC) — NOT active.
	now = time.Date(2026, 5, 3, 22, 0, 0, 0, time.UTC)
	res, err = getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("partial-day should NOT be active at May 3 22:00 UTC, got %d", len(res))
	}

	// At May 4 07:30 UTC (= after 5pm AEST which is 07:00 UTC) — NOT active.
	now = time.Date(2026, 5, 4, 7, 30, 0, 0, time.UTC)
	res, err = getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("partial-day should NOT be active at May 4 07:30 UTC, got %d", len(res))
	}
}

func TestActiveReservations_MultipleUsers_DifferentTZ(t *testing.T) {
	setupTestDB(t)

	// AEST user full day May 4: UTC window May 3 14:00 – May 4 14:00
	insertBooking(t, "b1", "aest-user", "nvidia.com/gpu", 0, "2026-05-04", 0, 24, 10)
	// EST user full day May 4: UTC window May 4 05:00 – May 5 05:00
	insertBooking(t, "b2", "est-user", "nvidia.com/gpu", 1, "2026-05-04", 0, 24, -5)

	// At May 4 10:00 UTC — both should be active (AEST window: 14:00-14:00, EST: 05:00-05:00)
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 reservations (both active), got %d", len(res))
	}

	// At May 4 15:00 UTC — only EST should be active (AEST expired at 14:00)
	now = time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
	res, err = getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 reservation (only EST), got %d", len(res))
	}
	if res[0].User != "est-user" {
		t.Errorf("expected est-user, got %q", res[0].User)
	}
}

func TestActiveReservations_MultipleSlots_SameUser(t *testing.T) {
	setupTestDB(t)

	// Same user books 2 GPU slots for the same day.
	insertBooking(t, "b1", "alice", "nvidia.com/gpu", 0, "2026-05-04", 0, 24, 0)
	insertBooking(t, "b2", "alice", "nvidia.com/gpu", 1, "2026-05-04", 0, 24, 0)

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 reservation (one user), got %d", len(res))
	}
	if res[0].Resources["nvidia.com/gpu"] != 2 {
		t.Errorf("gpu count = %d, want 2", res[0].Resources["nvidia.com/gpu"])
	}
}

func TestActiveReservations_UntilTimestamp(t *testing.T) {
	setupTestDB(t)

	// AEST user full day May 4, offset +10.
	// UTC end = May 4 00:00 + (24-10)h = May 4 14:00 UTC
	insertBooking(t, "b1", "bob", "nvidia.com/gpu", 0, "2026-05-04", 0, 24, 10)

	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 reservation, got %d", len(res))
	}

	expectedUntil := time.Date(2026, 5, 4, 14, 0, 0, 0, time.UTC).Unix()
	if res[0].Until != expectedUntil {
		t.Errorf("until = %d, want %d (May 4 14:00 UTC)", res[0].Until, expectedUntil)
	}
}

func TestActiveReservations_ConsumedBookingsIgnored(t *testing.T) {
	setupTestDB(t)

	db := database.DB()
	// Insert a consumed booking (should be ignored by getActiveReservationsAt)
	_, err := db.Exec(
		"INSERT INTO bookings ("+database.BookingColumns+") VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		"c1", "kueue-user", "", "nvidia.com/gpu", 0, "2026-05-04", "full", "now", database.SourceConsumed, "", 0, 24, 0, database.BookingTypeUser,
	)
	if err != nil {
		t.Fatalf("insert consumed: %v", err)
	}

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("consumed bookings should be ignored, got %d reservations", len(res))
	}
}

func TestActiveReservations_Empty(t *testing.T) {
	setupTestDB(t)

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected 0 reservations, got %d", len(res))
	}
}

func TestActiveReservations_SystemBooking(t *testing.T) {
	setupTestDB(t)

	// Admin-created system booking for the demo-maas namespace.
	// Active window: May 4 00:00 UTC to May 5 00:00 UTC (utcOffset 0).
	insertBookingWithType(t, database.BookingTypeSystem, "b1", "demo-maas", "nvidia.com/gpu", 0, "2026-05-04", 0, 24, 0)

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 reservation, got %d", len(res))
	}
	if res[0].User != "demo-maas" {
		t.Errorf("user = %q, want demo-maas (target namespace)", res[0].User)
	}
	if !res[0].IsSystem {
		t.Error("IsSystem = false, want true")
	}
	if res[0].Resources["nvidia.com/gpu"] != 1 {
		t.Errorf("gpu count = %d, want 1", res[0].Resources["nvidia.com/gpu"])
	}
}

func TestActiveReservations_UserAndSystemSeparate(t *testing.T) {
	setupTestDB(t)

	// User booking and system booking must produce separate reservations,
	// never merged (grouping is by booking type + user).
	insertBooking(t, "b1", "alice", "nvidia.com/gpu", 0, "2026-05-04", 0, 24, 0)
	insertBookingWithType(t, database.BookingTypeSystem, "b2", "demo-maas", "nvidia.com/gpu", 1, "2026-05-04", 0, 24, 0)

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 reservations (user + system), got %d", len(res))
	}

	found := map[string]userReservation{}
	for _, r := range res {
		found[r.User] = r
	}
	alice, ok := found["alice"]
	if !ok || alice.IsSystem {
		t.Errorf("expected alice user reservation (IsSystem=false), got %+v", alice)
	}
	sys, ok := found["demo-maas"]
	if !ok || !sys.IsSystem {
		t.Errorf("expected demo-maas system reservation (IsSystem=true), got %+v", sys)
	}
}

func TestCQNaming(t *testing.T) {
	userRes := userReservation{User: "alice"}
	if got := userRes.cqName(); got != "user-alice" {
		t.Errorf("cqName() = %q, want user-alice", got)
	}
	if got := userRes.namespace(); got != "user-alice" {
		t.Errorf("namespace() = %q, want user-alice", got)
	}

	// Usernames with @domain sanitize to the short name
	emailRes := userReservation{User: "cluster-admin@redhat.com"}
	if got := emailRes.cqName(); got != "user-cluster-admin" {
		t.Errorf("cqName() = %q, want user-cluster-admin", got)
	}

	sysRes := userReservation{User: "demo-maas", IsSystem: true}
	if got := sysRes.cqName(); got != "system-demo-maas" {
		t.Errorf("cqName() = %q, want system-demo-maas", got)
	}
	if got := sysRes.namespace(); got != "demo-maas" {
		t.Errorf("namespace() = %q, want demo-maas (target namespace, not a user- ns)", got)
	}
}

func TestNamespaceForCQ(t *testing.T) {
	// User CQ: name equals namespace
	if got := namespaceForCQ("user-alice", nil); got != "user-alice" {
		t.Errorf("namespaceForCQ(user CQ) = %q, want user-alice", got)
	}

	// System CQ: namespace from the rhai-tmm.dev/namespace label
	if got := namespaceForCQ("system-demo-maas", map[string]string{"rhai-tmm.dev/namespace": "demo-maas"}); got != "demo-maas" {
		t.Errorf("namespaceForCQ(system CQ, label) = %q, want demo-maas", got)
	}

	// System CQ without label: falls back to prefix strip
	if got := namespaceForCQ("system-demo-maas", nil); got != "demo-maas" {
		t.Errorf("namespaceForCQ(system CQ, no label) = %q, want demo-maas", got)
	}
}

func TestSanitizeK8sName(t *testing.T) {
	cases := map[string]string{
		"cluster-admin@redhat.com": "cluster-admin",
		"alice":                    "alice",
		"User@EXAMPLE.com":         "user",
		"a_b":                      "a-b",
		"-weird-":                  "weird",
	}
	for in, want := range cases {
		if got := sanitizeK8sName(in); got != want {
			t.Errorf("sanitizeK8sName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestActiveReservations_IST_HalfHourOffset(t *testing.T) {
	setupTestDB(t)

	// IST user (UTC+5.5) books "2026-05-04" full day (local hours 0-24).
	// UTC active window: May 3 18:30 UTC to May 4 18:30 UTC.
	insertBooking(t, "b1", "priya", "nvidia.com/gpu", 0, "2026-05-04", 0, 24, 5.5)

	// At May 3 22:00 UTC (= May 4 03:30 IST) — should be active.
	now := time.Date(2026, 5, 3, 22, 0, 0, 0, time.UTC)
	res, err := getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("IST booking should be active at May 3 22:00 UTC, got %d reservations", len(res))
	}
	if res[0].User != "priya" {
		t.Errorf("user = %q, want priya", res[0].User)
	}

	// At May 3 18:00 UTC (30 minutes before UTC window opens) — should NOT be active.
	now = time.Date(2026, 5, 3, 18, 0, 0, 0, time.UTC)
	res, err = getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("IST booking should NOT be active at May 3 18:00 UTC (before 18:30 start), got %d", len(res))
	}

	// Verify Until timestamp = May 4 18:30 UTC.
	now = time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	res, err = getActiveReservationsAt(now)
	if err != nil {
		t.Fatalf("getActiveReservationsAt: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 reservation, got %d", len(res))
	}
	expectedUntil := time.Date(2026, 5, 4, 18, 30, 0, 0, time.UTC).Unix()
	if res[0].Until != expectedUntil {
		t.Errorf("until = %d (%s), want %d (May 4 18:30 UTC)",
			res[0].Until, time.Unix(res[0].Until, 0).UTC(), expectedUntil)
	}
}

func TestApplyUserReservation_SystemLQClusterQueue(t *testing.T) {
	var mu sync.Mutex
	lqBodies := map[string]map[string]any{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var manifest map[string]any
		json.Unmarshal(body, &manifest)

		if manifest["kind"] == "LocalQueue" {
			mu.Lock()
			lqBodies[r.URL.Path] = manifest
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"metadata":{}}`))
	}))
	defer ts.Close()

	origHost, origToken, origClient := k8sHost, k8sToken, k8sHTTPClient
	k8sHost = ts.URL
	k8sToken = "test-token"
	k8sHTTPClient = ts.Client()
	defer func() { k8sHost, k8sToken, k8sHTTPClient = origHost, origToken, origClient }()

	// System booking: namespace=demo-maas, CQ=system-demo-maas
	sysRes := userReservation{
		User:      "demo-maas",
		IsSystem:  true,
		Resources: map[string]int{"nvidia.com/gpu": 1},
		CPU:       2,
		Memory:    8,
		Until:     time.Now().Add(time.Hour).Unix(),
	}
	if err := applyUserReservation(sysRes); err != nil {
		t.Fatalf("applyUserReservation (system): %v", err)
	}

	lqPath := "/apis/kueue.x-k8s.io/v1beta1/namespaces/demo-maas/localqueues/reserved"
	lq, ok := lqBodies[lqPath]
	if !ok {
		t.Fatalf("no LocalQueue PATCH to %s", lqPath)
	}
	spec := lq["spec"].(map[string]any)
	if got := spec["clusterQueue"]; got != "system-demo-maas" {
		t.Errorf("system LQ clusterQueue = %q, want system-demo-maas", got)
	}

	// User booking: namespace=user-alice, CQ=user-alice (both match)
	lqBodies = map[string]map[string]any{}
	userRes := userReservation{
		User:      "alice",
		Resources: map[string]int{"nvidia.com/gpu": 1},
		CPU:       2,
		Memory:    8,
		Until:     time.Now().Add(time.Hour).Unix(),
	}
	if err := applyUserReservation(userRes); err != nil {
		t.Fatalf("applyUserReservation (user): %v", err)
	}

	lqPath = "/apis/kueue.x-k8s.io/v1beta1/namespaces/user-alice/localqueues/reserved"
	lq, ok = lqBodies[lqPath]
	if !ok {
		t.Fatalf("no LocalQueue PATCH to %s", lqPath)
	}
	spec = lq["spec"].(map[string]any)
	if got := spec["clusterQueue"]; got != "user-alice" {
		t.Errorf("user LQ clusterQueue = %q, want user-alice", got)
	}
}
