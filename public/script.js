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
    $('#send').on('click', function (e) {
        e.preventDefault();
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
    $('body').on('click', '.upvote', function (e) {
        var btn = $(this);
        if (btn.hasClass('voted')) return;
        var id = $(this).parents('.blockquote-container').attr('id');
        console.log('upvoting ' + id);
        $.getJSON('upvote?id=' + id)
            .done(function (msg) {
                console.log(msg);
                btn.removeClass('fa-arrow-circle-o-up');
                btn.addClass('fa-check-circle-o');
                btn.addClass('voted');
                if (!msg.error && msg.upvotes) {
                    $('#' + msg.timestamp + ' .upvotes span').text(formatVotes(msg.upvotes));
                }
            })
    });
    //detects when user reaches the end
    $('#messages').on("scroll", function () {
        var pos = $(this).scrollTop();
        var height = $(this).get(0).scrollHeight;
        var toBottom = height - pos - $(this).height();
        if (Math.floor(toBottom) <= 300) {
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
    });

    $('#show-comment').on('click', function () {
        $(this).hide('fade');
        $('#new-comment').removeClass('hidden');
    });

    $('#hide-comment').on('click', function () {
        $('#new-comment').addClass('hidden');
        $('#show-comment').show('fade');
    });

    setInterval(function () {
        $('.blockquote-container').each(function () {
            $(this).find('span.label.date').text(ago(parseInt($(this).attr('id'))))
        })
    }, 3000)
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
        container.attr('id', msg.timestamp);
        container.find('.blockquote-title').html(twemoji.parse(msg.name))
            .append('<span class="label secondary date">' + ago(msg.timestamp) + '</span>');
        container.find('.blockquote-content').html(twemoji.parse(msg.message));
        container.find('.upvotes span').text(msg.upvotes);
        if (msg.giphy) {
            container.find('.giphy').append('<img src="' + msg.giphy + '" align="right"/>');
        }

        if ($('#' + msg.timestamp).length) continue;

        if (append) {
            $('#messages').append(container);
        } else {
            $('#messages').prepend(container);
        }

    }
    var all = $('#messages .blockquote-container').sort(function (a, b) {
        return parseInt($(b).attr('id')) - parseInt($(a).attr('id'))
    });
    $('#messages').html(all)
}

function ago(previous) {

    var msPerMinute = 60 * 1000;
    var msPerHour = msPerMinute * 60;
    var msPerDay = msPerHour * 24;
    var dateOpts = {
        day: 'numeric',
        hour: 'numeric',
        minute: 'numeric',
        month: 'numeric',
        year: 'numeric'

    };

    var elapsed = Date.now() - previous;
    if (elapsed < 5000) return 'just now';

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

function formatVotes(votes) {
    var number = parseInt(votes);
    if (number >= 1000000) return Math.floor(number / 1000000) + 'M';
    if (number >= 1000) return Math.floor(number / 1000) + 'k';
    return number

}