import { describe, it, before, after } from 'mocha'
import { expect } from 'chai'
import { ApiClient, insertDummyAccount, sql } from '../../lib/api.js'

// Pins down SubmitPost's behaviour for each kind of submission. Posts go to the
// built-in dummy connector, which never contacts a real network.
describe('api: SubmitPost', function () {
  let api
  let plainAccountId
  let approvalAccountId

  before(async function () {
    await runner.start('api')

    api = new ApiClient(runner.baseUrl())
    await api.waitUntilReady()
    await api.login('admin', 'admin')

    const [[adminId]] = await sql("SELECT id FROM user_accounts WHERE username = 'admin'")
    const suffix = Date.now()
    plainAccountId = await insertDummyAccount(`it-plain-${suffix}`)
    approvalAccountId = await insertDummyAccount(`it-approval-${suffix}`)

    const policy = await api.call('CreateAccountPolicy', {
      name: `it-approval-${suffix}`,
      applyToMcp: true,
      applyToUi: true,
      stages: [{ stageOrder: 0, userId: Number(adminId) }],
      socialAccountIds: [approvalAccountId]
    })
    expect(policy.status, JSON.stringify(policy.body)).to.equal(200)
  })

  after(async function () {
    await runner.stop()
  })

  const submit = (msg) => api.call('SubmitPost', msg)

  it('saves a draft with no accounts', async function () {
    const { status, body } = await submit({ content: 'draft only', saveAsDraft: true })
    expect(status).to.equal(200)
    expect(body.posts).to.have.length(1)
    expect(body.posts[0]).to.include({ success: true, state: 'draft', content: 'draft only' })
    expect(body.posts[0].id).to.be.greaterThan(0)
  })

  it('rejects a draft with no content or media', async function () {
    const { status, body } = await submit({ saveAsDraft: true })
    expect(status).to.equal(400)
    expect(body.code).to.equal('invalid_argument')
  })

  it('saves a draft for an account', async function () {
    const { body } = await submit({ content: 'account draft', saveAsDraft: true, socialAccounts: [plainAccountId] })
    expect(body.posts[0]).to.include({ success: true, state: 'draft', socialAccountId: plainAccountId })
    expect(body.posts[0].socialAccountIdentity).to.match(/^it-plain-/)
  })

  it('posts immediately', async function () {
    const { body } = await submit({ content: 'posted now', socialAccounts: [plainAccountId] })
    expect(body.posts[0]).to.include({ success: true, state: 'completed', socialAccountId: plainAccountId })
    expect(body.posts[0].postUrl).to.match(/^https:\/\/dummy\.local\/posts\//)
    expect(body.posts[0].id).to.be.greaterThan(0)
  })

  it('schedules a post for later', async function () {
    const later = new Date(Date.now() + 60 * 60 * 1000).toISOString()
    const { body } = await submit({ content: 'later', socialAccounts: [plainAccountId], scheduledAt: later })
    expect(body.posts[0]).to.include({ success: true, state: 'scheduled' })
    expect(body.posts[0].id).to.be.greaterThan(0)

    const [[state, scheduledAt]] = await sql(`SELECT state, scheduled_at FROM posts WHERE id = ${body.posts[0].id}`)
    expect(state).to.equal('scheduled')
    expect(scheduledAt).to.not.equal('NULL')
  })

  it('reports an error for an unknown account', async function () {
    const { status, body } = await submit({ content: 'nowhere', socialAccounts: [999999] })
    expect(status).to.equal(200)
    expect(body.posts[0]).to.include({ state: 'error', socialAccountId: 999999 })
    expect(body.posts[0].success).to.not.equal(true)
  })

  it('holds a post for approval when a policy applies', async function () {
    const { body } = await submit({ content: 'needs approval', socialAccounts: [approvalAccountId] })
    expect(body.posts[0]).to.include({ success: true, state: 'pending_approval' })

    const [[state]] = await sql(`SELECT state FROM posts WHERE id = ${body.posts[0].id}`)
    expect(state).to.equal('pending_approval')
  })

  it('rejects an unknown source draft', async function () {
    const { status, body } = await submit({ content: 'x', socialAccounts: [plainAccountId], fromDraftPostId: 999999 })
    expect(status).to.equal(400)
    expect(body.code).to.equal('invalid_argument')
  })

  it('rejects a source draft that is not a draft', async function () {
    const posted = await submit({ content: 'not a draft', socialAccounts: [plainAccountId] })
    const { status, body } = await submit({ content: 'x', socialAccounts: [plainAccountId], fromDraftPostId: posted.body.posts[0].id })
    expect(status).to.equal(400)
    expect(body.code).to.equal('invalid_argument')
  })
})
