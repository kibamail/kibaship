import { Server } from 'socket.io'
import server from '@adonisjs/core/services/server'
import env from '#start/env'

class Ws {
    public io: Server | undefined
    private booted = false

    public boot() {
        /**
         * Ignore multiple calls to the boot method
         */
        if (this.booted) {
            return
        }

        this.booted = true
        this.io = new Server(server.getNodeServer(), {
            cors: {
                origin: env.get('APP_URL'),
            },
        })
    }
}

export const ws = new Ws()
