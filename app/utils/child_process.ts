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
        ...process.env,
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
  async execute(): Promise<{ stdout: string; stderr: string; exitCode: number }> {
    if (!this._command) {
      throw new Error('Command must be specified before execution')
    }

    return new Promise((resolve, reject) => {
      let stdout = ''
      let stderr = ''

      const childProcess = execa(this._command, this._args, {
        ...this._options,
        stdio: 'pipe' // Always use pipe to capture output
      })

      // Handle stdout
      if (childProcess.stdout) {
        childProcess.stdout.on('data', (data: Buffer) => {
          const output = data.toString()
          stdout += output
          if (this._onStdout) {
            this._onStdout(output)
          }
        })
      }

      // Handle stderr
      if (childProcess.stderr) {
        childProcess.stderr.on('data', (data: Buffer) => {
          const output = data.toString()
          stderr += output
          if (this._onStderr) {
            this._onStderr(output)
          }
        })
      }

      // Handle process close
      childProcess.on('close', (code: number | null) => {
        if (this._onClose) {
          this._onClose(code)
        }

        if (code === 0) {
          resolve({
            stdout,
            stderr,
            exitCode: code || 0
          })
        } else {
          reject(new Error(`Process exited with code ${code}. stderr: ${stderr}`))
        }
      })

      // Handle process errors
      childProcess.on('error', (error: Error) => {
        if (this._onError) {
          this._onError(error)
        }
        reject(error)
      })
    })
  }

  /**
   * Execute the process and return the execa child process for advanced usage
   */
  spawn() {
    if (!this._command) {
      throw new Error('Command must be specified before spawning')
    }

    const childProcess = execa(this._command, this._args, this._options)

    // Attach event listeners if provided
    if (this._onStdout && childProcess.stdout) {
      childProcess.stdout.on('data', (data: Buffer) => {
        this._onStdout!(data.toString())
      })
    }

    if (this._onStderr && childProcess.stderr) {
      childProcess.stderr.on('data', (data: Buffer) => {
        this._onStderr!(data.toString())
      })
    }

    if (this._onClose) {
      childProcess.on('close', this._onClose)
    }

    if (this._onError) {
      childProcess.on('error', this._onError)
    }

    return childProcess
  }

  /**
   * Static method for quick execution without fluent interface
   */
  static async run(
    command: string,
    args: string[] = [],
    options: ExecaOptions = {}
  ): Promise<{ stdout: string; stderr: string; exitCode: number }> {
    const childProcess = new ChildProcess()
      .command(command)
      .args(args)

    if (options.env) {
      childProcess.env(options.env as Record<string, string>)
    }

    if (options.cwd) {
      childProcess.cwd(options.cwd as string)
    }

    return childProcess.execute()
  }
}
