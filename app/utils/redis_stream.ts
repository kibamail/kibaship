import redis from '@adonisjs/redis/services/main'

export type StreamEntry = {
  id: string
  fields: Record<string, string>
}

export type StreamMessage = {
  stream: string
  entries: StreamEntry[]
}

export type ConsumerGroupInfo = {
  name: string
  consumers: number
  pending: number
  lastDeliveredId: string
}

export type StreamReadOptions = {
  count?: number
  block?: number
}

export type StreamAddCallback = (id: string) => void
export type StreamReadCallback = (messages: StreamMessage[]) => void
export type StreamErrorCallback = (error: Error) => void

/**
 * A fluent interface wrapper for Redis streams operations
 * 
 * Usage for adding to stream:
 * ```typescript
 * await new RedisStream()
 *   .stream('events')
 *   .fields({ event: 'user_created', userId: '123' })
 *   .onAdd((id) => console.log('Added with ID:', id))
 *   .onError((error) => console.error('Add error:', error))
 *   .add()
 * ```
 * 
 * Usage for reading from stream:
 * ```typescript
 * await new RedisStream()
 *   .stream('events')
 *   .from('0')
 *   .count(10)
 *   .onRead((messages) => console.log('Received:', messages))
 *   .onError((error) => console.error('Read error:', error))
 *   .read()
 * ```
 * 
 * Usage for consumer groups:
 * ```typescript
 * await new RedisStream()
 *   .stream('events')
 *   .group('processors')
 *   .consumer('worker-1')
 *   .count(5)
 *   .block(1000)
 *   .onRead((messages) => console.log('Received:', messages))
 *   .readGroup()
 * ```
 */
export class RedisStream {
  private _streamName: string = ''
  private _fields: Record<string, string> = {}
  private _id: string = '*'
  private _from: string = '$'
  private _groupName: string = ''

  private _count?: number
  private _block?: number
  private _onAdd?: StreamAddCallback
  private _onRead?: StreamReadCallback
  private _onError?: StreamErrorCallback

  /**
   * Set the stream name
   */
  stream(name: string): this {
    this._streamName = name
    return this
  }

  /**
   * Set fields for adding to stream
   */
  fields(fields: Record<string, string>): this {
    this._fields = { ...this._fields, ...fields }
    return this
  }

  /**
   * Set a single field for adding to stream
   */
  field(key: string, value: string): this {
    this._fields[key] = value
    return this
  }

  /**
   * Set the ID for adding to stream (default: '*' for auto-generated)
   */
  id(id: string): this {
    this._id = id
    return this
  }

  /**
   * Set the starting position for reading from stream
   */
  from(position: string): this {
    this._from = position
    return this
  }

  /**
   * Set consumer group name
   */
  group(groupName: string): this {
    this._groupName = groupName
    return this
  }

  /**
   * Set maximum number of entries to read
   */
  count(count: number): this {
    this._count = count
    return this
  }

  /**
   * Set blocking timeout in milliseconds
   */
  block(milliseconds: number): this {
    this._block = milliseconds
    return this
  }

  /**
   * Set callback for successful add operations
   */
  onAdd(callback: StreamAddCallback): this {
    this._onAdd = callback
    return this
  }

  /**
   * Set callback for successful read operations
   */
  onRead(callback: StreamReadCallback): this {
    this._onRead = callback
    return this
  }

  /**
   * Set callback for error handling
   */
  onError(callback: StreamErrorCallback): this {
    this._onError = callback
    return this
  }

  /**
   * Add entry to stream
   */
  async add(): Promise<string> {
    try {
      const fieldArray: string[] = []
      for (const [key, value] of Object.entries(this._fields)) {
        fieldArray.push(key, value)
      }

      const id = await redis.xadd(this._streamName, this._id, ...fieldArray)

      if (this._onAdd && id) {
        this._onAdd(id)
      }

      return id || ''
    } catch (error) {
      if (this._onError) {
        this._onError(error as Error)
      }
      throw error
    }
  }

  /**
   * Read entries from stream
   */
  async read(): Promise<StreamMessage[]> {
    try {
      const args: (string | number)[] = []

      if (this._count !== undefined) {
        args.push('COUNT', this._count)
      }

      if (this._block !== undefined) {
        args.push('BLOCK', this._block)
      }

      args.push('STREAMS', this._streamName, this._from)

      const result = await (redis as any).xread(...args)
      const messages = this.parseStreamResult(result)

      if (this._onRead) {
        this._onRead(messages)
      }

      return messages
    } catch (error) {
      if (this._onError) {
        this._onError(error as Error)
      }
      throw error
    }
  }

  /**
   * Create consumer group
   */
  async createGroup(startId: string = '0'): Promise<void> {
    await redis.xgroup('CREATE', this._streamName, this._groupName, startId, 'MKSTREAM')
  }

  /**
   * Acknowledge message processing
   */
  async ack(messageIds: string[]): Promise<number> {
    return redis.xack(this._streamName, this._groupName, ...messageIds)
  }

  /**
   * Get stream length
   */
  async length(): Promise<number> {
    return redis.xlen(this._streamName)
  }

  /**
   * Parse Redis stream result into structured format
   */
  private parseStreamResult(result: any): StreamMessage[] {
    if (!result || !Array.isArray(result)) {
      return []
    }

    return result.map(([streamName, entries]: [string, any[]]) => ({
      stream: streamName,
      entries: entries.map(([id, fields]: [string, string[]]) => ({
        id,
        fields: this.parseFields(fields)
      }))
    }))
  }

  /**
   * Parse field array into key-value object
   */
  private parseFields(fields: string[]): Record<string, string> {
    const result: Record<string, string> = {}
    for (let i = 0; i < fields.length; i += 2) {
      result[fields[i]] = fields[i + 1]
    }
    return result
  }

  /**
   * Static method for quick stream addition
   */
  static async add(
    streamName: string,
    fields: Record<string, string>,
    id: string = '*'
  ): Promise<string> {
    return new RedisStream()
      .stream(streamName)
      .fields(fields)
      .id(id)
      .add()
  }

  /**
   * Static method for quick stream reading
   */
  static async read(
    streamName: string,
    from: string = '0',
    options: StreamReadOptions = {}
  ): Promise<StreamMessage[]> {
    const stream = new RedisStream()
      .stream(streamName)
      .from(from)

    if (options.count !== undefined) {
      stream.count(options.count)
    }

    if (options.block !== undefined) {
      stream.block(options.block)
    }

    return stream.read()
  }
}
