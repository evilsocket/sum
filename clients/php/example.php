<?php

/*
 * You're not supposed to do this shit but use composer.
 */
foreach( glob('/opt/grpc/src/php/lib/Grpc/*.php') as $file ) {
    require_once $file;
}

$base = realpath(dirname(__FILE__));

foreach( glob("$base/Sum/*.php") as $file ) {
    require_once $file;
}

require_once "$base/GPBMetadata/Proto/Sum.php";
require_once "$base/SumClient.php";
 
function rand_data($columns) {
    $data = [];
    for( $i = 0; $i < $columns; $i++ ) {
        $data []= mt_rand(0,1000000)/1000000;
    }
    return $data;
}

$num_rows = 300;
$num_columns = 100;
$index = [];
$client = new SumClient('127.0.0.1:50051'); 

echo "@ Creating $num_rows vectors ...\n";

for( $i = 0; $i < $num_rows; $i++ ) { 
    $record = $client->CreateRecord([], rand_data($num_columns) );
    $index[$record->getId()] = $record;
}

$oracle_id = $client->DefineOracle("$base/../example_oracles/findsimilar.js", "findSimilar");

foreach( $index as $rec_id => $record ) {
    $index[$rec_id] = [
        'record' => $record,
        'neighbours' => $client->InvokeOracle($oracle_id, [ $rec_id, 0.8 ]),
    ];
}

echo "@ Deleting records ...\n";

foreach( $index as $rec_id => $record ) {
    $client->DeleteRecord($rec_id);
}

foreach( $index as $rec_id => $obj ) {
    $n = count($obj['neighbours']);
    if( $n > 0 ) {
        echo "Vector $rec_id has $n neighbours with a cosine similarity >= 0.8\n";
    }
}
