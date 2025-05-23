package protocol_test

import (
	"strings"
	"testing"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/net"
	. "github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/uuid"
)

func TestAlwaysValidStrategy(t *testing.T) {
	strategy := AlwaysValid()
	if !strategy.IsValid() {
		t.Error("strategy not valid")
	}
	strategy.Invalidate()
	if !strategy.IsValid() {
		t.Error("strategy not valid")
	}
}

func TestTimeoutValidStrategy(t *testing.T) {
	strategy := BeforeTime(time.Now().Add(2 * time.Second))
	if !strategy.IsValid() {
		t.Error("strategy not valid")
	}
	time.Sleep(3 * time.Second)
	if strategy.IsValid() {
		t.Error("strategy is valid")
	}

	strategy = BeforeTime(time.Now().Add(2 * time.Second))
	strategy.Invalidate()
	if strategy.IsValid() {
		t.Error("strategy is valid")
	}
}

func TestPickUser(t *testing.T) {
	spec := NewServerSpec(net.Destination{}, AlwaysValid(), &MemoryUser{Email: "test1@example.com"}, &MemoryUser{Email: "test2@example.com"}, &MemoryUser{Email: "test3@example.com"})
	user := spec.PickUser()
	if !strings.HasSuffix(user.Email, "@example.com") {
		t.Error("user: ", user.Email)
	}
}
