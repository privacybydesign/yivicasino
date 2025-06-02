<?php require_once "config.php"?>
<!DOCTYPE html>
<html lang="<?= $language ?>">

<head>
  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
  <meta name="keywords" content="Yivi, Casino, IRMA, privacy, security">
  <meta name="description" content="Experimental Yivi Casino with age verification and fun!">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">

  <title>Yivi Casino - Age Checked, Fun Unlocked!</title>

  <link href="css/mosaic.css" rel="stylesheet" type="text/css" />
  <link href="css/casino.css" rel="stylesheet" type="text/css" />
  <link href="css/slotmachine.css" rel="stylesheet" type="text/css" />

  <link href="node_modules/bootstrap/dist/css/bootstrap.min.css" rel="stylesheet" />

  <script src="node_modules/jquery/dist/jquery.min.js" type="text/javascript"></script>
  <script src="js/mosaic.1.0.1.min.js" type="text/javascript"></script>
  <script src="node_modules/mustache/mustache.min.js" type="text/javascript"></script>
  <script src="node_modules/bootstrap/dist/js/bootstrap.min.js" type="text/javascript"></script>

  <script src="node_modules/@privacybydesign/yivi-frontend/dist/yivi.js" type="text/javascript" async></script>
  <script src="js/slotmachine.js" type="text/javascript"></script>

  <script type="text/javascript">
    $(window).on('load', function() {
      $('#enterModal').modal('show');
    });

    function enterCasino() {
      let onVerifySuccess = function (data) {
        console.log("IRMA Session data:", data);

        fetch("php/verifysession.php", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },

          body: JSON.stringify({
            token: data,
          }),
        })
        .then((response) => {
          if (response.status === 403) {
            $('#enterModalLabel').text('Age Verification failed');
            $('#enterModal .modal-body').html('<p><img src="img/under-18-sign.jpg" /><br/><br/>We could not verify your age is over 18. Therefor, you are not allowed to play on our slotmachine.</p>');
            $('#enterModal .modal-footer').hide();
            return;
          }
          if (!response.ok) {
            throw new Error("HTTP error, status = " + response.status);
          }
          return response.json();
        })
        .then((result) => {
          if (result.success) {
            // Users age is verified, proceed to play in the casino
            $('#enterModal').modal('hide');
            $('#slotMachineModal').modal('show');
            playSlotMachine();
          } else {
            $('#enterModalLabel').text('Age Verification failed');
            $('#enterModal .modal-body').html('Could not verify your age. Please try again.');
          }
        })
      };

      let url = "php/session.php?type=verification&" + Math.random(); // Append randomness so that IE doesn't consider it 304 not modified

      yivi.newPopup({
        language: '<?= $language ?>',
        session: {
          start: {
            url: () => url,
          },
          result: {
            url: (o, {sessionPtr, sessionToken}) => `${sessionPtr.u.split('/irma')[0]}/session/${sessionToken}/result-jwt`,
            parseResponse: (r) => r.text(),
          },
        },
      })
      .start()
      .then(onVerifySuccess, onIrmaFailure);
    }

    function onIrmaFailure(data) {
      if(data === 'Aborted')
        showWarning(data);
      else
        showError(data);
    }
  </script>
</head>

<body style="background-image: url('img/bg.jpg'); background-size: cover; background-repeat: no-repeat; background-attachment: fixed;">
  <div id="enterModal" class="modal fade" tabindex="-1" role="dialog"
    aria-labelledby="enterModalLabel" aria-hidden="true" data-bs-keyboard="false" data-bs-backdrop="static">
    <div class="modal-dialog">
      <div class="modal-content">
        <div class="modal-header" style="margin: auto;text-align: center;">
          <h4 id="enterModalLabel"> ! Welcome to the Yivi Casino ! </h4>
        </div>
        <div class="modal-body" style="text-align: center;">
          Click on the button below, open the Yivi app, scan the QR code using Yivi and verify your age to enter the casino!
        </div>
        <div class="modal-footer" style="margin: auto;text-align: center;">
          <button class="btn" aria-hidden="true" id="enterCasinoBtn" onclick="enterCasino()">
            <img src="img/enter-btn.png" alt="Enter Casino" width="400" height="100" />
          </button>
        </div>
      </div>
    </div>
  </div>
  <div id="slotMachineModal" class="modal fade" tabindex="-1" role="dialog"
    aria-labelledby="slotMachineModalLabel" aria-hidden="true" data-bs-keyboard="false" data-bs-backdrop="static">
    <div class="modal-dialog modal-lg">
      <div class="modal-content">
        <div class="modal-header" style="margin: auto;text-align: center;">
          <h4 id="slotMachineModalLabel"> ! Thanks for playing at Yivi Casino ! </h4>
        </div>
        <div class="modal-body" style="text-align: center;">
          <div id="slotmachine">
            <div class="window-border">
                <div class="window">
                    <div class="outer-spacer"></div>
                    <div class="outer-col">
                        <div class="col"></div>
                    </div>
                    <div class="outer-spacer"></div>
                    <div class="outer-col">
                        <div class="col"></div>
                    </div>
                    <div class="outer-spacer"></div>
                    <div class="outer-col">
                        <div class="col"></div>
                    </div>
                    <div class="outer-spacer"></div>
                    <div class="outer-col">
                        <div class="col"></div>
                    </div>
                    <div class="outer-spacer"></div>
                    <div class="outer-col">
                        <div class="col"></div>
                    </div>
                    <div class="outer-spacer"></div>
                </div>
            </div>
            <input type="button" onclick="spin(this)" class="start-button" value="Spin"/>
        </div>
      </div>
    </div>
  </div>
</body>
</html>
