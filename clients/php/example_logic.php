<?php

/*
 * You're not supposed to do this shit but use composer.
 */
foreach( glob('/opt/grpc/src/php/lib/Grpc/*.php') as $file ) {
    require_once $file;
}

foreach( glob('Sum/*.php') as $file ) {
    require_once $file;
}

require_once 'GPBMetadata/Proto/Sum.php';

require('SumClient.php');


$client = new SumClient('127.0.0.1:50051'); 
$client->CreateRecord(["zio" => "cane"], [0.00,0.2] );
