import { describe, it, before, after } from 'mocha'
import { expect } from 'chai'
import { ApiClient } from '../../lib/api.js'

// GetStatus is public: it must hide system details from anyone who can't
// view diagnostics, and describe the caller's session.
describe('api: GetStatus', function () {
  before(async function () {
    await runner.start('api')
    await new ApiClient(runner.baseUrl()).waitUntilReady()
  })

  after(async function () {
    await runner.stop()
  })

  const status = async (client) => {
    const { status: code, body } = await client.call('GetStatus')
    expect(code).to.equal(200)
    return body
  }

  it('hides system details from anonymous callers', async function () {
    const body = await status(new ApiClient(runner.baseUrl()))

    expect(body).to.include({ status: 'OK!', username: '<anonymous>', databaseConnected: true })
    expect(body.isLoggedIn).to.not.equal(true)
    expect(body.version).to.be.a('string').and.not.empty
    expect(body.webhookEvents).to.include('post.completed')
    expect(body.availableThemes).to.be.an('array').and.not.empty
    for (const hidden of ['databaseHost', 'databaseName', 'listenAddress', 'configFileAbsolutePath', 'databaseSchemaVersion', 'nanoservices', 'rbacPermissions', 'createdAt', 'lastLoginAt']) {
      expect(body, hidden).to.not.have.property(hidden)
    }
  })

  it('describes an admin session, with system details', async function () {
    const admin = new ApiClient(runner.baseUrl())
    await admin.login('admin', 'admin')
    const body = await status(admin)

    expect(body).to.include({
      status: 'OK!',
      username: 'admin',
      isLoggedIn: true,
      rbacIsSuperuser: true,
      databaseHost: '127.0.0.1',
      databaseName: 'japella',
      listenAddress: '0.0.0.0:8080'
    })
    expect(body.isImpersonating).to.not.equal(true)
    expect(body.databaseSchemaVersion).to.be.greaterThan(0)
    expect(body.configFileAbsolutePath).to.match(/config\.yaml$/)
    expect(Date.parse(body.createdAt)).to.not.be.NaN
    expect(Date.parse(body.lastLoginAt)).to.not.be.NaN
  })

  it('reports impersonation', async function () {
    const admin = new ApiClient(runner.baseUrl())
    await admin.login('admin', 'admin')
    const username = `it-status-${Date.now()}`
    const created = await admin.call('CreateUser', { username, password: 'password-1234' })
    expect(created.status, JSON.stringify(created.body)).to.equal(200)

    const imp = await admin.call('ImpersonateUser', { userId: created.body.user.id })
    expect(imp.status, JSON.stringify(imp.body)).to.equal(200)

    const body = await status(admin)
    expect(body).to.include({ username, isLoggedIn: true, isImpersonating: true, impersonatorUsername: 'admin' })
    expect(body.rbacIsSuperuser).to.not.equal(true)
    expect(body, 'a non-admin sees no system details').to.not.have.property('databaseHost')
  })
})
