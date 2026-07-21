<?php

// Pure, side-effect-free helpers for validating a decoded Yivi/IRMA session
// result. These live in their own file so they can be unit-tested directly,
// without running verifysession.php (which reads the request body, sends HTTP
// headers and calls exit).

// The JWT signature only proves the token originated from our Yivi server; it
// does not prove the disclosure itself succeeded. A signed result may report an
// unsuccessful disclosure, so only accept results the server marked as valid
// before trusting the disclosed attributes.
function isDisclosureProofValid($decoded): bool {
    return isset($decoded->proofStatus) && $decoded->proofStatus === 'VALID';
}

// Returns true when the disclosed attributes prove the user meets the age
// restriction (over 18) for any of the supported credential types.
function isAgeAllowed($disclosed): bool {
    $age_restriction = 18;

    $age_key_passport = "pbdf.pbdf.passport.over" . $age_restriction;
    $age_key_idcard = "pbdf.pbdf.idcard.over" . $age_restriction;
    $age_key_drivinglicence = "pbdf.pbdf.drivinglicence.over" . $age_restriction;
    $age_key_nijmegen = "pbdf.nijmegen.ageLimits.over" . $age_restriction;
    $age_key_gemeente = "pbdf.gemeente.personalData.over" . $age_restriction;
    $age_key_demo_gemeente = "irma-demo.gemeente.personalData.over" . $age_restriction;

    foreach ($disclosed as $con) {
        foreach ($con as $attr) {
            if ($attr->id == $age_key_passport
                || $attr->id == $age_key_nijmegen
                || $attr->id == $age_key_idcard
                || $attr->id == $age_key_drivinglicence
                || $attr->id == $age_key_gemeente
                || $attr->id == $age_key_demo_gemeente
            ) {
                return strtolower($attr->rawvalue) == "yes" || strtolower($attr->rawvalue) == "ja";
            }
        }
    }

    return false;
}
