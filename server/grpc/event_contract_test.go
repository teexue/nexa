package grpcapi

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/teexue/nexakit/event"
	commonagentv1 "github.com/teexue/common-agent/proto"
)

func TestEventToProtoCoversAllTypes(t *testing.T) {
	for _, typ := range event.AllTypes() {
		pb := EventToProto(event.Event{Type: typ})
		require.NotEqual(t, commonagentv1.EventType_EVENT_TYPE_UNSPECIFIED, pb.Type, string(typ))
		require.Equal(t, typ, protoToEventType(pb.Type), string(typ))
	}
}
