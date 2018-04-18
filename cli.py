#!/usr/bin/python
import sys
import os
import random
import datetime
import grpc

path = os.path.join(os.path.abspath(os.path.dirname(__file__)), "proto")
sys.path.append(path)

import sum_pb2
import sum_pb2_grpc 

def gen_record(columns):
    return sum_pb2.Record( \
        data=[random.uniform(0,100) for i in range(0, columns)],
        meta=[sum_pb2.Meta( \
            name="example_metadata",
            value="Random number is %f" % random.random()
        )]
    )

start = 0
end = 0
num_columns = 500
num_rows = 1000
index = {}
sumcli = None

def timer_start():
    global start
    sys.stdout.flush()
    start = datetime.datetime.now()

def timer_stop():
    global start, end, index, sumcli
    end = datetime.datetime.now()
    diff = end - start
    elapsed_ms = (diff.days * 86400000) + (diff.seconds * 1000) + (diff.microseconds / 1000)
    print "%d ms / %.2fms avg" % ( elapsed_ms, float(elapsed_ms) / float(len(index)) )
    # print sumcli.Info(sum_pb2.Empty())

if __name__ == '__main__':
    channel = grpc.insecure_channel('127.0.0.1:50051')
    sumcli = sum_pb2_grpc.SumServiceStub(channel)

    print "CREATE (%dx%d) : " % ( num_rows, num_columns ),
    timer_start()
    for row in range (0, num_rows):
        record = gen_record(num_columns)
        resp = sumcli.Create(record)

        if resp.success is not True:
            print "ERROR: %s" % resp.msg
            quit()

        # msg contains the identifier
        index[resp.msg] = record
    timer_stop()

    print "READ x%d : " % len(index),
    timer_start()
    for ident, record in index.iteritems():
        query = sum_pb2.Query(id=ident)
        resp = sumcli.Read(query)

        if resp.success is not True:
            print "Error while querying %s: %s" % (ident, resp.msg)
            quit()
    timer_stop()

    print "DEL x%d : " % len(index),
    timer_start()
    for ident, record in index.iteritems():
        query = sum_pb2.Query(id=ident)
        resp = sumcli.Delete(query)

        if resp.success is not True:
            print "Error while deleting %s: %s" % (ident, resp.msg)
            quit()
    timer_stop()
