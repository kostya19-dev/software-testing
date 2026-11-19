import sys
from pathlib import Path

from playwright.sync_api import sync_playwright


def capture(url, output):
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(channel="chrome")
        page = browser.new_page(viewport={"width": 1600, "height": 1550}, device_scale_factor=1)
        page.goto(url + "/d/lab5-auth/auth?kiosk&theme=light", timeout=90000)
        page.get_by_text("Длительность сценариев", exact=True).first.wait_for(timeout=60000)
        page.locator("canvas").first.wait_for(state="visible", timeout=60000)
        page.get_by_text("Результаты сценариев", exact=True).first.wait_for()
        page.get_by_text("Результат", exact=True).first.wait_for()
        page.evaluate("document.fonts.ready")
        page.screenshot(path=output, animations="disabled")
        browser.close()


if __name__ == "__main__":
    capture(sys.argv[1], Path(sys.argv[2]))
