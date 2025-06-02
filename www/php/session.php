<?php
require_once '../vendor/autoload.php';
require_once '../config.php';

function start_session($sessionrequest) {

    $jsonsr = json_encode($sessionrequest);

    $api_call = array(
        'http' => array(
            'method' => 'POST',
            'header' => "Content-type: application/json\r\n"
                . "Content-Length: " . strlen($jsonsr) . "\r\n"
                . "Authorization: " . IRMA_SERVER_API_TOKEN . "\r\n",
            'content' => $jsonsr
        )
    );

    $resp = file_get_contents(IRMA_SERVER_URL . '/session', false, stream_context_create($api_call));
    if (! $resp) {
        trigger_error("Failed to start session", E_USER_ERROR);
    }
    return $resp;
}

function start_verification_session() {
    $age = 18;
    $attrs[] = [
        ["pbdf.pbdf.ageLimits.over" . $age],
        ["pbdf.nijmegen.ageLimits.over" . $age ],
        ["pbdf.gemeente.personalData.over" . $age ],
        ["pbdf.pilot-amsterdam.idcard.over" . $age ],
        ["pbdf.pilot-amsterdam.passport.over" . $age ],
        ["irma-demo.nijmegen.ageLimits.over" . $age ],
        ["irma-demo.gemeente.personalData.over" . $age ],
    ];
    return start_session([
        "@context" => "https://irma.app/ld/request/disclosure/v2",
        "disclose" => $attrs
    ]);
}

if (empty($_REQUEST['type'])) {
    header("HTTP/1.0 400 Bad Request");
    exit;
}

$type = $_REQUEST['type'];
switch ($type) {
    case "verification":
        echo start_verification_session();
        break;
    default:
        header("HTTP/1.0 400 Bad Request");
        exit;
}

exit;
