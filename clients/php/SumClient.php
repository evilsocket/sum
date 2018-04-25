<?php

class SumClient {
    const MAX_MESSAGE_SIZE = 10 * 1024 * 1024;

    private $connection_string;
    private $options = [];
    private $rpc = NULL;

    public function __construct($conn_string) {
        $this->connection_string = $conn_string;
        $this->options = [
            'grpc.max_send_message_length' => self::MAX_MESSAGE_SIZE, 
            'grpc.max_receive_message_length' => self::MAX_MESSAGE_SIZE,
            'credentials' => Grpc\ChannelCredentials::createInsecure()
        ];

        $this->rpc = new Sum\SumServiceClient($this->connection_string, $this->options);
    }

    private function checkResponse($s, $r) {
        if($s->code != \Grpc\STATUS_OK) {
            throw new \Exception($s->details);
        }

        if(!$r->getSuccess()) {
            throw new \Exception($r->getMsg());
        }
    }

    public function CreateRecord($meta, $data) {
        $record = new Sum\Record;
        $record->setMeta($meta);
        $record->setData($data);
        list($result, $status) = $this->rpc->CreateRecord($record)->wait();
        $this->checkResponse($status, $result);
        $id = $result->getMsg();
        $record->setId($id);

        return $record;
    }

    public function ReadRecord($identifier) {
        $id = (int)$identifier;
        $byId = new Sum\ById();
        $byId->setId($id);
        list($result, $status) = $this->rpc->ReadRecord($byId)->wait();
        $this->checkResponse($status, $result);

        return $result->getRecord();
    }

    public function DeleteRecord($identifier) {
        $id = (int)$identifier;
        $byId = new Sum\Byid;
        $byId->setId($id);
        list($result, $status) = $this->rpc->DeleteRecord($byId)->wait();
        $this->checkResponse($status, $result);
    }

    public function DefineOracle($filename, $name) {
        $byName = new Sum\ByName;
        $byName->setName($name);
        list($result, $status) = $this->rpc->FindOracle($byName)->wait();
        $this->checkResponse($status, $result);

        $oracles = $result->getOracles();
        if( count($oracles) == 0 ) {
            $code = file_get_contents($filename);
            $oracle = new Sum\Oracle;
            $oracle->setName($name);
            $oracle->setCode($code);
            list($result, $status) = $this->rpc->CreateOracle($oracle)->wait();
            $this->checkResponse($status, $result);
            return (int)$result->getMsg();
        }
        else {
            return $oracles[0]->getId();
        }
    }

    /*
    def invoke_oracle(self, oracle_id, args):
        resp = self._rpc.Run(proto.Call(oracle_id=oracle_id, args=map(json.dumps, args)))
        self._check_resp(resp)
        return self._get_oracle_payload(resp.data)
     */
    public function InvokeOracle($oracle_id, $args) {
    }
}
