#!/usr/bin/python
import sys
import os
import random
import datetime
import zlib
import json
import grpc

path = os.path.join(os.path.abspath(os.path.dirname(__file__)), "proto")
sys.path.append(path)

import sum_pb2
import sum_pb2_grpc 

start = 0
end = 0
num_columns = 100
num_rows = 300
index = {}
client = None
oracle_file = 'oracles/findsimilar.js'
oracle_name = 'findSimilar'
oracle_threshold = '0.81'
oracle_id = None

def gen_record(columns):
    return sum_pb2.Record( \
        data=[random.uniform(0,100) for i in range(0, columns)],
        meta=[sum_pb2.NamedValue( \
            name="example_metadata",
            value="Random number is %f" % random.random()
        )]
    )

def check(resp):
    if resp.success == False:
        print "ERROR: %s" % resp.msg
        quit()

def timer_start():
    global start
    sys.stdout.flush()
    start = datetime.datetime.now()

def timer_stop(with_avg=True):
    global start, end, index, client
    end = datetime.datetime.now()
    diff = end - start
    elapsed_ms = (diff.days * 86400000) + (diff.seconds * 1000) + (diff.microseconds / 1000)
    if with_avg:
        print "%d ms / %.2fms avg" % ( elapsed_ms, float(elapsed_ms) / float(len(index)) )
    else:
        print "%d ms" % elapsed_ms

def create_client(connection):
    max_len = 10 * 1024 * 1024
    opts=[('grpc.max_send_message_length', max_len), ('grpc.max_receive_message_length', max_len)]
    return sum_pb2_grpc.SumServiceStub(grpc.insecure_channel(connection,options=opts))

def define_oracle(filename, name):
    global client

    resp = client.FindOracle(sum_pb2.ByName(name=name))
    check(resp)

    if len(resp.oracles) == 0:
        print "Defining oracle %s ..." % name
        with open( filename, 'r') as fp:
            oracle = sum_pb2.Oracle(name=name, code=fp.read())
            resp = client.CreateOracle(oracle)
            check(resp)
            print "  -> id:%s\n" % resp.msg
            return resp.msg

    else:
        o = resp.oracles[0]
        print "Oracle %s -> id:%s\n" % ( o.name, o.id )
        return o.id

    return None


def call_oracle(ident, threshold):
    global client, oracle_id
    return client.Run(sum_pb2.Call(oracle_id=oracle_id, args=("\"%s\"" % ident, threshold)))

def get_payload(data):
    raw = data.payload
    if data.compressed:
        raw = zlib.decompress(raw, 16+zlib.MAX_WBITS)
    return json.loads(str(raw))


if __name__ == '__main__':
    client = create_client('127.0.0.1:50051')

    # STEP 1: send the oracle code to the server and get its id
    oracle_id = define_oracle(oracle_file, oracle_name)

    # STEP 2: generate `num_rows` vectors of `num_columns` columns
    # each and ask the server to store them
    print "CREATE (%dx%d) : " % ( num_rows, num_columns ),
    timer_start()
    for row in range (0, num_rows):
        record = gen_record(num_columns)
        resp = client.CreateRecord(record)
        check(resp)
        # msg contains the identifier
        index[resp.msg] = record
    timer_stop()
    
    # STEP 3: for every vector, query the oracle to get a list
    # of vectors that have a big cosine similarity
    print "CALL %s x%d : " % (oracle_name, len(index)),
    timer_start()
    for ident, record in index.iteritems():
        resp = call_oracle(ident, oracle_threshold)
        check(resp)
        neighbours = get_payload(resp.data)
        index[ident] = {
            'record': record,
            'neighbours': neighbours,
        }
    timer_stop()

    # STEP 4: remove all, fin.
    print "DEL x%d : " % len(index),
    timer_start()
    for ident, record in index.iteritems():
        check( client.DeleteRecord(sum_pb2.ById(id=ident)) )
    timer_stop()

    print 

    for ident, obj in index.iteritems():
        n = len(obj['neighbours'])
        if n > 0:
            print "Vector %s has %d neighbours with a cosine similarity >= than %s" % ( ident, n, oracle_threshold )
