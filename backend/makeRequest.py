import requests

r = requests.post(
    "http://localhost:8080/shorten",
    json={"shorturl": "12as12", "longurl": "https://bing.com"},
)

print(r.status_code)
print(r.json())
