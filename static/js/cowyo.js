$(document).ready(function() {
  var isTyping = false;
  var typingTimer; //timer identifier
  var updateInterval;
  var doneTypingInterval = 100; //time in ms, 5 second for example
  var pollToGetNewestCopyInterval = 10000;
  //on keyup, start the countdown
  $('#emit').keyup(function() {
    clearTimeout(typingTimer);
    clearInterval(updateInterval);
    typingTimer = setTimeout(doneTyping, doneTypingInterval);
  });

  //on keydown, clear the countdown
  $('#emit').keydown(function() {
    clearTimeout(typingTimer);
    clearInterval(updateInterval);
    document.title = "[UNSAVED] " + title_name;
  });

  //user is "finished typing," do something
  function doneTyping() {
    payload = JSON.stringify({ TextData: $('#emit_data').val(), Title: title_name, UpdateServer: true, UpdateClient: false })
    send(payload)
    console.log("Done typing")
    updateInterval = setInterval(updateText, pollToGetNewestCopyInterval);
    document.title = "[SAVED] " + title_name;
  }

  function updateText() {
    console.log("Getting server's latest copy")
    payload = JSON.stringify({ TextData: $('#emit_data').val(), Title: title_name, UpdateServer: false, UpdateClient: true })
    send(payload)
  }

  // websockets
  url = 'ws://'+external_ip+'/ws';
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
      document.title = "[LOADED] " + title_name;
    }
    console.log(data)
  }

  c.onopen = function(){
    // updateText();
    updateInterval = setInterval(updateText, pollToGetNewestCopyInterval);
  }
});
