"""Scrolls the timeline past upcoming posts to the first published one.

The timeline loads 10 posts at a time as you scroll, newest (furthest in the
future) first, so keep scrolling until a published post has loaded.
"""
import time

SCROLL_JS = """
const link = document.querySelector('.post-preview .original-post-link');
if (!link) { window.scrollTo(0, document.scrollingElement.scrollHeight); return false; }
const card = link.closest('.post-preview').parentElement;
window.scrollTo(0, card.getBoundingClientRect().top + window.scrollY - 64);
return true;
"""


def run(driver):
    deadline = time.time() + 15
    while time.time() < deadline and not driver.execute_script(SCROLL_JS):
        time.sleep(0.5)
    time.sleep(1)
    driver.execute_script(SCROLL_JS)
