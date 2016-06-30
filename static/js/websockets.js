var selfDestruct = false;

$(document).ready(function() {
  var isTyping = false;
  var typingTimer; //timer identifier
  var updateInterval;
  var uhohTimer;
  var doneTypingInterval = 500; //time in ms, 5 second for example
  var pollToGetNewestCopyInterval = 20000;
  //on keyup, start the countdown
  $('#emit').keyup(function() {
    clearTimeout(typingTimer);
    clearInterval(updateInterval);
    $('#saveInfo').removeClass().addClass("glyphicon glyphicon-floppy-remove");
    typingTimer = setTimeout(doneTyping, doneTypingInterval);
  });

  //on keydown, clear the countdown
  $('#emit').keydown(function() {
    clearTimeout(typingTimer);
    clearInterval(updateInterval);
    $('#saveInfo').removeClass().addClass("glyphicon glyphicon-floppy-remove");
    document.title = '✗ ' + title_name;
  });

  //user is "finished typing," do something
  function doneTyping() {
    payload = JSON.stringify({ TextData: currentText(), Title: title_name, UpdateServer: true, UpdateClient: false })
    send(payload)
    uhohTimer = setTimeout(uhoh, 3000);
    $('#saveInfo').removeClass().addClass("glyphicon glyphicon-floppy-open");
    console.log("Done typing")
    updateInterval = setInterval(updateText, pollToGetNewestCopyInterval);
    document.title = "✓ " + title_name;
    if (currentText().indexOf("self-destruct\n") > -1 || currentText().indexOf("\nself-destruct") > -1) {
      if (selfDestruct == false) {
        selfDestruct = true;
        swal({   title: "Info",   text: "This page is primed to self-destruct.",   timer: 1000,   showConfirmButton: true });
      }
    } else {
      if (selfDestruct == true) {
        selfDestruct = false;
        swal({   title: "Info",   text: "This page is no longer primed to self-destruct.",   timer: 1000,   showConfirmButton: true });
      }
    }
  }

  function uhoh() {
      $('#saveInfo').removeClass().addClass("glyphicon glyphicon-remove");
      setInterval(location.reload(), 1000);
  }

  function updateText() {
    console.log("Getting server's latest copy")
    payload = JSON.stringify({ TextData: currentText(), Title: title_name, UpdateServer: false, UpdateClient: true })
    send(payload)
  }

  // websockets
  url = socketType + '://'+external_ip+'/ws';
  c = new WebSocket(url);

  send = function(data){

    console.log("Sending: " + data)
    c.send(data)
  }

  c.onmessage = function(msg){
    console.log(msg)
    data = JSON.parse(msg.data);
    if (data.UpdateClient == true) {
      console.log("Updating...")
      $('#emit_data').val(data.TextData)
      document.title = " " + title_name;
    }
    console.log((data.TextData == "saved"))
    console.log(data.TextData)
    if (data.TextData == "saved") {
      $('#saveInfo').removeClass().addClass("glyphicon glyphicon-floppy-saved");
      clearTimeout(uhohTimer);
    }
    console.log(data)
  }

  c.onopen = function(){
    // updateText();
    updateInterval = setInterval(updateText, pollToGetNewestCopyInterval);
  }


  $('.postselfdestruct').click(function(event) {
    event.preventDefault();
    $('#emit_data').val("self-destruct\n\n"+currentText() + "\n\n");
    doneTyping();
  });

});
