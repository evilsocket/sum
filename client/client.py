import os
import sys
import zlib
import grpc
import json

path = os.path.join(os.path.abspath(os.path.dirname(__file__)), "../proto")
sys.path.append(path)

import sum_pb2
import sum_pb2_grpc 

class Client:
    MAX_MESSAGE_SIZE = 10 * 1024 * 1024

    def __init__(self, connection_string):
        self._conn_str = connection_string
        self._opts =[ 
                ('grpc.max_send_message_length', Client.MAX_MESSAGE_SIZE), 
                ('grpc.max_receive_message_length', Client.MAX_MESSAGE_SIZE)]
        self._rpc = sum_pb2_grpc.SumServiceStub(grpc.insecure_channel(self._conn_str, options=self._opts))
    
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
            nv = sum_pb2.NamedValue(name=k, value=v)
            pbmeta.append(nv)

        record = sum_pb2.Record(data=data, meta=pbmeta)
        resp = self._rpc.CreateRecord(record)
        self._check_resp(resp)
        record.id = resp.msg
        return record 

    def delete_record(self, identifier):
        resp = self._rpc.DeleteRecord(sum_pb2.ById(id=identifier))
        self._check_resp(resp)

    def define_oracle(self, filename, name):
        resp = self._rpc.FindOracle(sum_pb2.ByName(name=name))
        self._check_resp(resp)

        if len(resp.oracles) == 0:
            with open( filename, 'r') as fp:
                oracle = sum_pb2.Oracle(name=name, code=fp.read())
                resp = self._rpc.CreateOracle(oracle)
                self._check_resp(resp)
                return resp.msg
        else:
            return resp.oracles[0].id

    def invoke_oracle(self, oracle_id, args):
        resp = self._rpc.Run(sum_pb2.Call(oracle_id=oracle_id, args=map(json.dumps, args)))
        self._check_resp(resp)
        return self._get_oracle_payload(resp.data)

