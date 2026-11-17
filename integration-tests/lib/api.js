import { execFile } from 'node:child_process'
import { promisify } from 'node:util'

const execFileAsync = promisify(execFile)

const SERVICE = 'japella.controlapi.v1.JapellaControlApiService'

// ApiClient calls the Connect API with JSON, keeping the session cookie.
export class ApiClient {
  constructor (baseUrl) {
    this.baseUrl = baseUrl
    this.cookie = ''
  }

  // Returns { status, body }; Connect errors come back as { code, message }.
  async call (method, body = {}) {
    const res = await fetch(new URL(`api/${SERVICE}/${method}`, this.baseUrl), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(this.cookie ? { Cookie: this.cookie } : {})
      },
      body: JSON.stringify(body)
    })
    const setCookie = res.headers.get('set-cookie')
    if (setCookie) {
      this.cookie = setCookie.split(';')[0]
    }
    const text = await res.text()
    return { status: res.status, body: text ? JSON.parse(text) : {} }
  }

  // The API answers "Service is starting" until the database and connectors are up.
  async waitUntilReady (timeoutMs = 60000) {
    const deadline = Date.now() + timeoutMs
    for (;;) {
      const res = await fetch(new URL(`api/${SERVICE}/GetStatus`, this.baseUrl), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: '{}'
      })
      if (res.ok && (await res.json()).status === 'OK!') {
        return
      }
      if (Date.now() > deadline) {
        throw new Error('Japella API did not become ready')
      }
      await new Promise((resolve) => setTimeout(resolve, 500))
    }
  }

  async login (username, password) {
    const { status, body } = await this.call('LoginWithUsernameAndPassword', { username, password })
    if (status !== 200 || !body.standardResponse?.success) {
      throw new Error(`login failed for ${username}: ${status} ${JSON.stringify(body)}`)
    }
  }
}

// sql runs statements in the integration-test MariaDB and returns the output rows.
export async function sql (statements) {
  const { stdout } = await execFileAsync('docker', [
    'exec', 'japella-it-mariadb',
    'mariadb', '-ujapella', '-ppassword', 'japella', '-N', '-e', statements
  ])
  return stdout.trim().split('\n').filter(Boolean).map((line) => line.split('\t'))
}
