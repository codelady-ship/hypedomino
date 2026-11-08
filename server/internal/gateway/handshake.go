package gateway

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"

	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/secrand"
)

func VersionAtLeast(version, minimum string) bool {
	have, ok := parseVersion(version)
	if !ok {
		return false
	}
	want, ok := parseVersion(minimum)
	if !ok {
		return true
	}
	for index := range have {
		if have[index] != want[index] {
			return have[index] > want[index]
		}
	}
	return true
}

func parseVersion(text string) ([3]int, bool) {
	var parts [3]int
	fields := strings.SplitN(strings.TrimPrefix(text, "v"), ".", 4)
	if len(fields) > 3 {
		return parts, false
	}
	for index, field := range fields {
		value, err := strconv.Atoi(field)
		if err != nil || value < 0 {
			return parts, false
		}
		parts[index] = value
	}
	return parts, true
}

type Admission struct {
	MinimumAppVersion string
	EnforceBuilds     bool
	Builds            []string
	MaintenanceEndsAt time.Time
}

func (a Admission) Maintenance() bool {
	return !a.MaintenanceEndsAt.IsZero()
}

func (a Admission) Admits(version, build string) bool {
	if !VersionAtLeast(version, a.MinimumAppVersion) {
		return false
	}
	return !a.EnforceBuilds || slices.Contains(a.Builds, build)
}

type clientBuild struct {
	version string
	hash    string
}

func (c *Conn) admission() Admission {
	if c.options.Admission == nil {
		return Admission{MinimumAppVersion: c.options.Config.MinimumAppVersion}
	}
	return c.options.Admission()
}

func (c *Conn) enforce() {
	build := c.build.Load()
	if build == nil {
		return
	}
	if c.admission().Maintenance() && !c.session.InMatch() {
		c.refuseForMaintenance()
		return
	}
	if c.admission().Admits(build.version, build.hash) {
		return
	}
	c.Send(c.serverHello(pb.HelloStatus_HELLO_STATUS_UPDATE_REQUIRED, 0, false))
	c.Close(websocket.StatusNormalClosure, "update required")
}

func (c *Conn) serverHello(status pb.HelloStatus, acked uint64, inMatch bool) *pb.Envelope {
	hello := &pb.ServerHello{
		Status:            status,
		MinimumAppVersion: c.admission().MinimumAppVersion,
		ServerTimeUnixMs:  uint64(c.options.Clock.Now().UnixMilli()),
		CatalogVersion:    c.options.Config.CatalogVersion,
		ContentBaseUrl:    c.options.Config.ContentBaseURL,
		InMatch:           inMatch,
	}
	if status == pb.HelloStatus_HELLO_STATUS_MAINTENANCE {
		hello.MaintenanceEndsUnixMs = uint64(c.admission().MaintenanceEndsAt.UnixMilli())
	}
	if status == pb.HelloStatus_HELLO_STATUS_ACCEPTED {
		hello.SessionNonce = c.nonce
	}
	if status == pb.HelloStatus_HELLO_STATUS_ACCEPTED && c.options.Strings != nil {
		localized := c.options.Strings()
		hello.StringBundles, hello.FallbackChains = localized.Bundles, localized.Chains
	}
	return &pb.Envelope{
		ProtocolVersion: ProtocolVersion,
		AckedSequence:   acked,
		Payload:         &pb.Envelope_ServerHello{ServerHello: hello},
	}
}

type ReplayCounter interface {
	Replayed()
}

type Queuer interface {
	JoinQueue(intent *pb.JoinQueueIntent)
	LeaveQueue()
}

func QueueStateFrame(state *pb.QueueState, acked uint64) *pb.Envelope {
	return &pb.Envelope{ProtocolVersion: ProtocolVersion, AckedSequence: acked, Payload: &pb.Envelope_QueueState{QueueState: state}}
}

func sessionNonce() uint64 {
	for {
		if nonce := (secrand.Source{}).Uint64(); nonce != 0 {
			return nonce
		}
	}
}

func ErrorFrame(code pb.ErrorCode, rejected, acked uint64) *pb.Envelope {
	return &pb.Envelope{
		ProtocolVersion: ProtocolVersion,
		AckedSequence:   acked,
		Payload:         &pb.Envelope_Error{Error: &pb.Error{Code: code, RejectedSequence: rejected}},
	}
}

func (c *Conn) refuseForMaintenance() {
	c.Send(c.serverHello(pb.HelloStatus_HELLO_STATUS_MAINTENANCE, 0, false))
	c.Close(websocket.StatusNormalClosure, "maintenance")
}
