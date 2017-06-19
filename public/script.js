$(function () {
  var socket = createWebSocket(location.pathname + 'ws');

  socket.onmessage = function (event) {
    var msg = JSON.parse(event.data);
    console.log(msg);
    $('#messages').append(
        $('<div/>', { text: msg.name }).append($('<p/>').html(msg.message))
    )
  }
  $('#send').on('click', function () {
    socket.send(JSON.stringify({ name: $('#name').val(), message: $('#message').val() }));
  })
});

function createWebSocket(path) {
  var protocolPrefix = (window.location.protocol === 'https:') ? 'wss:' : 'ws:';
  return new WebSocket(protocolPrefix + '//' + location.host + path);
}