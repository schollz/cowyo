"use strict";
var oulipo = false;

$(window).load(function() {
    // Returns a function, that, as long as it continues to be invoked, will not
    // be triggered. The function will be called after it stops being called for
    // N milliseconds. If `immediate` is passed, trigger the function on the
    // leading edge, instead of the trailing.
    function debounce(func, wait, immediate) {
        var timeout;
        return function() {
            $('#saveEditButton').removeClass()
            $('#saveEditButton').text("Editing");
            var context = this,
                args = arguments;
            var later = function() {
                timeout = null;
                if (!immediate) func.apply(context, args);
            };
            var callNow = immediate && !timeout;
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
            if (callNow) func.apply(context, args);
        };
    };

    // This will apply the debounce effect on the keyup event
    // And it only fires 500ms or half a second after the user stopped typing
    var prevText = $('#userInput').val();
    console.log("debounce: " + window.cowyo.debounceMS)
    $('#userInput').on('keyup', debounce(function() {
        if (prevText == $('#userInput').val()) {
            return // no changes
        }
        prevText = $('#userInput').val();

        if (oulipo) {
            $('#userInput').val($('#userInput').val().replace(/e/g,""));
        }
        $('#saveEditButton').removeClass()
        $('#saveEditButton').text("Saving")
        upload();
    }, window.cowyo.debounceMS));

    var latestUpload = null, needAnother = false;
    function upload() {
        // Prevent concurrent uploads
        if (latestUpload != null) {
            needAnother = true;
            return
        }
        latestUpload = $.ajax({
            type: 'POST',
            url: '/update',
            data: JSON.stringify({
                new_text: $('#userInput').val(),
                page: window.cowyo.pageName,
                fetched_at: window.lastFetch,
            }),
            success: function(data) {
                latestUpload = null;

                $('#saveEditButton').removeClass()
                if (data.success == true) {
                    $('#saveEditButton').addClass("success");
                    window.lastFetch = data.unix_time;

                    if (needAnother) {
                        upload();
                    };
                } else {
                    $('#saveEditButton').addClass("failure");
                }
                $('#saveEditButton').text(data.message);
                needAnother = false;
            },
            error: function(xhr, error) {
                latestUpload = null;
                needAnother = false;
                $('#saveEditButton').removeClass()
                $('#saveEditButton').addClass("failure");
                $('#saveEditButton').text(error);
            },
            contentType: "application/json",
            dataType: 'json'
        });
    }

    function primeForSelfDestruct() {
        $.ajax({
            type: 'POST',
            url: '/prime',
            data: JSON.stringify({
                page: window.cowyo.pageName,
            }),
            success: function(data) {
                $('#saveEditButton').removeClass()
                if (data.success == true) {
                    $('#saveEditButton').addClass("success");
                } else {
                    $('#saveEditButton').addClass("failure");
                }
                $('#saveEditButton').text(data.message);
            },
            error: function(xhr, error) {
                $('#saveEditButton').removeClass()
                $('#saveEditButton').addClass("failure");
                $('#saveEditButton').text(error);
            },
            contentType: "application/json",
            dataType: 'json'
        });
    }

    function lockPage(passphrase) {
        $.ajax({
            type: 'POST',
            url: '/lock',
            data: JSON.stringify({
                page: window.cowyo.pageName,
                passphrase: passphrase
            }),
            success: function(data) {
                $('#saveEditButton').removeClass()
                if (data.success == true) {
                    $('#saveEditButton').addClass("success");
                } else {
                    $('#saveEditButton').addClass("failure");
                }
                $('#saveEditButton').text(data.message);
                if (data.success == true && $('#lockPage').text() == "Lock") {
                    window.location = "/" + window.cowyo.pageName + "/view";
                }
                if (data.success == true && $('#lockPage').text() == "Unlock") {
                    window.location = "/" + window.cowyo.pageName + "/edit";
                }
            },
            error: function(xhr, error) {
                $('#saveEditButton').removeClass()
                $('#saveEditButton').addClass("failure");
                $('#saveEditButton').text(error);
            },
            contentType: "application/json",
            dataType: 'json'
        });
    }

    function publishPage() {
        $.ajax({
            type: 'POST',
            url: '/publish',
            data: JSON.stringify({
                page: window.cowyo.pageName,
                publish: $('#publishPage').text() == "Publish"
            }),
            success: function(data) {
                $('#saveEditButton').removeClass()
                if (data.success == true) {
                    $('#saveEditButton').addClass("success");
                } else {
                    $('#saveEditButton').addClass("failure");
                }
                $('#saveEditButton').text(data.message);
                if (data.message == "Unpublished") {
                    $('#publishPage').text("Publish");
                } else {
                    $('#publishPage').text("Unpublish");
                }
            },
            error: function(xhr, error) {
                $('#saveEditButton').removeClass()
                $('#saveEditButton').addClass("failure");
                $('#saveEditButton').text(error);
            },
            contentType: "application/json",
            dataType: 'json'
        });
    }

    function encryptPage(passphrase) {
        $.ajax({
            type: 'POST',
            url: '/encrypt',
            data: JSON.stringify({
                page: window.cowyo.pageName,
                passphrase: passphrase
            }),
            success: function(data) {
                $('#saveEditButton').removeClass()
                if (data.success == true) {
                    $('#saveEditButton').addClass("success");
                } else {
                    $('#saveEditButton').addClass("failure");
                }
                $('#saveEditButton').text(data.message);

                if (data.success == true && $('#encryptPage').text() == "Encrypt") {
                    window.location = "/" + window.cowyo.pageName + "/view";
                }
                if (data.success == true && $('#encryptPage').text() == "Decrypt") {
                    window.location = "/" + window.cowyo.pageName + "/edit";
                }
            },
            error: function(xhr, error) {
                $('#saveEditButton').removeClass()
                $('#saveEditButton').addClass("failure");
                $('#saveEditButton').text(error);
            },
            contentType: "application/json",
            dataType: 'json'
        });
    }

    function clearOld() {
        $.ajax({
            type: 'DELETE',
            url: '/oldlist',
            data: JSON.stringify({
                page: window.cowyo.pageName
            }),
            success: function(data) {
                $('#saveEditButton').removeClass()
                if (data.success == true) {
                    $('#saveEditButton').addClass("success");
                } else {
                    $('#saveEditButton').addClass("failure");
                }
                $('#saveEditButton').text(data.message);
                if (data.success == true) {
                    window.location = "/" + window.cowyo.pageName + "/list";
                }
            },
            error: function(xhr, error) {
                $('#saveEditButton').removeClass();
                $('#saveEditButton').addClass("failure");
                $('#saveEditButton').text(error);
            },
            contentType: "application/json",
            dataType: 'json'
        });
    }

    $("#encryptPage").click(function(e) {
        e.preventDefault();
        var passphrase = prompt("Please enter a passphrase. Note: Encrypting will remove all previous history.", "");
        if (passphrase != null) {
            encryptPage(passphrase);
        }
    });

    $("#erasePage").click(function(e) {
        e.preventDefault();
        var r = confirm("Are you sure you want to erase?");
        if (r == true) {
            window.location = "/" + window.cowyo.pageName + "/erase";
        } else {
            x = "You pressed Cancel!";
        }
    });

    $("#selfDestructPage").click(function(e) {
        e.preventDefault();
        var r = confirm("This will erase the page the next time it is opened, are you sure you want to do that?");
        if (r == true) {
            primeForSelfDestruct();
        } else {
            x = "You pressed Cancel!";
        }
    });

    $("#lockPage").click(function(e) {
        e.preventDefault();

        var passphrase;
        if ($('#lockPage').text() == "Lock")
            passphrase = prompt("Please enter a passphrase to lock", "");
        else
            passphrase = prompt("Please enter a passphrase to unlock", "");

        if (passphrase != null) {
            if ($('#lockPage').text() == "Lock") {
                $('#saveEditButton').removeClass();
                $("#saveEditButton").text("Locking");
            } else {
                $('#saveEditButton').removeClass();
                $("#saveEditButton").text("Unlocking");
            }
            lockPage(passphrase);
            // POST encrypt page
            // reload page
        }
    });

    $("#publishPage").click(function(e) {
        e.preventDefault();
        var message = " This will add your page to the sitemap.xml so it will be indexed by search engines.";
        if ($('#publishPage').text() == "Unpublish") {
            message = "";
        }
        var confirmed = confirm("Are you sure?" + message);
        if (confirmed == true) {
            if ($('#publishPage').text() == "Unpublish") {
                $('#saveEditButton').removeClass();
                $("#saveEditButton").text("Unpublishing");
            } else {
                $('#saveEditButton').removeClass();
                $("#saveEditButton").text("Publishing");
            }
            publishPage();
        }
    });

    $("#clearOld").click(function(e) {
        e.preventDefault();
        var r = confirm("This will erase all cleared list items, are you sure you want to do that? (Versions will stay in history).");
        if (r == true) {
            clearOld()
        } else {
            x = "You pressed Cancel!";
        }
    });

    $("textarea").keydown(function(e) {
        if(e.keyCode === 9) { // tab was pressed
            // get caret position/selection
            var start = this.selectionStart;
            var end = this.selectionEnd;

            var $this = $(this);
            var value = $this.val();

            // set textarea value to: text before caret + tab + text after caret
            $this.val(value.substring(0, start)
                        + "\t"
                        + value.substring(end));

            // put caret at right position again (add one for the tab)
            this.selectionStart = this.selectionEnd = start + 1;

            // prevent the focus lose
            e.preventDefault();
        }
    });

    $('.deletable').click(function(event) {
      event.preventDefault();
      var lineNum = $(this).attr('id');

      $.ajax({
        url: "/listitem" + '?' + $.param({
          "lineNum": lineNum,
          "page": window.cowyo.pageName
        }),
        type: 'DELETE',
        success: function() {
          window.location.reload(true);
        }
      });
      event.target.classList.add('deleting');
    });
});

// TODO: Avoid uploading the same thing twice (check if it's already present while allowing failed uploads to be overwritten?)
function onUploadFinished(file) {
    this.removeFile(file);
    var cursorPos = $('#userInput').prop('selectionStart');
    var cursorEnd = $('#userInput').prop('selectionEnd');
    var v = $('#userInput').val();
    var textBefore = v.substring(0,  cursorPos);
    var textAfter  = v.substring(cursorPos, v.length);
    var message = 'uploaded file';
    if (cursorEnd > cursorPos) {
        message = v.substring(cursorPos, cursorEnd);
        textAfter = v.substring(cursorEnd, v.length);
    }
    var prefix = '';
    if (file.type.startsWith("image")) {
        prefix = '!';
    }
    var extraText = prefix+'['+file.xhr.getResponseHeader("Location").split('filename=')[1]+'](' +
        file.xhr.getResponseHeader("Location") +
        ')';

    $('#userInput').val(
        textBefore +
        extraText +
        textAfter
    );

    // Select the newly-inserted link
    $('#userInput').prop('selectionStart', cursorPos);
    $('#userInput').prop('selectionEnd', cursorPos + extraText.length);
    $('#userInput').trigger('keyup'); // trigger a save
}
