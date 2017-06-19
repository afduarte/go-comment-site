$(function () {
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
        socket.send(JSON.stringify({name: $('#name').val(), message: $('#comment').val()}));
    });
    //detects when user reaches the end
    $('#messages').on("scroll", function () {
        var wrap = document.getElementById('messages');
        var contentHeight = wrap.offsetHeight;
        var yOffset = window.pageYOffset;
        var y = yOffset + window.innerHeight;
        console.log(y, contentHeight);
        if (y >= contentHeight) {

        }
    })
});

function createWebSocket(path) {
    var protocolPrefix = (window.location.protocol === 'https:') ? 'wss:' : 'ws:';
    return new WebSocket(protocolPrefix + '//' + location.host + path);
}

function addComments(messages) {
    for (var m in messages) {
        var msg = messages[m];
        var dateOpts = {
            weekday: 'long',
            hour: 'numeric',
            minute: 'numeric'
        };
        var container = $('#comment-template').clone();
        container.attr('id', '');
        var date = new Date(msg.timestamp).toLocaleDateString('en-UK', dateOpts)
        container.find('.blockquote-title').text(msg.name)
            .append('<span class="label secondary date">' + date + '</span>');
        container.find('.blockquote-title .badge').text(date);
        container.find('.blockquote-content').html(msg.message);
        $('#messages').prepend(container)
    }
}