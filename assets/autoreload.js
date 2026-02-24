const socket = new WebSocket("ws://" + location.host + "{{.WEBSOCKET_PATH}}")
socket.onclose = () => setTimeout(() => location.reload(true),{{.TIMEOUT}})