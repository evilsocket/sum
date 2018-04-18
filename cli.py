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
        meta=[sum_pb2.NamedValue( \
            name="example_metadata",
            value="Random number is %f" % random.random()
        )]
    )

start = 0
end = 0
num_columns = 475
num_rows = 3000
index = {}
sumcli = None
oracle_name = 'findSimilarVectors'

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

def define_oracle(filename, name):
    global sumcli

    resp = sumcli.FindOracle(sum_pb2.ByName(name=name))
    if resp.success == False:
        print resp.msg
        quit()

    if len(resp.oracles) == 0:
        print "Defining oracle %s ..." % name

        with open( filename, 'r') as fp:
            oracle = sum_pb2.Oracle(name=name, code=fp.read())
            resp = sumcli.CreateOracle(oracle)
            if resp.success == False:
                print resp.msg
                quit()

    else:
        o = resp.oracles[0]
        print "Oracle %s already defined as %s" % ( o.name, o.id )

if __name__ == '__main__':
    channel = grpc.insecure_channel('127.0.0.1:50051')
    sumcli = sum_pb2_grpc.SumServiceStub(channel)

    define_oracle('oracle.js', oracle_name)
    print

    print "CREATE (%dx%d) : " % ( num_rows, num_columns ),
    timer_start()
    for row in range (0, num_rows):
        record = gen_record(num_columns)
        resp = sumcli.CreateRecord(record)

        if resp.success is not True:
            print "ERROR: %s" % resp.msg
            quit()

        # msg contains the identifier
        index[resp.msg] = record
    timer_stop()

    print "READ x%d : " % len(index),
    timer_start()
    for ident, record in index.iteritems():
        resp = sumcli.ReadRecord(sum_pb2.ById(id=ident))

        if resp.success is not True:
            print "Error while querying %s: %s" % (ident, resp.msg)
            quit()
    timer_stop()

    print "DEL x%d : " % len(index),
    timer_start()
    for ident, record in index.iteritems():
        resp = sumcli.DeleteRecord(sum_pb2.ById(id=ident))

        if resp.success is not True:
            print "Error while deleting %s: %s" % (ident, resp.msg)
            quit()
    timer_stop()
