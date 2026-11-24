package gateway

import (
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
)

func admissionHarness(t *testing.T, current func() Admission) (*harness, *Gateway) {
	t.Helper()
	h := &harness{t: t, clock: &fakeClock{now: time.UnixMilli(1761480000000)}, config: testConfig(), sessions: map[string]*recordingSession{}}
	g := New(Options{Config: h.config, Handler: HandlerFunc(h.connect), Clock: h.clock, Admission: current})
	h.server = httptest.NewServer(g)
	t.Cleanup(h.server.Close)
	return h, g
}

func helloWithBuild(version, build string) *pb.Envelope {
	frame := hello("ok", version, 1)
	frame.GetClientHello().BuildHash = build
	return frame
}

func TestUnlistedBuildGetsUpdateRequired(t *testing.T) {
	admission := Admission{MinimumAppVersion: "1.0.0", EnforceBuilds: true, Builds: []string{"listed"}}
	h, _ := admissionHarness(t, func() Admission { return admission })
	leaked := h.dial()
	leaked.send(helloWithBuild("1.2.3", "leaked"))
	if status := leaked.recv().GetServerHello().GetStatus(); status != pb.HelloStatus_HELLO_STATUS_UPDATE_REQUIRED {
		t.Fatalf("unlisted build answered %v", status)
	}
	leaked.expectClose(websocket.StatusNormalClosure)
	if h.session("ok") != nil {
		t.Fatal("an unlisted build reached the handler")
	}
	listed := h.dial()
	listed.send(helloWithBuild("1.2.3", "listed"))
	if status := listed.recv().GetServerHello().GetStatus(); status != pb.HelloStatus_HELLO_STATUS_ACCEPTED {
		t.Fatalf("listed build answered %v", status)
	}
}

func TestEnforceCutsOffAConnectedBuildAfterTheControlsChange(t *testing.T) {
	var admission atomic.Pointer[Admission]
	admission.Store(&Admission{MinimumAppVersion: "1.0.0"})
	h, g := admissionHarness(t, func() Admission { return *admission.Load() })
	old, current := h.dial(), h.dial()
	old.handshake("ok")
	current.send(helloWithBuild("1.5.0", "current"))
	if status := current.recv().GetServerHello().GetStatus(); status != pb.HelloStatus_HELLO_STATUS_ACCEPTED {
		t.Fatalf("current build answered %v", status)
	}
	g.Enforce()
	admission.Store(&Admission{MinimumAppVersion: "1.4.0"})
	g.Enforce()
	frame := old.recv().GetServerHello()
	if frame.GetStatus() != pb.HelloStatus_HELLO_STATUS_UPDATE_REQUIRED || frame.GetMinimumAppVersion() != "1.4.0" {
		t.Fatalf("connected old build got %v", frame)
	}
	old.expectClose(websocket.StatusNormalClosure)
	deadline := time.Now().Add(5 * time.Second)
	for g.Connections() != 1 {
		if time.Now().After(deadline) {
			t.Fatalf("connections %d, want the current build only", g.Connections())
		}
		time.Sleep(time.Millisecond)
	}
}

func TestMaintenanceTurnsAwayEveryoneOutsideAMatch(t *testing.T) {
	var admission atomic.Pointer[Admission]
	admission.Store(&Admission{MinimumAppVersion: "1.0.0"})
	h, g := admissionHarness(t, func() Admission { return *admission.Load() })
	idle, playing := h.dial(), h.dial()
	idle.handshake("ok")
	playing.handshake("ok-playing")
	ends := time.UnixMilli(1761490000000).UTC()
	admission.Store(&Admission{MinimumAppVersion: "1.0.0", MaintenanceEndsAt: ends})
	g.Enforce()
	frame := idle.recv().GetServerHello()
	if frame.GetStatus() != pb.HelloStatus_HELLO_STATUS_MAINTENANCE || frame.GetMaintenanceEndsUnixMs() != 1761490000000 {
		t.Fatalf("connected idle player got %v", frame)
	}
	idle.expectClose(websocket.StatusNormalClosure)

	late := h.dial()
	late.send(hello("ok-late", "1.2.3", 1))
	if status := late.recv().GetServerHello().GetStatus(); status != pb.HelloStatus_HELLO_STATUS_MAINTENANCE {
		t.Fatalf("a new hello answered %v", status)
	}
	late.expectClose(websocket.StatusNormalClosure)
	select {
	case <-h.session("ok-late").closed:
	case <-time.After(5 * time.Second):
		t.Fatal("the refused session stayed open")
	}
	h.dial().handshake("ok-playing")
	deadline := time.Now().Add(5 * time.Second)
	for g.Connections() != 2 {
		if time.Now().After(deadline) {
			t.Fatalf("connections %d, want the two in a match", g.Connections())
		}
		time.Sleep(time.Millisecond)
	}
}
