import { describe, it, before, after } from 'mocha'
import { expect } from 'chai'
import { ApiClient, insertDummyAccount, sql } from '../../lib/api.js'

// Pins down GetFeedColumnPosts. Each test posts a unique token through the
// dummy connector, which adds it to the simulated network, then searches for it.
describe('api: GetFeedColumnPosts', function () {
  let api
  let accountA
  let accountB
  let n = 0

  before(async function () {
    await runner.start('api')
    api = new ApiClient(runner.baseUrl())
    await api.waitUntilReady()
    await api.login('admin', 'admin')
    const suffix = Date.now()
    accountA = await insertDummyAccount(`it-feed-a-${suffix}`)
    accountB = await insertDummyAccount(`it-feed-b-${suffix}`)
  })

  after(async function () {
    await runner.stop()
  })

  // publish posts a unique token from account A and returns it.
  const publish = async () => {
    const token = `itfeed${Date.now()}x${n++}`
    const { body } = await api.call('SubmitPost', { content: `hello ${token}`, socialAccounts: [accountA] })
    expect(body.posts[0].state).to.equal('completed')
    return token
  }

  const createColumn = async (query, accounts = [accountA, accountB]) => {
    const { status, body } = await api.call('CreateFeedColumn', { query, socialAccountIds: accounts })
    expect(status, JSON.stringify(body)).to.equal(200)
    return body.column.id
  }

  const columnPosts = async (id, client = api) => {
    const { status, body } = await client.call('GetFeedColumnPosts', { id })
    return { status, body }
  }

  it('searches every account in the column and merges the results', async function () {
    const token = await publish()
    const { status, body } = await columnPosts(await createColumn(token))

    expect(status).to.equal(200)
    expect(body).to.not.have.property('error')
    expect(Date.parse(body.fetchedAt)).to.not.be.NaN
    expect(body.posts).to.be.an('array').and.not.empty
    for (const post of body.posts) {
      expect(post.content).to.include(token)
      expect([accountA, accountB]).to.include(post.socialAccountId)
      expect(post.remoteUrl).to.match(/^https:\/\/dummy\.local\/posts\//)
      expect(post.socialAccountIcon).to.equal('mdi:flask-outline')
    }
  })

  it('leaves out posts the user has marked read', async function () {
    const token = await publish()
    const id = await createColumn(token)
    const before = (await columnPosts(id)).body.posts
    expect(before).to.not.be.empty

    const read = before[0]
    await api.call('MarkFeedPostRead', { remoteUrl: read.remoteUrl, remoteId: read.remoteId })

    const after = (await columnPosts(id)).body.posts || []
    expect(after.map((p) => p.remoteId)).to.not.include(read.remoteId)
  })

  it('shows the tags the user gave a post', async function () {
    const token = await publish()
    const id = await createColumn(token)
    const post = (await columnPosts(id)).body.posts[0]
    const tag = `it${n++}`

    const set = await api.call('SetPostTags', {
      post: { socialAccountId: post.socialAccountId, remoteId: post.remoteId, remoteUrl: post.remoteUrl },
      snapshot: post,
      tags: [tag]
    })
    expect(set.status, JSON.stringify(set.body)).to.equal(200)

    const tagged = (await columnPosts(id)).body.posts.find((p) => p.remoteId === post.remoteId)
    expect(tagged.tags).to.deep.equal([tag])

    const tagColumn = await api.call('CreateFeedColumn', { query: `tag:${tag}` })
    expect(tagColumn.status, JSON.stringify(tagColumn.body)).to.equal(200)
    const byTag = (await columnPosts(tagColumn.body.column.id)).body
    expect(byTag.posts.map((p) => p.remoteId)).to.include(post.remoteId)
  })

  it('reports a stored query that no longer parses', async function () {
    const id = await createColumn(await publish())
    await sql(`UPDATE feed_columns SET query = '' WHERE id = ${id}`)

    const { status, body } = await columnPosts(id)
    expect(status).to.equal(200)
    expect(body.error).to.be.a('string').and.not.empty
    expect(body).to.not.have.property('posts')
  })

  it('reports a column whose accounts are all gone', async function () {
    const lonely = await insertDummyAccount(`it-feed-gone-${Date.now()}`)
    const id = await createColumn(await publish(), [lonely])
    await sql(`DELETE FROM social_accounts WHERE id = ${lonely}`)

    const { status, body } = await columnPosts(id)
    expect(status).to.equal(200)
    expect(body.error).to.equal("None of this column's accounts are available to you any more. Edit the column to choose another.")
  })

  it("refuses another user's column", async function () {
    const id = await createColumn(await publish())
    const username = `it-feed-user-${Date.now()}`
    await api.call('CreateUser', { username, password: 'password-1234' })
    const other = new ApiClient(runner.baseUrl())
    await other.login(username, 'password-1234')

    const { status } = await columnPosts(id, other)
    expect(status).to.not.equal(200)
  })
})
