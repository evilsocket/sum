<?php
// GENERATED CODE -- DO NOT EDIT!

namespace Sum;

/**
 */
class SumServiceClient extends \Grpc\BaseStub {

    /**
     * @param string $hostname hostname
     * @param array $opts channel options
     * @param \Grpc\Channel $channel (optional) re-use channel object
     */
    public function __construct($hostname, $opts, $channel = null) {
        parent::__construct($hostname, $opts, $channel);
    }

    /**
     * vectors CRUD
     * @param \Sum\Record $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function CreateRecord(\Sum\Record $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/sum.SumService/CreateRecord',
        $argument,
        ['\Sum\RecordResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * @param \Sum\Record $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function UpdateRecord(\Sum\Record $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/sum.SumService/UpdateRecord',
        $argument,
        ['\Sum\RecordResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * @param \Sum\ById $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function ReadRecord(\Sum\ById $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/sum.SumService/ReadRecord',
        $argument,
        ['\Sum\RecordResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * @param \Sum\ListRequest $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function ListRecords(\Sum\ListRequest $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/sum.SumService/ListRecords',
        $argument,
        ['\Sum\RecordListResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * @param \Sum\ById $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function DeleteRecord(\Sum\ById $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/sum.SumService/DeleteRecord',
        $argument,
        ['\Sum\RecordResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * oracles CRUD
     * @param \Sum\Oracle $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function CreateOracle(\Sum\Oracle $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/sum.SumService/CreateOracle',
        $argument,
        ['\Sum\OracleResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * @param \Sum\Oracle $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function UpdateOracle(\Sum\Oracle $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/sum.SumService/UpdateOracle',
        $argument,
        ['\Sum\OracleResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * @param \Sum\ById $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function ReadOracle(\Sum\ById $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/sum.SumService/ReadOracle',
        $argument,
        ['\Sum\OracleResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * @param \Sum\ByName $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function FindOracle(\Sum\ByName $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/sum.SumService/FindOracle',
        $argument,
        ['\Sum\OracleResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * @param \Sum\ById $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function DeleteOracle(\Sum\ById $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/sum.SumService/DeleteOracle',
        $argument,
        ['\Sum\OracleResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * execute a call to a oracle given its id
     * @param \Sum\Call $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function Run(\Sum\Call $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/sum.SumService/Run',
        $argument,
        ['\Sum\CallResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * get info about the service
     * @param \Sum\PBEmpty $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     */
    public function Info(\Sum\PBEmpty $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/sum.SumService/Info',
        $argument,
        ['\Sum\ServerInfo', 'decode'],
        $metadata, $options);
    }

}
