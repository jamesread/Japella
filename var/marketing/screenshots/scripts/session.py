"""Shared setup: logs in as the throwaway admin and pins the picocrank theme.

repo-helper reuses one browser session for every capture, so cookies and
localStorage carry over between sections.
"""
import time

LOGIN_JS = """
const done = arguments[arguments.length - 1];
fetch('/api/japella.controlapi.v1.JapellaControlApiService/LoginWithUsernameAndPassword', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({username: 'admin', password: 'admin'}),
}).then(r => done(r.status), e => done(String(e)));
"""


def ensure_login(driver):
    if any(c["name"] == "japella-sid" for c in driver.get_cookies()):
        return False
    driver.execute_async_script(LOGIN_JS)
    return True


def prepare_session(driver, theme):
    # The URL repo-helper loaded, before any SPA redirect to the login page.
    target = driver.execute_script("return performance.getEntriesByType('navigation')[0].name;")
    reload_needed = ensure_login(driver)
    current = driver.execute_script("return localStorage.getItem('picocrank-theme');")
    if current != theme:
        driver.execute_script("localStorage.setItem('picocrank-theme', arguments[0]);", theme)
        reload_needed = True
    if reload_needed:
        # The SPA routes to Welcome after login, so go back to the target.
        driver.get(target)
        time.sleep(3)
