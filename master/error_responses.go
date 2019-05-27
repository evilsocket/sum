package master

import (
	"fmt"
	. "github.com/evilsocket/sum/proto"
	"github.com/golang/protobuf/proto"
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

// Get the error message from either a GRPC error or a application-level one
func getErrorMessage(err error, response proto.Message) string {
	if err != nil {
		return err.Error()
	}

	var success bool
	var msg string

	switch response.(type) {
	case *RecordResponse:
		success = response.(*RecordResponse).Success
		msg = response.(*RecordResponse).Msg
	case *OracleResponse:
		success = response.(*OracleResponse).Success
		msg = response.(*OracleResponse).Msg
	case *CallResponse:
		success = response.(*CallResponse).Success
		msg = response.(*CallResponse).Msg
	case *FindResponse:
		success = response.(*FindResponse).Success
		msg = response.(*FindResponse).Msg
	default:
		panic(fmt.Sprintf("unsupported message %T: %v", response, response))
	}

	if !success {
		return msg
	}

	panic("no errors dude")
}
