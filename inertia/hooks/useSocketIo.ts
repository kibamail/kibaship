import { useEffect, useRef } from 'react'
import { socket as io } from '~/sockets/io'

export function useSocketIo() {
    const { current: socket } = useRef(io)

    useEffect(() => {
        return () => {
            socket && socket.removeAllListeners();
            socket && socket.close();
        };
    }, [socket]);

    return socket
}

