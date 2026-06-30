import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import * as process from 'node:process'
import { spawn, execFile } from 'node:child_process'
import { promisify } from 'node:util'
import waitOn from 'wait-on'

const execFileAsync = promisify(execFile)

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = path.resolve(__dirname, '..')
const BINARY_PATH = path.join(REPO_ROOT, 'service', 'japella')
const RUNTIME_HOME = path.join(__dirname, '.runtime', 'home')
const DEFAULT_PORT = 18080

export default function getRunner () {
  const type = process.env.JAPELLA_TEST_RUNNER

  console.log('JAPELLA_TEST_RUNNER env value is:', type)

  switch (type) {
    case 'container':
      return new JapellaTestRunnerEnv()
    default:
      return new JapellaTestRunnerStartLocalProcess()
  }
}

export async function startDatabase () {
  console.log('Starting integration-test MariaDB via docker compose...')

  await execFileAsync('docker', ['compose', 'up', '-d', '--wait'], {
    cwd: __dirname
  })

  console.log('MariaDB is ready')
}

export async function stopDatabase () {
  console.log('Stopping integration-test MariaDB...')

  await execFileAsync('docker', ['compose', 'down', '-v'], {
    cwd: __dirname
  })
}

class JapellaTestRunner {
  BASE_URL = 'http://localhost:18080/'

  baseUrl () {
    return this.BASE_URL
  }

  healthUrl () {
    return new URL('healthz', this.baseUrl()).toString()
  }
}

class JapellaTestRunnerStartLocalProcess extends JapellaTestRunner {
  async start (cfg) {
    if (this.proc != null && this.proc.exitCode == null) {
      await this.stop()
    }

    if (!fs.existsSync(BINARY_PATH)) {
      throw new Error(`Japella binary not found at ${BINARY_PATH}. Run "make service" from the repository root first.`)
    }

    const frontendDist = path.join(REPO_ROOT, 'frontend', 'dist')
    if (!fs.existsSync(path.join(frontendDist, 'index.html'))) {
      throw new Error(`Frontend build not found at ${frontendDist}. Run "make frontend" from the repository root first.`)
    }

    const configSource = path.join(__dirname, 'tests', cfg, 'config.yaml')
    const configTargetDir = path.join(RUNTIME_HOME, '.config', 'japella')
    const configTarget = path.join(configTargetDir, 'config.yaml')

    fs.mkdirSync(configTargetDir, { recursive: true })
    fs.copyFileSync(configSource, configTarget)

    let stdout = ''
    let stderr = ''

    console.log('      Japella starting local process...')

    const env = {
      ...process.env,
      HOME: RUNTIME_HOME,
      JAPELLA_SECURE_COOKIES: 'false',
      JAPELLA_RESET_ADMIN_PASSWORD: 'true',
      JAPELLA_DB_HOST: '127.0.0.1',
      JAPELLA_DB_PORT: '3307',
      JAPELLA_DB_USER: 'japella',
      JAPELLA_DB_PASS: 'password',
      JAPELLA_DB_NAME: 'japella'
    }

    this.proc = spawn(BINARY_PATH, [], {
      cwd: __dirname,
      env
    })

    const logStdout = process.env.CI === 'true' || process.env.JAPELLA_TEST_RUNNER_LOG_STDOUT === '1'

    this.proc.stdout.on('data', (data) => {
      stdout += data
      if (logStdout) {
        console.log(`stdout: ${data}`)
      }
    })

    this.proc.stderr.on('data', (data) => {
      stderr += data
      if (logStdout) {
        console.log(`stderr: ${data}`)
      }
    })

    this.proc.on('close', (code) => {
      if (code != null) {
        console.log(`Japella local process exited with code ${code}`)
        if (stdout) {
          console.log(stdout)
        }
        if (stderr) {
          console.log(stderr)
        }
      }
    })

    this.BASE_URL = `http://localhost:${DEFAULT_PORT}/`

    console.log('      Japella waiting for local process to start...')

    await waitOn({
      resources: [this.healthUrl(), this.baseUrl()],
      timeout: 120000,
      interval: 500,
      tcpTimeout: 1000
    })

    console.log('      Japella local process started and web UI accessible')
  }

  async stop () {
    if (this.proc == null) {
      return
    }

    if (this.proc.exitCode != null) {
      console.log('      Japella local process tried stop(), but it already exited with code', this.proc.exitCode)
    } else {
      const stopTimeoutMs = 5000
      const closed = new Promise((resolve) => {
        this.proc.once('close', resolve)
      })

      this.proc.kill('SIGTERM')

      const didStopGracefully = await Promise.race([
        closed.then(() => true),
        new Promise((resolve) => setTimeout(() => resolve(false), stopTimeoutMs))
      ])

      if (!didStopGracefully) {
        console.log('      Japella local process did not exit after SIGTERM, sending SIGKILL')
        if (this.proc.exitCode == null) {
          this.proc.kill('SIGKILL')
        }
        await Promise.race([
          closed,
          new Promise((resolve) => setTimeout(resolve, stopTimeoutMs))
        ])
      }

      console.log('      Japella local process stopped')
    }

    this.proc = null

    if (process.env.CI === 'true') {
      await new Promise((resolve) => setTimeout(resolve, 3000))
    } else {
      await new Promise((resolve) => setTimeout(resolve, 100))
    }
  }
}

class JapellaTestRunnerEnv extends JapellaTestRunner {
  constructor () {
    super()

    const ip = process.env.IP || '127.0.0.1'
    const port = process.env.PORT || String(DEFAULT_PORT)

    this.BASE_URL = `http://${ip}:${port}/`
    console.log('Runner ENV endpoint:', this.BASE_URL)
  }

  async start () {
    await waitOn({
      resources: [this.healthUrl(), this.baseUrl()],
      timeout: 120000
    })
  }

  async stop () {
  }
}
