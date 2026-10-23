import base64
import json
import os
import time
import urllib.parse
import urllib.request

base = "http://localhost:" + os.environ.get("JENKINS_PORT", "8092")
auth = (
    "Basic "
    + base64.b64encode(
        ("student:" + os.environ["LAB_ADMIN_PASSWORD"]).encode()
    ).decode()
)
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor())


def get(path):
    return json.load(
        opener.open(
            urllib.request.Request(base + path, headers={"Authorization": auth}),
            timeout=15,
        )
    )


crumb = get("/crumbIssuer/api/json")
headers = {"Authorization": auth, crumb["crumbRequestField"]: crumb["crumb"]}
data = urllib.parse.urlencode({"FAIL_STAGE": os.getenv("LAB2_FAIL_STAGE", "")}).encode()
job = get("/job/tiopo-lab2/api/json")
parameterized = any("parameterDefinitions" in prop for prop in job.get("property", []))
endpoint = "/buildWithParameters" if parameterized else "/build"
response = opener.open(
    urllib.request.Request(
        base + "/job/tiopo-lab2" + endpoint, data=data, headers=headers
    ),
    timeout=15,
)
queue = response.headers["Location"].replace(base, "").rstrip("/") + "/api/json"
for _ in range(180):
    item = get(queue)
    if item.get("cancelled"):
        raise SystemExit("Build cancelled")
    if "executable" in item:
        number = item["executable"]["number"]
        break
    time.sleep(1)
else:
    raise SystemExit("Queue timeout")
print(f"Jenkins build #{number}: {base}/job/tiopo-lab2/{number}/", flush=True)
for _ in range(1800):
    build = get(f"/job/tiopo-lab2/{number}/api/json")
    if not build["building"]:
        print(build["result"], flush=True)
        raise SystemExit(0 if build["result"] == "SUCCESS" else 1)
    time.sleep(1)
raise SystemExit("Build wait timeout; check Jenkins")
