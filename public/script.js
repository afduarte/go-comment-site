$(function () {
    window.offset = 0;
    window.isEnd = false;
    $.getJSON('get-comments')
        .done(function (data) {
            addComments(data.messages)
        });
    var socket = createWebSocket(location.pathname + 'ws');

    socket.onmessage = function (event) {
        var msg = JSON.parse(event.data);
        addComments([msg]);
    };
    $('#send').on('click', function () {
        socket.send(JSON.stringify({
            name: $('#name').val(),
            message: $('#comment').val(),
            giphy: $('#giphy').val()
        }));
        $('#messages').animate({scrollTop: 0});
        $('#name').val('');
        $('#comment').val('');
        $('#giphy').val('');
    });
    //detects when user reaches the end
    $('#messages').on("scroll", function () {
        var pos = $(this).scrollTop();
        var height = $(this).get(0).scrollHeight;
        var toBottom = height - pos - $(this).height();
        console.log(toBottom);
        if (Math.floor(toBottom) <= 300) {
            console.log('bottom');
            if (!window.loading && !window.isEnd) {
                window.loading = true;
                $.getJSON('get-comments?offset=' + window.offset)
                    .done(function (data) {
                        window.loading = false;
                        window.isEnd = data.isEnd;
                        addComments(data.messages, true)
                    });
            }
        }
    })
});

function createWebSocket(path) {
    var protocolPrefix = (window.location.protocol === 'https:') ? 'wss:' : 'ws:';
    return new WebSocket(protocolPrefix + '//' + location.host + path);
}

function addComments(messages, append) {
    window.offset += messages.length;
    for (var m in messages) {
        var msg = messages[m];

        var container = $('#comment-template').clone();
        container.attr('id', '');
        container.find('.blockquote-title').text(msg.name)
            .append('<span class="label secondary date">' + ago(msg.timestamp) + '</span>');
        if (msg.giphy) {
            container.find('.giphy').append('<img src="' + msg.giphy + '"/>')
        } else {
            container.find('.giphy').remove();
            var quote = container.find('.quote-holder');
            quote.removeClass('small-8');
            quote.addClass('small-12');
        }
        container.find('.blockquote-content').html(msg.message);
        if (append) {
            $('#messages').append(container)
        } else {
            $('#messages').prepend(container)
        }

    }
}

function ago(previous) {

    var msPerMinute = 60 * 1000;
    var msPerHour = msPerMinute * 60;
    var msPerDay = msPerHour * 24;
    var dateOpts = {
        weekday: 'long',
        hour: 'numeric',
        minute: 'numeric'
    };

    var elapsed = Date.now() - previous;

    var number;
    var unit;

    if (elapsed < msPerMinute) {
        number = Math.round(elapsed / 1000);
        unit = 'second';
    }

    else if (elapsed < msPerHour) {
        number = Math.round(elapsed / msPerMinute);
        unit = 'minute';
    }

    else if (elapsed < msPerDay) {
        number = Math.round(elapsed / msPerHour);
        unit = 'hour';
        if (number > 5) {
            return new Date(previous).toLocaleDateString('en-UK', dateOpts);
        }
    }
    else {
        return new Date(previous).toLocaleDateString('en-UK', dateOpts);
    }

    return number + ' ' + unit + (number > 1 ? 's' : '') + ' ago';
}