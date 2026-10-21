"""Picks a campaign and fills the compose box with a fictional post."""
import time

from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import Select

CONTENT = (
    "Fresh roast day at Acme Roasters! The Autumn Blend is back on the shelf: "
    "baked apple, hazelnut and a little cinnamon. Find it in all three cafes "
    "or order online for next-day delivery."
)


def run(driver):
    deadline = time.time() + 15
    while time.time() < deadline and not driver.find_elements(By.CSS_SELECTOR, ".account-list input[type=checkbox]"):
        time.sleep(0.25)
    # Choosing a campaign selects the accounts that belong to it.
    Select(driver.find_element(By.CSS_SELECTOR, "#submit-post select")).select_by_visible_text("Autumn Blend launch")
    time.sleep(0.5)
    textarea = driver.find_element(By.ID, "post")
    textarea.clear()
    textarea.send_keys(CONTENT)
    driver.execute_script("document.activeElement.blur(); window.scrollTo(0, 0);")
