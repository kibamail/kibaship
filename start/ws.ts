import { ws } from "#services/websockets/Ws"

ws.boot()

/**
 * Listen for incoming socket connections
 */
ws.io?.on('connection', (socket) => {
    socket.emit('news', { hello: 'world' })

    socket.on('my other event', (data) => {
        console.log(data)
    })
})
