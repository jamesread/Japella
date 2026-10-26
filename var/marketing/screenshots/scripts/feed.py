"""Waits for the feed's search columns to load their posts."""
import time

from selenium.webdriver.common.by import By


def run(driver):
    deadline = time.time() + 20
    while time.time() < deadline:
        columns = driver.find_elements(By.CSS_SELECTOR, "[data-feed-column-id]")
        if columns and all(c.find_elements(By.CSS_SELECTOR, ".feed-column-post") for c in columns):
            break
        time.sleep(0.25)
    time.sleep(1)
