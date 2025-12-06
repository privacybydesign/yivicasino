<?php
require_once '../vendor/autoload.php';
require_once '../config.php';
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

JWT::$leeway = 60 * 60;
try {
    $decoded = JWT::decode($token, new Key($jwt_pk, 'RS256'));
} catch (Exception $e) {
    header("HTTP/1.0 403 Forbidden");
    exit;
}
$disclosed = (array) $decoded->disclosed;

function isMember($disclosed) {
    $member_key = IRMATUBE_CREDENTIAL_ID . ".type";
    foreach ($disclosed as $con) {
        foreach ($con as $attr) {
            if ($attr->id == $member_key) {
                return $attr->rawvalue === "regular" || $attr->rawvalue === "premium";
            }
        }
    }

    return false;
}

function isAgeAllowed($disclosed) {
    $age_restriction = 18;

    $age_key_passport = "pbdf.pbdf.passport.over" . $age_restriction;
    $age_key_nijmegen = "pbdf.nijmegen.ageLimits.over" . $age_restriction;
    $age_key_gemeente = "pbdf.gemeente.personalData.over" . $age_restriction;
    $age_key_demo_gemeente = "irma-demo.gemeente.personalData.over" . $age_restriction;

    foreach ($disclosed as $con) {
        foreach ($con as $attr) {
            if ($attr->id == $age_key_passport
                || $attr->id == $age_key_nijmegen
                || $attr->id == $age_key_gemeente
                || $attr->id == $age_key_demo_gemeente
            ) {
                return strtolower($attr->rawvalue) == "yes" || strtolower($attr->rawvalue) == "ja";
            }
        }
    }

    return false;
}


if (isAgeAllowed($disclosed)) {
    echo json_encode(['success' => true]);
    http_response_code(200);
} else {
    echo json_encode(['success' => false]);
    http_response_code(403);
}

exit;

?>
