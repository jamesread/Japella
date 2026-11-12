import { describe, it, before, after, afterEach } from 'mocha'
import { expect } from 'chai'
import { By } from 'selenium-webdriver'
import {
  openApp,
  loginAsAdmin,
  getVisibleUsername,
  getElementRect,
  getViewportAndDocumentHeight,
  openPage,
  overrideGetStatus,
  removeGetStatusOverride,
  waitForElement,
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

  describe('app shell layout', function () {
    it('sidebar sits directly below the header on a page shorter than the viewport', async function () {
      await openPage('feed')

      const header = await getElementRect('body > #app header')
      const sidebar = await getElementRect('.sidebar')
      const page = await getViewportAndDocumentHeight()

      expect(sidebar.top).to.be.closeTo(header.bottom, 1)
      expect(page.document).to.equal(page.viewport)
    })

    describe('with banners', function () {
      let overrideId

      before(async function () {
        overrideId = await overrideGetStatus({
          isImpersonating: true,
          impersonatorUsername: 'integration-tester',
          statusMessages: [
            { type: 'error', message: 'Integration test error banner' },
            { type: 'warning', message: 'Integration test warning banner' }
          ]
        })
      })

      after(async function () {
        await removeGetStatusOverride(overrideId)
      })

      it('shows impersonation and server messages below the header', async function () {
        await openPage('welcome')
        await waitForElement('.app-shell-banners .notification')

        const banners = await webdriver.findElements(By.css('.app-shell-banners > .notification'))
        const texts = await Promise.all(banners.map((banner) => banner.getText()))

        expect(texts).to.have.length(3)
        expect(texts[0]).to.include('Impersonating')
        expect(texts[0]).to.include('integration-tester')
        expect(texts[1]).to.include('Integration test error banner')
        expect(texts[2]).to.include('Integration test warning banner')

        const exit = await webdriver.findElements(By.css('.app-shell-banners .impersonation-exit'))
        expect(exit).to.have.length(1)

        const header = await getElementRect('body > #app header')
        const bannerArea = await getElementRect('.app-shell-banners')
        expect(bannerArea.top).to.be.closeTo(header.bottom, 1)
      })

      it('feed columns still fit the viewport', async function () {
        await openPage('feed')
        await waitForElement('.app-shell-banners .notification')
        await waitForElement('.feed-layout')

        const page = await getViewportAndDocumentHeight()
        expect(page.document).to.equal(page.viewport)
      })
    })
  })
})
