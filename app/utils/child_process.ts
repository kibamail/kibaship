import { execa, type Options as ExecaOptions } from 'execa'

export type StdoutCallback = (data: string) => void
export type StderrCallback = (data: string) => void
export type CloseCallback = (code: number | null) => void
export type ErrorCallback = (error: Error) => void

/**
 * A fluent interface wrapper around execa for executing child processes
 * 
 * Usage:
 * ```typescript
 * await new ChildProcess()
 *   .command('terraform')
 *   .args(['init', '-backend-config=bucket=my-bucket'])
 *   .cwd('/path/to/working/directory')
 *   .env({ AWS_ACCESS_KEY_ID: 'key' })
 *   .onStdout((data) => console.log('OUT:', data))
 *   .onStderr((data) => console.log('ERR:', data))
 *   .onClose((code) => console.log('Exit code:', code))
 *   .onError((error) => console.error('Process error:', error))
 *   .execute()
 * ```
 */
export class ChildProcess {
  private _command: string = ''
  private _args: string[] = []
  private _options: ExecaOptions = {}
  private _onStdout?: StdoutCallback
  private _onStderr?: StderrCallback
  private _onClose?: CloseCallback
  private _onError?: ErrorCallback

  /**
   * Set the command to execute
   */
  command(cmd: string): this {
    this._command = cmd
    return this
  }

  /**
   * Set the arguments for the command
   */
  args(args: string[]): this {
    this._args = args
    return this
  }

  /**
   * Set the working directory
   */
  cwd(directory: string): this {
    this._options = { ...this._options, cwd: directory }
    return this
  }

  /**
   * Set environment variables
   */
  env(environment: Record<string, string>): this {
    this._options = {
      ...this._options,
      env: {
        PATH: process.env.PATH,
        ...environment
      }
    }
    return this
  }

  /**
   * Set stdio configuration
   */
  stdio(stdio: 'pipe' | 'inherit' | 'ignore'): this {
    this._options = { ...this._options, stdio }
    return this
  }

  /**
   * Set timeout in milliseconds
   */
  timeout(ms: number): this {
    this._options = { ...this._options, timeout: ms }
    return this
  }

  /**
   * Set callback for stdout data
   */
  onStdout(callback: StdoutCallback): this {
    this._onStdout = callback
    return this
  }

  /**
   * Set callback for stderr data
   */
  onStderr(callback: StderrCallback): this {
    this._onStderr = callback
    return this
  }

  /**
   * Set callback for process close
   */
  onClose(callback: CloseCallback): this {
    this._onClose = callback
    return this
  }

  /**
   * Set callback for process errors
   */
  onError(callback: ErrorCallback): this {
    this._onError = callback
    return this
  }

  /**
   * Execute the child process
   */
  async execute() {
    const childProcess = execa(this._command, this._args, {
      ...this._options,
      stdio: 'pipe'
    })

    const self = this

    childProcess.stdout?.on('data', (data: Buffer) => {
      const output = data.toString()

      self._onStdout?.(output)
    })

    childProcess.stderr?.on('data', (data: Buffer) => {
      const output = data.toString()

      self._onStderr?.(output)
    })

    childProcess.on('close', (code) => {
      return self._onClose?.(code)
    })

    childProcess.on('error', (error) => {
      self._onError?.(error)
    })

    return childProcess
  }

  async executeAsync(): Promise<[{ stdout: string; stderr: string } | null, Error | null]> {
    try {
      const result = await execa(this._command, this._args, this._options)

      return [{
        stdout: result.stdout as string,
        stderr: result.stderr as string,
      }, null]
    } catch (error) {
      return [null, error]
    }
  }
}
