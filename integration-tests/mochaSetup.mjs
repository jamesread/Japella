import { Options } from 'selenium-webdriver/chrome.js'
import { Builder, Browser } from 'selenium-webdriver'
import getRunner, { startDatabase, stopDatabase } from './runner.mjs'

export async function mochaGlobalSetup () {
  await startDatabase()

  const options = new Options()
  options.addArguments('--headless=new')
  options.addArguments('--no-sandbox')
  options.addArguments('--disable-dev-shm-usage')

  global.webdriver = await new Builder().forBrowser(Browser.CHROME).setChromeOptions(options).build()

  global.runner = getRunner()

  console.log('Runner constructor:', global.runner.constructor.name)
}

export async function mochaGlobalTeardown () {
  if (global.webdriver) {
    await global.webdriver.quit()
  }

  await stopDatabase()
}
