package grpcapi

import (
	"encoding/json"

	"github.com/teexue/nexakit/event"
	"github.com/teexue/nexakit/provider"
	nexav1 "github.com/teexue/nexa/proto"
)

// EventToProto converts a core event.Event to a proto AgentEvent.
func EventToProto(ev event.Event) *nexav1.AgentEvent {
	pb := &nexav1.AgentEvent{
		Type:       eventTypeToProto(ev.Type),
		Content:    ev.Content,
		Tool:       ev.Tool,
		ToolCallId: ev.ToolCallID,
		ApprovalId: ev.ApprovalID,
		Code:       ev.Code,
		Message:    ev.Message,
		Status:     ev.Status,
		Turns:      int32(ev.Turns),
	}
	if ev.Input != nil {
		pb.Input = ev.Input
	}
	if ev.Output != nil {
		pb.Output = ev.Output
	}
	return pb
}

// ProtoToEvent converts a proto AgentEvent to a core event.Event.
func ProtoToEvent(pb *nexav1.AgentEvent) event.Event {
	return event.Event{
		Type:       protoToEventType(pb.Type),
		Content:    pb.Content,
		Tool:       pb.Tool,
		Input:      json.RawMessage(pb.Input),
		ToolCallID: pb.ToolCallId,
		ApprovalID: pb.ApprovalId,
		Output:     json.RawMessage(pb.Output),
		Code:       pb.Code,
		Message:    pb.Message,
		Status:     pb.Status,
		Turns:      int(pb.Turns),
	}
}

// ProtoMessagesToProvider converts proto Messages to provider.Messages.
func ProtoMessagesToProvider(msgs []*nexav1.Message) []provider.Message {
	out := make([]provider.Message, len(msgs))
	for i, m := range msgs {
		out[i] = provider.Message{
			Role:    provider.Role(m.Role),
			Content: m.Content,
		}
	}
	return out
}

func eventTypeToProto(t event.Type) nexav1.EventType {
	switch t {
	case event.TypeTextDelta:
		return nexav1.EventType_EVENT_TYPE_TEXT_DELTA
	case event.TypeReasoningDelta:
		return nexav1.EventType_EVENT_TYPE_REASONING_DELTA
	case event.TypeToolStart:
		return nexav1.EventType_EVENT_TYPE_TOOL_START
	case event.TypeToolResult:
		return nexav1.EventType_EVENT_TYPE_TOOL_RESULT
	case event.TypeToolApproval:
		return nexav1.EventType_EVENT_TYPE_TOOL_APPROVAL
	case event.TypeCompaction:
		return nexav1.EventType_EVENT_TYPE_COMPACTION
	case event.TypeSubAgentStart:
		return nexav1.EventType_EVENT_TYPE_SUB_AGENT_START
	case event.TypeSubAgentEnd:
		return nexav1.EventType_EVENT_TYPE_SUB_AGENT_END
	case event.TypeError:
		return nexav1.EventType_EVENT_TYPE_ERROR
	case event.TypeDone:
		return nexav1.EventType_EVENT_TYPE_DONE
	default:
		return nexav1.EventType_EVENT_TYPE_UNSPECIFIED
	}
}

func protoToEventType(t nexav1.EventType) event.Type {
	switch t {
	case nexav1.EventType_EVENT_TYPE_TEXT_DELTA:
		return event.TypeTextDelta
	case nexav1.EventType_EVENT_TYPE_REASONING_DELTA:
		return event.TypeReasoningDelta
	case nexav1.EventType_EVENT_TYPE_TOOL_START:
		return event.TypeToolStart
	case nexav1.EventType_EVENT_TYPE_TOOL_RESULT:
		return event.TypeToolResult
	case nexav1.EventType_EVENT_TYPE_TOOL_APPROVAL:
		return event.TypeToolApproval
	case nexav1.EventType_EVENT_TYPE_COMPACTION:
		return event.TypeCompaction
	case nexav1.EventType_EVENT_TYPE_SUB_AGENT_START:
		return event.TypeSubAgentStart
	case nexav1.EventType_EVENT_TYPE_SUB_AGENT_END:
		return event.TypeSubAgentEnd
	case nexav1.EventType_EVENT_TYPE_ERROR:
		return event.TypeError
	case nexav1.EventType_EVENT_TYPE_DONE:
		return event.TypeDone
	default:
		return ""
	}
}
