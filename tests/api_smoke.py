"""API smoke — Mock 模式，预期成本 ¥0。"""
import json
import os
import urllib.error
import urllib.parse
import urllib.request

API = os.environ.get("API_URL", "http://127.0.0.1:19282").rstrip("/")
USER = os.environ.get("DEMO_USER", "admin")
PASS = os.environ.get("DEMO_PASS", "gorag123")


def req(method, path, body=None, token=None, timeout=30):
    data = None
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if body is not None:
        data = json.dumps(body).encode("utf-8")
    r = urllib.request.Request(API + path, data=data, headers=headers, method=method)
    with urllib.request.urlopen(r, timeout=timeout) as resp:
        raw = resp.read().decode("utf-8")
        return resp.status, json.loads(raw)


def test_healthz():
    st, env = req("GET", "/healthz")
    assert st == 200
    assert env["code"] == 0


def test_login_search_rag_flush():
    st, env = req("POST", "/api/v1/auth/login", {"username": USER, "password": PASS})
    assert st == 200
    token = env["data"]["token"]
    q = urllib.parse.quote("向量检索")
    st, env = req(
        "POST",
        "/api/v1/search/text",
        {"query": "向量检索", "top_k": 5, "compare_flat": True},
        token=token,
    )
    assert st == 200, env
    assert env["data"]["hits"] is not None
    st, env = req("GET", "/api/v1/stats", token=token)
    assert st == 200
    assert env["data"]["cost_cny"] == 0
    st, env = req("POST", "/api/v1/admin/flush", {}, token=token)
    assert st == 200
    # rag SSE：只验证入口不是 401/500
    headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}
    body = json.dumps({"question": "什么是混合检索", "top_k": 3}).encode("utf-8")
    r = urllib.request.Request(API + "/api/v1/rag/query", data=body, headers=headers, method="POST")
    with urllib.request.urlopen(r, timeout=30) as resp:
        chunk = resp.read(64)
        assert resp.status == 200
        assert chunk  # 有流式输出
    _ = q
