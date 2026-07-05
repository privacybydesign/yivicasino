<?php

// Dependency-free unit tests for the pure validation helpers in
// www/php/verification.php. Run with:
//
//     php www/tests/verification_test.php
//
// This deliberately avoids PHPUnit/Composer: the repo ships only runtime
// dependencies and the Docker image builds with `composer install --no-dev`
// against the committed lock file, so adding a dev dependency here would
// desync composer.lock and break that build. A plain PHP script gives the
// same coverage with no toolchain setup and exits non-zero on any failure.

require_once __DIR__ . '/../php/verification.php';

$failures = 0;

function check(bool $cond, string $name): void {
    global $failures;
    if ($cond) {
        echo "PASS: $name\n";
    } else {
        echo "FAIL: $name\n";
        $failures++;
    }
}

// Build a single disclosed attribute object as it appears in a Yivi result JWT.
function attr(string $id, string $rawvalue): object {
    return (object) ['id' => $id, 'rawvalue' => $rawvalue];
}

// --- isDisclosureProofValid -------------------------------------------------

check(isDisclosureProofValid((object) ['proofStatus' => 'VALID']) === true,
    'a valid proof status is accepted');
check(isDisclosureProofValid((object) ['proofStatus' => 'INVALID']) === false,
    'a non-valid proof status is rejected');
check(isDisclosureProofValid((object) ['proofStatus' => 'EXPIRED']) === false,
    'an expired proof status is rejected');
check(isDisclosureProofValid((object) []) === false,
    'a missing proof status is rejected');
check(isDisclosureProofValid((object) ['proofStatus' => 'valid']) === false,
    'proof status matching is case-sensitive');

// --- isAgeAllowed -----------------------------------------------------------

check(isAgeAllowed([[attr('pbdf.pbdf.idcard.over18', 'yes')]]) === true,
    'over-18 "yes" from a supported credential is allowed');
check(isAgeAllowed([[attr('pbdf.pbdf.passport.over18', 'ja')]]) === true,
    'over-18 "ja" from a supported credential is allowed');
check(isAgeAllowed([[attr('irma-demo.gemeente.personalData.over18', 'YES')]]) === true,
    'the raw value comparison is case-insensitive');
check(isAgeAllowed([[attr('pbdf.pbdf.idcard.over18', 'no')]]) === false,
    'over-18 "no" is rejected');
check(isAgeAllowed([[attr('pbdf.pbdf.email.email', 'x@example.com')]]) === false,
    'an unrelated attribute does not grant access');
check(isAgeAllowed([]) === false,
    'an empty disclosure is rejected');

// ---------------------------------------------------------------------------

if ($failures > 0) {
    fwrite(STDERR, "\n$failures test(s) failed\n");
    exit(1);
}

echo "\nAll tests passed\n";
