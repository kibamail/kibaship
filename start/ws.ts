import { ws } from "#services/websockets/Ws"
import { ClusterLogsHandler } from '#services/websockets/cluster_logs_handler'
import app from "@adonisjs/core/services/app"
import logger from "@adonisjs/core/services/logger"
import redis from "@adonisjs/redis/services/main"

const clusterLogsHandler = new ClusterLogsHandler()

/**
 * Listen for incoming socket connections
 */
app.ready(() => {
    ws.boot()

    ws.io?.on('connection', (socket) => {
        redis.subscribe('cluster:updated', async (data) => {
            const payload = JSON.parse(data) as { id: string }

            const emit = socket.emit(`cluster:${payload.id}:updated`, payload)

            logger.info(`Emitted cluster update for ${payload.id}:`, emit)
        })

        socket.on('cluster:logs', (data) => {
            logger.info(`Received subscription request for cluster logs:`, data)
            clusterLogsHandler.onSubscribe(socket, data)
        })

        socket.on('cluster:logs:unsubscribe', (data) => {
            clusterLogsHandler.onUnsubscribe(socket, data)
        })

        socket.on('disconnect', () => {
            clusterLogsHandler.onDisconnect(socket)
        })
    })

})