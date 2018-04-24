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

    private function checkResponse($r) {
        if($r->code != \Grpc\STATUS_OK) {
            throw new \Exception($r->details);
        } 
    }

    public function createRecord($meta, $data) {
        $record = new Sum\Record;
        $record->setMeta($meta);
        $record->setData($data);
        list($result, $status) = $this->rpc->CreateRecord($record)->wait();
        $this->checkResponse($status);
        var_dump($status); die();
    }
}
