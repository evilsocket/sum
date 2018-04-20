import os
import sys
import zlib
import grpc
import json

import proto.sum_pb2 as proto
import proto.sum_pb2_grpc as proto_svc

class Client:
    MAX_MESSAGE_SIZE = 10 * 1024 * 1024

    def __init__(self, connection_string):
        self._conn_str = connection_string
        self._opts =[ 
                ('grpc.max_send_message_length', Client.MAX_MESSAGE_SIZE), 
                ('grpc.max_receive_message_length', Client.MAX_MESSAGE_SIZE)]
        self._rpc = proto_svc.SumServiceStub(grpc.insecure_channel(self._conn_str, options=self._opts))
    
    def _check_resp(self, r):
        if r.success == False:
            raise Exception(r.msg)

    def _get_oracle_payload(self, data):
        raw = data.payload
        if data.compressed:
            raw = zlib.decompress(raw, 16+zlib.MAX_WBITS)
        return json.loads(str(raw))

    def create_record(self, meta, data):
        pbmeta = []
        for k, v in meta.iteritems():
            nv = proto.NamedValue(name=k, value=v)
            pbmeta.append(nv)

        record = proto.Record(data=data, meta=pbmeta)
        resp = self._rpc.CreateRecord(record)
        self._check_resp(resp)
        record.id = int(resp.msg)
        return record 
    
    def read_record(self, identifier):
        resp = self._rpc.ReadRecord(proto.ById(id=int(identifier)))
        self._check_resp(resp)
        return resp.record

    def delete_record(self, identifier):
        resp = self._rpc.DeleteRecord(proto.ById(id=identifier))
        self._check_resp(resp)

    def define_oracle(self, filename, name):
        resp = self._rpc.FindOracle(proto.ByName(name=name))
        self._check_resp(resp)

        if len(resp.oracles) == 0:
            with open( filename, 'r') as fp:
                oracle = proto.Oracle(name=name, code=fp.read())
                resp = self._rpc.CreateOracle(oracle)
                self._check_resp(resp)
                return int(resp.msg)
        else:
            return resp.oracles[0].id

    def invoke_oracle(self, oracle_id, args):
        resp = self._rpc.Run(proto.Call(oracle_id=oracle_id, args=map(json.dumps, args)))
        self._check_resp(resp)
        return self._get_oracle_payload(resp.data)

