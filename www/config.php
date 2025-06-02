<?php

define('ROOT_DIR', getenv('ROOT_DIR') ?: __DIR__ . '/../data/');

// URL and JWT public key of your IRMA server, also include your access token if enabled on your server
define('IRMA_SERVER_URL',getenv('IRMA_SERVER_URL') ?: 'http://localhost:8088');
define('IRMA_SERVER_API_TOKEN', getenv('IRMA_SERVER_API_TOKEN')?: '');
define('IRMA_SERVER_PUBLICKEY',getenv('IRMA_SERVER_PUBLICKEY')?: ROOT_DIR . '/pk.pem');

$language = 'en';
