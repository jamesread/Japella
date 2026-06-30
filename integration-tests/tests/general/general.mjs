import { describe, it, before, after, afterEach } from 'mocha'
import { expect } from 'chai'
import { By } from 'selenium-webdriver'
import {
  openApp,
  loginAsAdmin,
  getVisibleUsername,
  requireFooterLink,
  takeScreenshotOnFailure,
  waitForLoginForm,
} from '../../lib/elements.js'

describe('config: general', function () {
  before(async function () {
    await runner.start('general')
  })

  after(async function () {
    await runner.stop()
  })

  afterEach(function () {
    takeScreenshotOnFailure(this.currentTest, webdriver)
  })

  it('health endpoint responds', async function () {
    await webdriver.get(runner.healthUrl())
    const body = await webdriver.findElement(By.tagName('body')).getText()
    expect(body).to.equal('healthy')
  })

  it('web UI loads and shows the login form', async function () {
    await openApp()

    const title = await webdriver.getTitle()
    expect(title.length).to.be.greaterThan(0)

    await waitForLoginForm()

    const username = await webdriver.findElement(By.css('#username'))
    const password = await webdriver.findElement(By.css('#password'))

    expect(await username.isDisplayed()).to.equal(true)
    expect(await password.isDisplayed()).to.equal(true)
  })

  it('footer contains documentation links', async function () {
    await openApp()

    await requireFooterLink('github.com/jamesread/Japella')
    await requireFooterLink('jamesread.github.io/Japella')
  })

  it('admin can log in with default credentials', async function () {
    await loginAsAdmin()

    const username = await getVisibleUsername()
    expect(username).to.equal('admin')
  })
})
