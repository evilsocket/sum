package orchestrator

import (
	"fmt"
	. "github.com/evilsocket/sum/proto"
)

// builds a record response that contains an error
func errRecordResponse(format string, args ...interface{}) *RecordResponse {
	return &RecordResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

// builds a find response that contains an error
func errFindResponse(format string, args ...interface{}) *FindResponse {
	return &FindResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

// builds an oracle response that contains an error
func errOracleResponse(format string, args ...interface{}) *OracleResponse {
	return &OracleResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

// builds a call response that contains an error
func errCallResponse(format string, args ...interface{}) *CallResponse {
	return &CallResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

// builds a node response that contains an error
func errNodeResponse(format string, args ...interface{}) *NodeResponse {
	return &NodeResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}
