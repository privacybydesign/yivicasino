<?php
require_once '../vendor/autoload.php';
require_once '../config.php';
require_once __DIR__ . '/verification.php';
use \Firebase\JWT\JWT;
use \Firebase\JWT\Key;

// This script checks JWT validity and checks age restriction to see if ageover.age restiction is present in the disclosed attributes
// only if these are true, it the user is allowed to play in the casino


// token is passed in the body of the request
$json = file_get_contents('php://input');
$data = json_decode($json, true);


if (!isset($data['token']) || empty($data['token'])) {
    echo "No token provided";
    header("HTTP/1.0 400 Bad Request");
    exit;
}

$jwt_pk = file_get_contents(IRMA_SERVER_PUBLICKEY);
$token = $data['token'];

// Allow only a small clock-skew tolerance between this server and the Yivi
// server when validating the JWT's iat/exp claims.
JWT::$leeway = 60;
try {
    $decoded = JWT::decode($token, new Key($jwt_pk, 'RS256'));
} catch (Exception $e) {
    header("HTTP/1.0 403 Forbidden");
    exit;
}

// A valid signature only proves the token came from our Yivi server, not that
// the disclosure itself succeeded. Reject the result unless the server reported
// a valid disclosure before trusting the disclosed attributes.
if (!isDisclosureProofValid($decoded)) {
    header("HTTP/1.0 403 Forbidden");
    echo json_encode(['success' => false]);
    exit;
}

$disclosed = (array) $decoded->disclosed;

if (isAgeAllowed($disclosed)) {
    echo json_encode(['success' => true]);
    http_response_code(200);
} else {
    echo json_encode(['success' => false]);
    http_response_code(403);
}

exit;

?>
