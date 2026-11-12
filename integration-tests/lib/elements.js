import { By, Condition } from 'selenium-webdriver'
import fs from 'node:fs'
import { expect } from 'chai'

export const DEFAULT_UI_WAIT_MS = 10000

export function takeScreenshotOnFailure (test, driver) {
  if (test.state === 'failed') {
    const title = test.fullTitle()

    console.log(`Test failed, taking screenshot: ${title}`)
    takeScreenshot(driver, title)
  }
}

export function takeScreenshot (driver, title) {
  return driver.takeScreenshot().then((img) => {
    fs.mkdirSync('screenshots', { recursive: true })

    const safeTitle = title
      .replaceAll('config: ', '')
      .replaceAll(/[()|*<>:"]/g, '_')

    fs.writeFileSync(`screenshots/${safeTitle}.failed-test.png`, img, 'base64')
  })
}

export async function waitForAppReady (timeoutMs = DEFAULT_UI_WAIT_MS) {
  await webdriver.wait(new Condition('wait for loaded-app', async () => {
    const body = await webdriver.findElement(By.tagName('body'))
    const attr = await body.getAttribute('loaded-app')

    return attr != null && attr !== ''
  }), timeoutMs)
}

export async function waitForLoggedIn (timeoutMs = DEFAULT_UI_WAIT_MS) {
  await webdriver.wait(new Condition('wait for logged-in', async () => {
    const body = await webdriver.findElement(By.tagName('body'))
    const attr = await body.getAttribute('logged-in')

    return attr === 'true'
  }), timeoutMs)
}

export async function waitForLoginForm (timeoutMs = DEFAULT_UI_WAIT_MS) {
  await webdriver.wait(new Condition('wait for login form', async () => {
    const fields = await webdriver.findElements(By.css('#username'))
    return fields.length > 0
  }), timeoutMs)
}

export async function openApp () {
  await webdriver.get(runner.baseUrl())
  await waitForAppReady()
}

export async function loginAsAdmin () {
  await openApp()
  await waitForLoginForm()

  const username = await webdriver.findElement(By.css('#username'))
  const password = await webdriver.findElement(By.css('#password'))

  await username.clear()
  await username.sendKeys('admin')
  await password.clear()
  await password.sendKeys('admin')

  const submit = await webdriver.findElement(By.css('.local-login-form button[type="submit"]'))
  await submit.click()

  await waitForLoggedIn()
}

export async function getVisibleUsername () {
  const el = await webdriver.findElement(By.css('header .user-info span'))
  return el.getText()
}

export async function requireFooterLink (hrefSubstring) {
  const links = await webdriver.findElements(By.css('footer a'))
  const hrefs = await Promise.all(links.map((link) => link.getAttribute('href')))

  expect(hrefs.some((href) => href != null && href.includes(hrefSubstring))).to.equal(
    true,
    `expected footer link containing ${hrefSubstring}`
  )
}

export async function waitForElement (css, timeoutMs = DEFAULT_UI_WAIT_MS) {
  await webdriver.wait(new Condition(`wait for ${css}`, async () => {
    const found = await webdriver.findElements(By.css(css))
    return found.length > 0
  }), timeoutMs)
}

export async function openPage (pathname) {
  await webdriver.get(new URL(pathname, runner.baseUrl()).toString())
  await waitForAppReady()
  await waitForLoggedIn()
}

export async function getElementRect (css) {
  return webdriver.executeScript((selector) => {
    const rect = document.querySelector(selector).getBoundingClientRect()
    return { top: rect.top, bottom: rect.bottom, height: rect.height }
  }, css)
}

export async function getViewportAndDocumentHeight () {
  return webdriver.executeScript(() => ({
    viewport: window.innerHeight,
    document: document.documentElement.scrollHeight
  }))
}

/*
 * The seeded server never reports status messages or impersonation, so the
 * GetStatus response is rewritten in the browser. Returns the CDP script id
 * for removeGetStatusOverride.
 */
export async function overrideGetStatus (fields) {
  const source = `
    (() => {
      const extra = ${JSON.stringify(fields)};
      const originalFetch = window.fetch;
      window.fetch = async (input, init) => {
        const res = await originalFetch(input, init);
        const url = typeof input === 'string' ? input : input.url;
        if (!url.endsWith('/GetStatus')) return res;
        const body = Object.assign(await res.clone().json(), extra);
        return new Response(JSON.stringify(body), { status: res.status, headers: res.headers });
      };
    })();
  `
  const result = await webdriver.sendAndGetDevToolsCommand('Page.addScriptToEvaluateOnNewDocument', { source })
  return result.identifier
}

export async function removeGetStatusOverride (identifier) {
  await webdriver.sendDevToolsCommand('Page.removeScriptToEvaluateOnNewDocument', { identifier })
}
