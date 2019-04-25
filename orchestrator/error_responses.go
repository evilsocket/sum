package main

import (
	"fmt"
	. "github.com/evilsocket/sum/proto"
)

func errRecordResponse(format string, args ...interface{}) *RecordResponse {
	return &RecordResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func errFindResponse(format string, args ...interface{}) *FindResponse {
	return &FindResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func errOracleResponse(format string, args ...interface{}) *OracleResponse {
	return &OracleResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func errCallResponse(format string, args ...interface{}) *CallResponse {
	return &CallResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}
